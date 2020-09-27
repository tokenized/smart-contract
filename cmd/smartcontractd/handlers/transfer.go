package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/messages"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Transfer struct {
	handler         protomux.Handler
	MasterDB        *db.DB
	Config          *node.Config
	Headers         node.BitcoinHeaders
	Tracer          *filters.Tracer
	Scheduler       *scheduler.Scheduler
	HoldingsChannel *holdings.CacheChannel
}

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferRequest")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	msg, ok := itx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Could not assert as *actions.Transfer")
	}

	// Find "first" contract.
	first := firstContractOutputIndex(msg.Assets, itx)

	if first == 0xffff {
		node.LogWarn(ctx, "Transfer first contract not found : %x", rk.Address.Bytes())
		return errors.New("Transfer first contract not found")
	}

	if !itx.Outputs[first].Address.Equal(rk.Address) {
		node.LogVerbose(ctx, "Not contract for first transfer. Waiting for Message SettlementRequest : %x",
			itx.Outputs[first].Address.Bytes())
		if err := transactions.AddTx(ctx, t.MasterDB, itx); err != nil {
			return errors.Wrap(err, "Failed to save tx")
		}
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	// Check pre-processing reject code
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Transfer request invalid")
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			itx.RejectCode, false, itx.RejectText)
	}

	if msg.OfferExpiry != 0 && v.Now.Nano() > msg.OfferExpiry {
		node.LogWarn(ctx, "Transfer expired : %d", msg.OfferExpiry)
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsTransferExpired, false, "Transfer expired")
	}

	if len(msg.Assets) == 0 {
		node.LogWarn(ctx, "Transfer has no asset transfers")
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsMsgMalformed, false, "No transfers")
	}

	// Bitcoin balance of first (this) contract. Funding for bitcoin transfers.
	contractBalance := itx.Outputs[first].UTXO.Value

	settlementRequest := messages.SettlementRequest{
		Timestamp:    v.Now.Nano(),
		TransferTxId: itx.Hash[:],
	}

	ct, err := contract.Retrieve(ctx, t.MasterDB, rk.Address, t.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsContractMoved, false, "Contract address changed")
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen")
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsContractFrozen, false, "Contract frozen")
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsContractExpired, false, "Contract expired")
	}

	// Transfer Outputs
	//   Contract 1 : amount = calculated fee for settlement tx + contract fees + any bitcoins being transfered
	//   Contract 2 : contract fees if applicable or dust
	//   Boomerang to Contract 1 : amount = ((n-1) * 2) * (calculated fee for data passing tx)
	//     where n is number of contracts involved
	// Boomerang is only required when more than one contract is involved.
	// It is defined as an output from the transfer tx, that pays to the first contract of the
	//   transfer, but it's index is not referenced/spent by any of the asset transfers of the
	//   transfer tx.
	// The first contract is defined by the first valid contract index of a transfer. Some of the
	//   transfers will not reference a contract, like a bitcoin transfer.
	//
	// Transfer Inputs
	//   Any addresses sending tokens or bitcoin.
	//
	// Each contract can be involved in more than one asset in the transfer, but only needs to have
	//   one output since each asset transfer references the output of it's contract
	var settleTx *txbuilder.TxBuilder
	settleTx, err = buildSettlementTx(ctx, t.MasterDB, t.Config, itx, msg, &settlementRequest,
		contractBalance, rk)
	if err != nil {
		node.LogWarn(ctx, "Failed to build settlement tx : %s", err)
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsMsgMalformed, false, err.Error())
	}

	// Create initial settlement data
	settlement := actions.Settlement{Timestamp: v.Now.Nano()}

	// Serialize empty settlement data into OP_RETURN output as a placeholder to be updated by
	//   addSettlementData.
	var script []byte
	script, err = protocol.Serialize(&settlement, t.Config.IsTest)
	if err != nil {
		node.LogWarn(ctx, "Failed to serialize settlement : %s", err)
		return err
	}
	err = settleTx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	// Add this contract's data to the settlement op return data
	isSingleContract := transferIsSingleContract(ctx, itx, msg, rk)
	assetUpdates := make(map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding)
	err = addSettlementData(ctx, t.MasterDB, t.Config, rk, itx, msg, settleTx, &settlement,
		t.Headers, &assetUpdates, isSingleContract)
	if err != nil {
		rejectCode, ok := node.ErrorCode(err)
		if ok {
			node.LogWarn(ctx, "Rejecting Transfer : %s", err)
			return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg,
				rk, rejectCode, false, "")
		} else {
			return errors.Wrap(err, "Failed to add settlement data")
		}
	}

	// Check if settlement data is complete. No other contracts involved
	if isSingleContract {
		node.Log(ctx, "Single contract settlement complete")
		if err := settleTx.Sign([]bitcoin.Key{rk.Key}); err != nil {
			if errors.Cause(err) == txbuilder.ErrInsufficientValue {
				node.LogWarn(ctx, "Insufficient settlement tx funding : %s", err)
				return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx,
					msg, rk, actions.RejectionsInsufficientTxFeeFunding, false, err.Error())
			} else {
				node.LogWarn(ctx, "Failed to sign settlement tx : %s", err)
				return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx,
					msg, rk, actions.RejectionsMsgMalformed, false, "")
			}
		}

		err := node.Respond(ctx, w, settleTx.MsgTx)
		if err == nil {
			if err = saveHoldings(ctx, t.MasterDB, t.HoldingsChannel, assetUpdates, rk.Address); err != nil {
				return err
			}
		}

		return err
	}

	// Save tx
	if err := transactions.AddTx(ctx, t.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Send to next contract
	if err := sendToNextSettlementContract(ctx, w, rk, itx, itx, msg, settleTx, &settlement,
		&settlementRequest, t.Tracer); err != nil {
		return err
	}

	// Save pending transfer
	timeout := protocol.NewTimestamp(v.Now.Nano() + t.Config.RequestTimeout)
	pendingTransfer := state.PendingTransfer{TransferTxId: protocol.TxIdFromBytes(itx.Hash[:]),
		Timeout: timeout}
	if err := transfer.Save(ctx, t.MasterDB, rk.Address, &pendingTransfer); err != nil {
		return errors.Wrap(err, "Failed to save pending transfer")
	}

	// Schedule timeout for transfer in case the other contract(s) don't respond.
	if err := t.Scheduler.ScheduleJob(ctx, listeners.NewTransferTimeout(t.handler, itx, timeout)); err != nil {
		return errors.Wrap(err, "Failed to schedule transfer timeout")
	}

	if err := saveHoldings(ctx, t.MasterDB, t.HoldingsChannel, assetUpdates, rk.Address); err != nil {
		return err
	}

	return nil
}

func saveHoldings(ctx context.Context, masterDB *db.DB, holdingsChannel *holdings.CacheChannel,
	updates map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding,
	contractAddress bitcoin.RawAddress) error {

	for assetCode, hds := range updates {
		for _, h := range *hds {
			cacheItem, err := holdings.Save(ctx, masterDB, contractAddress, &assetCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
			holdingsChannel.Add(cacheItem)
		}
	}

	return nil
}

func revertHoldings(ctx context.Context, masterDB *db.DB, holdingsChannel *holdings.CacheChannel,
	updates map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding,
	contractAddress bitcoin.RawAddress, txid *protocol.TxId) error {

	for _, hds := range updates {
		for _, h := range *hds {
			if err := holdings.RevertStatus(h, txid); err != nil {
				return errors.Wrap(err, "Failed to revert holding status")
			}
		}
	}

	return saveHoldings(ctx, masterDB, holdingsChannel, updates, contractAddress)
}

// TransferTimeout is called when a multi-contract transfer times out because the other contracts are not responding.
func (t *Transfer) TransferTimeout(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferTimeout")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Could not assert as *actions.Transfer")
	}

	// Remove pending transfer
	if err := transfer.Remove(ctx, t.MasterDB, rk.Address, protocol.TxIdFromBytes(itx.Hash[:])); err != nil {
		if err != transfer.ErrNotFound {
			return errors.Wrap(err, "Failed to remove pending transfer")
		}
	}

	node.LogWarn(ctx, "Transfer timed out")
	return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
		actions.RejectionsTimeout, true, "")
}

// firstContractOutputIndex finds the "first" contract. The "first" contract of a transfer is the one
//   responsible for creating the initial settlement data and passing it to the next contract if
//   there are more than one.
func firstContractOutputIndex(assetTransfers []*actions.AssetTransferField, itx *inspector.Transaction) uint32 {
	for _, asset := range assetTransfers {
		if asset.AssetType != "BSV" && len(asset.AssetCode) != 0 && int(asset.ContractIndex) < len(itx.Outputs) {
			return asset.ContractIndex
		}
	}

	return 0x0000ffff
}

// buildSettlementTx builds the tx for a settlement action.
func buildSettlementTx(ctx context.Context, masterDB *db.DB, config *node.Config,
	transferTx *inspector.Transaction, transfer *actions.Transfer,
	settlementRequest *messages.SettlementRequest, contractBalance uint64, rk *wallet.Key) (*txbuilder.TxBuilder, error) {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.buildSettlementTx")
	defer span.End()

	// Settle Outputs
	//   Any addresses sending or receiving tokens or bitcoin.
	//   Referenced from indices from within settlement data.
	//
	// Settle Inputs
	//   Any contracts involved.
	settleTx := txbuilder.NewTxBuilder(config.FeeRate, config.DustFeeRate)
	settleTx.SetChangeAddress(rk.Address, "")

	var err error
	addresses := make(map[bitcoin.Hash20]uint32)
	outputUsed := make([]bool, len(transferTx.Outputs))

	// Setup inputs from outputs of the Transfer tx. One from each contract involved.
	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.ContractIndex == uint32(0x0000ffff) ||
			(assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0) {
			continue
		}

		if int(assetTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return nil, fmt.Errorf("Transfer contract index out of range %d", assetOffset)
		}

		if outputUsed[assetTransfer.ContractIndex] {
			continue
		}

		// Add input from contract to settlement tx so all involved contracts have to sign for a valid tx.
		err = settleTx.AddInput(wire.OutPoint{Hash: *transferTx.Hash, Index: uint32(assetTransfer.ContractIndex)},
			transferTx.Outputs[assetTransfer.ContractIndex].UTXO.LockingScript,
			transferTx.Outputs[assetTransfer.ContractIndex].UTXO.Value)
		if err != nil {
			return nil, err
		}
		outputUsed[assetTransfer.ContractIndex] = true
	}

	// Setup outputs
	//   One to each receiver, including any bitcoins received, or dust.
	//   One to each sender with dust amount.
	for assetOffset, assetTransfer := range transfer.Assets {
		assetIsBitcoin := assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0
		assetBalance := uint64(0)

		// Add all senders
		for senderOffset, quantityIndex := range assetTransfer.AssetSenders {
			assetBalance += quantityIndex.Quantity

			if quantityIndex.Index >= uint32(len(transferTx.Inputs)) {
				return nil, fmt.Errorf("Transfer sender index out of range %d", assetOffset)
			}

			input := transferTx.Inputs[quantityIndex.Index]

			if assetIsBitcoin {
				// Check sender input's bitcoin balance.
				// Bitcoin senders don't need an output.
				if uint64(quantityIndex.Quantity) >= input.UTXO.Value {
					return nil, fmt.Errorf("Sender bitcoin quantity higher than input amount for sender %d : %d/%d",
						senderOffset, input.UTXO.Value, quantityIndex.Quantity)
				}
			} else {
				// Add "notification" dust output for settlement.
				hash, err := input.Address.Hash()
				if err != nil {
					return nil, errors.Wrap(err, "sender address hash")
				}
				_, exists := addresses[*hash]
				if !exists {
					// Add output to sender
					addresses[*hash] = uint32(len(settleTx.MsgTx.TxOut))

					err = settleTx.AddDustOutput(input.Address, false)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		var receiverAddress bitcoin.RawAddress
		for _, assetReceiver := range assetTransfer.AssetReceivers {
			if assetReceiver.Quantity > assetBalance {
				return nil, fmt.Errorf("Sending more than received")
			}

			assetBalance -= assetReceiver.Quantity

			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if assetReceiver.Quantity > contractBalance {
					return nil, fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= assetReceiver.Quantity
			}

			receiverAddress, err = bitcoin.DecodeRawAddress(assetReceiver.Address)
			if err != nil {
				return nil, err
			}
			hash, err := receiverAddress.Hash()
			if err != nil {
				return nil, errors.Wrap(err, "Transfer receiver address invalid")
			}
			outputIndex, exists := addresses[*hash]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, assetReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver's address
				addresses[*hash] = uint32(len(settleTx.MsgTx.TxOut))
				if assetIsBitcoin {
					err = settleTx.AddPaymentOutput(receiverAddress, assetReceiver.Quantity, false)
				} else {
					err = settleTx.AddDustOutput(receiverAddress, false)
				}
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Add other contract's fees
	for _, fee := range settlementRequest.ContractFees {
		feeAddress, err := bitcoin.DecodeRawAddress(fee.Address)
		if err != nil {
			return nil, err
		}
		settleTx.AddPaymentOutput(feeAddress, fee.Quantity, false)
	}

	// Add this contract's fee output
	ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
	if err != nil {
		return settleTx, errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.ContractFee > 0 {
		settleTx.AddPaymentOutput(config.FeeAddress, ct.ContractFee, false)

		// Add to settlement request
		settlementRequest.ContractFees = append(settlementRequest.ContractFees,
			&messages.TargetAddressField{Address: config.FeeAddress.Bytes(), Quantity: ct.ContractFee})
	}

	// Add exchange fee
	if len(transfer.ExchangeFeeAddress) != 0 && transfer.ExchangeFee > 0 {
		exchangeAddress, err := bitcoin.DecodeRawAddress(transfer.ExchangeFeeAddress)
		if err != nil {
			return nil, errors.Wrap(err, "decode exchange fee address")
		}

		// Find output for receiver
		added := false
		for i, _ := range settleTx.MsgTx.TxOut {
			outputAddress, err := settleTx.OutputAddress(i)
			if err != nil {
				continue
			}
			if exchangeAddress.Equal(outputAddress) {
				// Add exchange fee to existing output
				settleTx.AddValueToOutput(uint32(i), transfer.ExchangeFee)
				added = true
				break
			}
		}

		if !added {
			// Add new output for exchange fee.
			if err := settleTx.AddPaymentOutput(exchangeAddress, transfer.ExchangeFee, false); err != nil {
				return nil, errors.Wrap(err, "add exchange fee output")
			}
		}
	}

	return settleTx, nil
}

// addSettlementData appends data to a pending settlement action.
func addSettlementData(ctx context.Context, masterDB *db.DB, config *node.Config, rk *wallet.Key,
	transferTx *inspector.Transaction, transfer *actions.Transfer, settleTx *txbuilder.TxBuilder,
	settlement *actions.Settlement, headers node.BitcoinHeaders,
	updates *map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding, isSingleContract bool) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addSettlementData")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	dataAdded := false

	ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		return node.NewError(actions.RejectionsContractFrozen, "")
	}

	// Generate public key hashes for all the outputs
	transferOutputAddresses := make([]bitcoin.RawAddress, 0, len(transferTx.Outputs))
	for _, output := range transferTx.Outputs {
		transferOutputAddresses = append(transferOutputAddresses, output.Address)
	}

	// Generate public key hashes for all the inputs
	settleInputAddresses := make([]bitcoin.RawAddress, 0, len(settleTx.Inputs))
	for _, input := range settleTx.Inputs {
		address, err := bitcoin.RawAddressFromLockingScript(input.LockingScript)
		if err != nil {
			settleInputAddresses = append(settleInputAddresses, bitcoin.RawAddress{})
			continue
		}
		settleInputAddresses = append(settleInputAddresses, address)
	}

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]bitcoin.RawAddress, 0, len(settleTx.MsgTx.TxOut))
	for _, output := range settleTx.MsgTx.TxOut {
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			settleOutputAddresses = append(settleOutputAddresses, bitcoin.RawAddress{})
			continue
		}
		settleOutputAddresses = append(settleOutputAddresses, address)
	}

	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			node.LogVerbose(ctx, "Asset transfer for bitcoin")
			continue // Skip bitcoin transfers since they should be handled already
		}

		if int(assetTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return fmt.Errorf("Contract index %d out of range : %d >= %d", assetOffset,
				assetTransfer.ContractIndex, len(transferTx.Outputs))
		}

		contractOutputAddress := transferOutputAddresses[assetTransfer.ContractIndex]
		if contractOutputAddress.IsEmpty() || !contractOutputAddress.Equal(rk.Address) {
			continue // This asset is not ours. Skip it.
		}

		assetCode := protocol.AssetCodeFromBytes(assetTransfer.AssetCode)
		assetID := protocol.AssetID(assetTransfer.AssetType, *assetCode)

		// Locate Asset
		as, err := asset.Retrieve(ctx, masterDB, rk.Address, assetCode)
		if err != nil {
			return fmt.Errorf("Asset ID not found : %s : %s", assetID, err)
		}

		if err := asset.IsTransferable(ctx, as, v.Now); err != nil {
			return err
		}

		// Find contract input
		contractInputIndex := uint32(0x0000ffff)
		for i, input := range settleInputAddresses {
			if !input.IsEmpty() && input.Equal(rk.Address) {
				contractInputIndex = uint32(i)
				break
			}
		}

		if contractInputIndex == uint32(0x0000ffff) {
			return fmt.Errorf("Contract input not found: %s", assetID)
		}

		node.LogVerbose(ctx, "Adding settlement data for asset : %s", assetID)
		assetSettlement := actions.AssetSettlementField{
			ContractIndex: contractInputIndex,
			AssetType:     assetTransfer.AssetType,
			AssetCode:     assetTransfer.AssetCode,
		}

		sendBalance := uint64(0)
		fromNonAdministration := uint64(0)
		fromAdministration := uint64(0)
		toNonAdministration := uint64(0)
		toAdministration := uint64(0)
		txid := protocol.TxIdFromBytes(transferTx.Hash[:])
		hds := make([]*state.Holding, len(settleTx.Outputs))
		updatedHoldings := make(map[bitcoin.Hash20]*state.Holding)
		(*updates)[*assetCode] = &updatedHoldings

		// Process senders
		// assetTransfer.AssetSenders []QuantityIndex {Index uint16, Quantity uint64}
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return fmt.Errorf("Sender input index out of range for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
			}

			if transferTx.Inputs[sender.Index].Address.Equal(ct.AdminAddress) {
				fromAdministration += sender.Quantity
			} else {
				fromNonAdministration += sender.Quantity
			}

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if !outputAddress.IsEmpty() && outputAddress.Equal(transferTx.Inputs[sender.Index].Address) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Sender output not found in settle tx for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Outputs))
			}

			// Check sender's available unfrozen balance
			if hds[settleOutputIndex] != nil {
				address := bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
					config.Net)
				node.LogWarn(ctx, "Duplicate sender entry: asset=%s party=%s", assetID, address)
				return node.NewError(actions.RejectionsMsgMalformed, "")
			}

			h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode,
				transferTx.Inputs[sender.Index].Address, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = h
			hash, err := transferTx.Inputs[sender.Index].Address.Hash()
			if err != nil {
				return errors.Wrap(err, "Invalid sender address")
			}
			updatedHoldings[*hash] = h

			address := bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
				config.Net)
			if err := holdings.AddDebit(h, txid, sender.Quantity, isSingleContract, v.Now); err != nil {
				if err == holdings.ErrInsufficientHoldings {
					node.LogWarn(ctx, "Insufficient funds: asset=%s party=%s : %d/%d", assetID,
						address.String(), sender.Quantity, holdings.SafeBalance(h))
					return node.NewError(actions.RejectionsInsufficientQuantity, "")
				}
				if err == holdings.ErrHoldingsFrozen {
					node.LogWarn(ctx, "Frozen funds: asset=%s party=%s", assetID, address)
					return node.NewError(actions.RejectionsHoldingsFrozen, "")
				}
				if err == holdings.ErrHoldingsLocked {
					node.LogWarn(ctx, "Locked funds: asset=%s party=%s", assetID, address)
					return node.NewError(actions.RejectionsHoldingsLocked, "")
				}
				node.LogWarn(ctx, "Send failed : %s : asset=%s party=%s", err, assetID, address)
				return node.NewError(actions.RejectionsMsgMalformed, "")
			} else {
				logger.Info(ctx, "Debit %d %s to %s", sender.Quantity, assetID, address)
			}

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
			if err != nil {
				return err
			}

			// Find output in settle tx
			settleOutputIndex := uint32(0x0000ffff)
			for i, outputAddress := range settleOutputAddresses {
				if !outputAddress.IsEmpty() && outputAddress.Equal(receiverAddress) {
					settleOutputIndex = uint32(i)
					break
				}
			}

			if settleOutputIndex == uint32(0x0000ffff) {
				address := bitcoin.NewAddressFromRawAddress(receiverAddress,
					config.Net)
				return fmt.Errorf("Receiver output not found in settle tx for asset %d receiver %d : %s",
					assetOffset, receiverOffset, address)
			}

			if receiverAddress.Equal(ct.AdminAddress) {
				toAdministration += receiver.Quantity
			} else {
				toNonAdministration += receiver.Quantity
			}

			if hds[settleOutputIndex] != nil {
				address := bitcoin.NewAddressFromRawAddress(receiverAddress,
					config.Net)
				node.LogWarn(ctx, "Duplicate receiver entry: asset=%s party=%s", assetID, address)
				return node.NewError(actions.RejectionsMsgMalformed, "")
			}

			h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode, receiverAddress, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = h
			hash, err := receiverAddress.Hash()
			if err != nil {
				return errors.Wrap(err, "Invalid receiver address")
			}
			updatedHoldings[*hash] = h

			address := bitcoin.NewAddressFromRawAddress(receiverAddress, config.Net)
			if err := holdings.AddDeposit(h, txid, receiver.Quantity, isSingleContract, v.Now); err != nil {
				if err == holdings.ErrHoldingsLocked {
					node.LogWarn(ctx, "Locked funds: asset=%s party=%s", assetID, address)
					return node.NewError(actions.RejectionsHoldingsLocked, "")
				}
				node.LogWarn(ctx, "Send failed : %s : asset=%s party=%s", err, assetID, address)
				return node.NewError(actions.RejectionsMsgMalformed, "")
			} else {
				logger.Info(ctx, "Deposit %d %s to %s", receiver.Quantity, assetID, address)
			}

			// Update asset balance
			if receiver.Quantity > sendBalance {
				return fmt.Errorf("Receiving more tokens than sending for asset %d", assetOffset)
			}
			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return fmt.Errorf("Not sending all input tokens for asset %d : %d remaining",
				assetOffset, sendBalance)
		}

		if !as.TransfersPermitted {
			if fromNonAdministration > toAdministration {
				node.LogWarn(ctx, "Transfers not permitted. Sending tokens not all to administration : %d/%d",
					fromNonAdministration, toAdministration)
				return node.NewError(actions.RejectionsAssetNotPermitted, "")
			}
			if toNonAdministration > fromAdministration {
				node.LogWarn(ctx, "Transfers not permitted. Receiving tokens not all from administration : %d/%d",
					toNonAdministration, fromAdministration)
				return node.NewError(actions.RejectionsAssetNotPermitted, "")
			}
		}

		for index, holding := range hds {
			if holding != nil {
				assetSettlement.Settlements = append(assetSettlement.Settlements,
					&actions.QuantityIndexField{Index: uint32(index), Quantity: holding.PendingBalance})
			}
		}

		// Check if settlement already exists for this asset.
		replaced := false
		for i, asset := range settlement.Assets {
			if asset.AssetType == assetSettlement.AssetType &&
				bytes.Equal(asset.AssetCode, assetSettlement.AssetCode) {
				replaced = true
				settlement.Assets[i] = &assetSettlement
				break
			}
		}

		if !replaced {
			settlement.Assets = append(settlement.Assets, &assetSettlement) // Append
		}
		dataAdded = true
	}

	if !dataAdded {
		return errors.New("No data added to settlement")
	}

	// Serialize settlement data back into OP_RETURN output.
	script, err := protocol.Serialize(settlement, config.IsTest)
	if err != nil {
		return fmt.Errorf("Failed to serialize empty settlement : %s", err)
	}

	// Find Settlement OP_RETURN.
	found := false
	settlementOutputIndex := 0
	for i, output := range settleTx.MsgTx.TxOut {
		action, err := protocol.Deserialize(output.PkScript, config.IsTest)
		if err != nil {
			continue
		}
		if action.Code() == actions.CodeSettlement {
			settlementOutputIndex = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Settlement op return not found in settle tx")
	}

	settleTx.MsgTx.TxOut[settlementOutputIndex].PkScript = script
	return nil
}

// findBoomerangIndex returns the index to the "boomerang" output from transfer tx. It is the
//   output to the contract that is not referenced/spent by the transfers. It is used to fund the
//   offer and signature request messages required between multiple contracts to get a fully
//   approved settlement tx.
func findBoomerangIndex(transferTx *inspector.Transaction,
	transfer *actions.Transfer,
	contractAddress bitcoin.RawAddress) uint32 {

	outputUsed := make([]bool, len(transferTx.Outputs))
	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.ContractIndex == uint32(0x0000ffff) ||
			(assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0) {
			continue
		}

		if int(assetTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return 0xffffffff
		}

		// Output will be spent by settlement tx.
		outputUsed[assetTransfer.ContractIndex] = true
	}

	for index, output := range transferTx.Outputs {
		if outputUsed[index] {
			continue
		}
		if output.Address.Equal(contractAddress) {
			return uint32(index)
		}
	}

	return 0xffffffff
}

// sendToNextSettlementContract sends settlement data to the next contract involved so it can add its data.
func sendToNextSettlementContract(ctx context.Context,
	w *node.ResponseWriter,
	rk *wallet.Key,
	itx *inspector.Transaction,
	transferTx *inspector.Transaction,
	transfer *actions.Transfer,
	settleTx *txbuilder.TxBuilder,
	settlement *actions.Settlement,
	settlementRequest *messages.SettlementRequest,
	tracer *filters.Tracer) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.sendToNextSettlementContract")
	defer span.End()

	boomerangIndex := uint32(0xffffffff)
	if !bytes.Equal(itx.Hash[:], transferTx.Hash[:]) {
		// If already an M1, use only output
		boomerangIndex = 0
	} else {
		boomerangIndex = findBoomerangIndex(transferTx, transfer, rk.Address)
	}

	if boomerangIndex == 0xffffffff {
		return fmt.Errorf("Multi-Contract Transfer missing boomerang output")
	}
	node.LogVerbose(ctx, "Boomerang output index : %d", boomerangIndex)

	// Find next contract
	nextContractIndex := uint32(0x0000ffff)
	currentFound := false
	completedContracts := make(map[bitcoin.Hash20]bool)
	for _, asset := range transfer.Assets {
		if asset.ContractIndex == uint32(0x0000ffff) {
			continue // Asset transfer doesn't have a contract (probably BSV transfer).
		}

		if int(asset.ContractIndex) >= len(transferTx.Outputs) {
			return errors.New("Transfer contract index out of range")
		}

		hash, err := transferTx.Outputs[asset.ContractIndex].Address.Hash()
		if err != nil {
			return errors.Wrap(err, "Transfer contract address invalid")
		}

		if !currentFound {
			completedContracts[*hash] = true
			if transferTx.Outputs[asset.ContractIndex].Address.Equal(rk.Address) {
				currentFound = true
			}
			continue
		}

		// Contracts can be used more than once, so ensure this contract wasn't referenced before
		//   the current contract.
		_, complete := completedContracts[*hash]
		if !complete {
			nextContractIndex = asset.ContractIndex
			break
		}
	}

	if nextContractIndex == 0xffff {
		return fmt.Errorf("Next contract not found in multi-contract transfer")
	}

	node.Log(ctx, "Sending settlement offer to %x",
		transferTx.Outputs[nextContractIndex].Address.Bytes())

	// Setup M1 response
	var err error
	err = w.SetUTXOs(ctx, []bitcoin.UTXO{itx.Outputs[boomerangIndex].UTXO})
	if err != nil {
		return err
	}

	// Add output to next contract.
	// Mark as change so it gets everything except the tx fee.
	err = w.AddChangeOutput(ctx, transferTx.Outputs[nextContractIndex].Address)
	if err != nil {
		return err
	}

	// Serialize settlement tx for Message payload.
	settlementRequest.Settlement, err = protocol.Serialize(settlement, w.Config.IsTest)
	if err != nil {
		return err
	}

	// Setup Message
	var payBuf bytes.Buffer
	err = settlementRequest.Serialize(&payBuf)
	if err != nil {
		return err
	}
	message := actions.Message{
		ReceiverIndexes: []uint32{0}, // First output is receiver of message
		MessageCode:     settlementRequest.Code(),
		MessagePayload:  payBuf.Bytes(),
	}

	if err := node.RespondSuccess(ctx, w, itx, rk, &message); err != nil {
		return err
	}

	if bytes.Equal(itx.Hash[:], transferTx.Hash[:]) {
		outpoint := wire.OutPoint{Hash: *itx.Hash, Index: boomerangIndex}
		tracer.Add(ctx, &outpoint)
	}
	return nil
}

// transferIsSingleContract returns true if this contract can settle all assets in the transfer.
func transferIsSingleContract(ctx context.Context, itx *inspector.Transaction,
	transfer *actions.Transfer, rk *wallet.Key) bool {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.transferIsSingleContract")
	defer span.End()

	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" {
			continue // All contracts can handle bitcoin transfers
		}

		if int(assetTransfer.ContractIndex) >= len(itx.Outputs) {
			return false // Invalid contract index
		}

		if !itx.Outputs[assetTransfer.ContractIndex].Address.Equal(rk.Address) {
			return false // Another contract is involved
		}
	}

	return true
}

// SettlementResponse handles an outgoing Settlement action and writes it to the state
func (t *Transfer) SettlementResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.SettlementResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Settlement)
	if !ok {
		return errors.New("Could not assert as *actions.Settlement")
	}

	txid := protocol.TxIdFromBytes(itx.Inputs[0].UTXO.Hash[:])
	ct, err := contract.Retrieve(ctx, t.MasterDB, rk.Address, t.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	assetUpdates := make(map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding)
	for assetIndex, assetSettlement := range msg.Assets {
		if assetSettlement.AssetType == "BSV" && len(assetSettlement.AssetCode) == 0 {
			continue // Bitcoin transaction
		}

		if assetSettlement.ContractIndex == 0x0000ffff {
			continue // No contract for this asset
		}

		if int(assetSettlement.ContractIndex) >= len(itx.Inputs) {
			return fmt.Errorf("Settlement contract index %d out of range : %d >= %d", assetIndex,
				assetSettlement.ContractIndex, len(itx.Inputs))
		}

		if !itx.Inputs[assetSettlement.ContractIndex].Address.Equal(rk.Address) {
			continue // Asset not under this contract
		}

		assetCode := protocol.AssetCodeFromBytes(assetSettlement.AssetCode)
		assetID := protocol.AssetID(assetSettlement.AssetType, *assetCode)

		hds := make(map[bitcoin.Hash20]*state.Holding)
		assetUpdates[*assetCode] = &hds

		timestamp := protocol.NewTimestamp(msg.Timestamp)

		// Finalize settlements
		for _, settlementQuantity := range assetSettlement.Settlements {
			if int(settlementQuantity.Index) >= len(itx.Outputs) {
				return fmt.Errorf("Settlement output index out of range %d >= %d : %s",
					settlementQuantity.Index, len(itx.Outputs), assetID)
			}

			h, err := holdings.GetHolding(ctx, t.MasterDB, rk.Address, assetCode,
				itx.Outputs[settlementQuantity.Index].Address, timestamp)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}

			ra := itx.Outputs[settlementQuantity.Index].Address
			address := bitcoin.NewAddressFromRawAddress(ra, w.Config.Net)
			if err = holdings.FinalizeTx(h, txid, settlementQuantity.Quantity,
				timestamp); err != nil {
				return fmt.Errorf("Failed settlement finalize for holding : %s %s : %s", assetID,
					address, err)
			}

			logger.Info(ctx, "Settled %s balance of %d for %s", assetID, h.FinalizedBalance,
				address)

			hash, err := itx.Outputs[settlementQuantity.Index].Address.Hash()
			if err != nil {
				return errors.Wrap(err, "Invalid settlement address")
			}
			hds[*hash] = h
		}
	}

	// Now that no errors were found we can save all the data.
	for assetCode, hds := range assetUpdates {
		for _, h := range *hds {
			cacheItem, err := holdings.Save(ctx, t.MasterDB, rk.Address, &assetCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
			t.HoldingsChannel.Add(cacheItem)
		}
	}

	return nil
}

// respondTransferReject sends a reject to all parties involved with a transfer request and refunds
//   any bitcoin involved. This can only be done by the first contract, because they hold the
//   bitcoin to be distributed.
func respondTransferReject(ctx context.Context, masterDB *db.DB,
	holdingsChannel *holdings.CacheChannel, config *node.Config, w *node.ResponseWriter,
	transferTx *inspector.Transaction, transfer *actions.Transfer, rk *wallet.Key, code uint32,
	started bool, text string) error {

	v := ctx.Value(node.KeyValues).(*node.Values)
	transferTxId := protocol.TxIdFromBytes(transferTx.Hash[:])

	// Determine UTXOs to fund the reject response.
	utxos, err := transferTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Transfer UTXOs not found")
	}

	// Remove boomerang from funding UTXOs since it was already spent.
	if started {
		// Remove utxo spent by boomerang
		boomerangIndex := findBoomerangIndex(transferTx, transfer, rk.Address)
		if boomerangIndex != 0xffffffff && transferTx.Outputs[boomerangIndex].Address.Equal(rk.Address) {
			found := false
			for i, utxo := range utxos {
				if utxo.Index == boomerangIndex {
					found = true
					utxos = append(utxos[:i], utxos[i+1:]...) // Remove
					break
				}
			}

			if !found {
				return errors.New("Boomerang output not found")
			}
		}
	}

	balance := uint64(0)
	for _, utxo := range utxos {
		balance += uint64(utxo.Value)
	}

	updates := make(map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding)

	w.SetRejectUTXOs(ctx, utxos)

	// Add refund amounts for all bitcoin senders (if "first" contract, first contract receives bitcoin funds to be distributed)
	first := firstContractOutputIndex(transfer.Assets, transferTx)
	if first == 0xffff {
		return errors.New("First contract output index not found")
	}

	// Determine if this contract is the first contract and needs to send a refund.
	if !transferTx.Outputs[first].Address.Equal(rk.Address) {
		return errors.New("This is not the first contract")
	}

	refundBalance := uint64(0)
	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			// Process bitcoin senders refunds
			for _, sender := range assetTransfer.AssetSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				node.LogVerbose(ctx, "Bitcoin refund %d : %x", sender.Quantity,
					transferTx.Inputs[sender.Index].Address.Bytes())
				w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, sender.Quantity)
				refundBalance += sender.Quantity
			}
		} else {
			// Add all other senders to be notified
			for _, sender := range assetTransfer.AssetSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, 0)
			}

			if started { // Revert holding statuses
				if len(transferTx.Outputs) <= int(assetTransfer.ContractIndex) {
					return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
				}

				if !transferTx.Outputs[assetTransfer.ContractIndex].Address.Equal(rk.Address) {
					continue // This asset is not ours. Skip it.
				}

				assetCode := protocol.AssetCodeFromBytes(assetTransfer.AssetCode)
				updatedHoldings := make(map[bitcoin.Hash20]*state.Holding)
				updates[*assetCode] = &updatedHoldings

				// Revert sender pending statuses
				for _, sender := range assetTransfer.AssetSenders {
					// Revert holding status
					h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode,
						transferTx.Inputs[sender.Index].Address, v.Now)
					if err != nil {
						return errors.Wrap(err, "get holding")
					}

					hash, err := transferTx.Inputs[sender.Index].Address.Hash()
					if err != nil {
						return errors.Wrap(err, "sender address hash")
					}
					updatedHoldings[*hash] = h

					// Revert holding status
					err = holdings.RevertStatus(h, transferTxId)
					if err != nil {
						return errors.Wrap(err, "revert status")
					}
				}

				// Revert receiver pending statuses
				for _, receiver := range assetTransfer.AssetReceivers {
					receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
					if err != nil {
						return err
					}

					h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode,
						receiverAddress, v.Now)
					if err != nil {
						return errors.Wrap(err, "get holding")
					}

					hash, err := receiverAddress.Hash()
					if err != nil {
						return errors.Wrap(err, "receiver address hash")
					}
					updatedHoldings[*hash] = h

					// Revert holding status
					err = holdings.RevertStatus(h, transferTxId)
					if err != nil {
						return errors.Wrap(err, "revert status")
					}
				}
			}
		}
	}

	if started {
		err = saveHoldings(ctx, masterDB, holdingsChannel, updates, rk.Address)
		if err != nil {
			return errors.Wrap(err, "save holdings")
		}
	}

	if refundBalance > balance {
		ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}

		// Funding not enough to refund everyone, so don't refund to anyone. Send it to the
		//   administration to hold.
		w.ClearRejectOutputValues(ct.AdminAddress)
	}

	return node.RespondRejectText(ctx, w, transferTx, rk, code, text)
}
