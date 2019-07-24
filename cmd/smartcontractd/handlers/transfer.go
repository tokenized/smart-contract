package handlers

import (
	"bytes"
	"context"
	"fmt"

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
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Transfer struct {
	handler   protomux.Handler
	MasterDB  *db.DB
	Config    *node.Config
	Headers   node.BitcoinHeaders
	Tracer    *filters.Tracer
	Scheduler *scheduler.Scheduler
}

type rejectError struct {
	code uint8
	text string
}

func (err rejectError) Error() string {
	if err.code == 0 {
		return err.text
	}
	value := protocol.GetRejectionCode(err.code)
	if value == nil {
		return err.text
	}
	if len(err.text) == 0 {
		return value.Label
	}
	return fmt.Sprintf("%s - %s", value.Label, err.text)
}

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferRequest")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	msg, ok := itx.MsgProto.(*protocol.Transfer)
	if !ok {
		return errors.New("Could not assert as *protocol.Transfer")
	}

	// Find "first" contract.
	first := firstContractOutputIndex(msg.Assets, itx)

	if first == 0xffff {
		node.LogWarn(ctx, "Transfer first contract not found : %s",
			rk.Address.String(wire.BitcoinNet(w.Config.ChainParams.Net)))
		return errors.New("Transfer first contract not found")
	}

	if !itx.Outputs[first].Address.Equal(rk.Address) {
		node.LogVerbose(ctx, "Not contract for first transfer. Waiting for Message Offer : %s",
			itx.Outputs[first].Address.String(wire.BitcoinNet(w.Config.ChainParams.Net)))
		if err := transactions.AddTx(ctx, t.MasterDB, itx); err != nil {
			return errors.Wrap(err, "Failed to save tx")
		}
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Transfer request invalid")
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, itx.RejectCode, false)
	}

	// Check pre-processing reject code
	if itx.RejectCode != 0 {
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, itx.RejectCode, false)
	}

	if msg.OfferExpiry.Nano() != 0 && v.Now.Nano() > msg.OfferExpiry.Nano() {
		node.LogWarn(ctx, "Transfer expired : %s", msg.OfferExpiry.String())
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectTransferExpired, false)
	}

	if len(msg.Assets) == 0 {
		node.LogWarn(ctx, "Transfer has no asset transfers")
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectTransferExpired, false)
	}

	// Bitcoin balance of first (this) contract. Funding for bitcoin transfers.
	contractBalance := uint64(itx.Outputs[first].Value)

	settlementRequest := protocol.SettlementRequest{
		Version:   0,
		Timestamp: v.Now,
	}

	err := settlementRequest.TransferTxId.Set(itx.Hash[:])
	if err != nil {
		return err
	}

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	ct, err := contract.Retrieve(ctx, t.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		node.LogWarn(ctx, "Contract address changed : %s", ct.MovedTo.String())
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectContractMoved, false)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen : %s", contractPKH.String())
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectContractFrozen, false)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectContractExpired, false)
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
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectMsgMalformed, false)
	}

	// Update outputs to pay bitcoin receivers.
	err = addBitcoinSettlements(ctx, itx, msg, settleTx)
	if err != nil {
		node.LogWarn(ctx, "Failed to add bitcoin settlements : %s", err)
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
			protocol.RejectMsgMalformed, false)
	}

	// Create initial settlement data
	settlement := protocol.Settlement{Timestamp: v.Now}

	// Serialize empty settlement data into OP_RETURN output as a placeholder to be updated by addSettlementData.
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
	assetUpdates := make(map[protocol.AssetCode]map[protocol.PublicKeyHash]*state.Holding)
	err = addSettlementData(ctx, t.MasterDB, t.Config, rk, itx, msg, settleTx, &settlement,
		t.Headers, assetUpdates)
	if err != nil {
		reject, ok := err.(rejectError)
		if ok {
			node.LogWarn(ctx, "Rejecting Transfer : %s", err)
			return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, reject.code, false)
		} else {
			return errors.Wrap(err, "Failed to add settlement data")
		}
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, msg, &settlement) {
		node.Log(ctx, "Single contract settlement complete")
		if err := settleTx.Sign([]bitcoin.Key{rk.Key}); err != nil {
			node.LogWarn(ctx, "Failed to sign settle tx : %s", err)
			return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk,
				protocol.RejectInsufficientValue, false)
		}

		err := node.Respond(ctx, w, settleTx.MsgTx)
		if err == nil {
			if err := t.saveHoldings(ctx, assetUpdates, contractPKH); err != nil {
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
	pendingTransfer := state.PendingTransfer{TransferTxId: *protocol.TxIdFromBytes(itx.Hash[:]),
		Timeout: timeout}
	if err := transfer.Save(ctx, t.MasterDB, contractPKH, &pendingTransfer); err != nil {
		return errors.Wrap(err, "Failed to save pending transfer")
	}

	// Schedule timeout for transfer in case the other contract(s) don't respond.
	if err := t.Scheduler.ScheduleJob(ctx, listeners.NewTransferTimeout(t.handler, itx, timeout)); err != nil {
		return errors.Wrap(err, "Failed to schedule transfer timeout")
	}

	if err := t.saveHoldings(ctx, assetUpdates, contractPKH); err != nil {
		return err
	}

	return nil
}

func (t *Transfer) saveHoldings(ctx context.Context,
	updates map[protocol.AssetCode]map[protocol.PublicKeyHash]*state.Holding,
	contractPKH *protocol.PublicKeyHash) error {

	for assetCode, hds := range updates {
		for _, h := range hds {
			if err := holdings.Save(ctx, t.MasterDB, contractPKH, &assetCode, h); err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
		}
	}

	return nil
}

func (t *Transfer) revertHoldings(ctx context.Context,
	updates map[protocol.AssetCode]map[protocol.PublicKeyHash]*state.Holding,
	contractPKH *protocol.PublicKeyHash,
	txid *protocol.TxId) error {

	for _, hds := range updates {
		for _, h := range hds {
			if err := holdings.RevertStatus(h, txid); err != nil {
				return errors.Wrap(err, "Failed to revert holding status")
			}
		}
	}

	return t.saveHoldings(ctx, updates, contractPKH)
}

// TransferTimeout is called when a multi-contract transfer times out because the other contracts are not responding.
func (t *Transfer) TransferTimeout(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferTimeout")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Transfer)
	if !ok {
		return errors.New("Could not assert as *protocol.Transfer")
	}

	// Remove pending transfer
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	if err := transfer.Remove(ctx, t.MasterDB, contractPKH, protocol.TxIdFromBytes(itx.Hash[:])); err != nil {
		if err != transfer.ErrNotFound {
			return errors.Wrap(err, "Failed to remove pending transfer")
		}
	}

	node.LogWarn(ctx, "Transfer timed out")
	return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectTimeout, true)
}

// firstContractOutputIndex finds the "first" contract. The "first" contract of a transfer is the one
//   responsible for creating the initial settlement data and passing it to the next contract if
//   there are more than one.
func firstContractOutputIndex(assetTransfers []protocol.AssetTransfer, itx *inspector.Transaction) uint16 {
	transferIndex := uint16(0)
	for _, asset := range assetTransfers {
		if asset.ContractIndex != 0xffff {
			break
		}
		// Asset transfer doesn't have a contract (probably bitcoin transfer).
		transferIndex++
	}

	if int(transferIndex) >= len(assetTransfers) {
		return 0xffff
	}

	if int(assetTransfers[transferIndex].ContractIndex) >= len(itx.Outputs) {
		return 0xffff
	}

	return assetTransfers[transferIndex].ContractIndex
}

// buildSettlementTx builds the tx for a settlement action.
func buildSettlementTx(ctx context.Context, masterDB *db.DB, config *node.Config, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settlementRequest *protocol.SettlementRequest, contractBalance uint64, rk *wallet.Key) (*txbuilder.TxBuilder, error) {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.buildSettlementTx")
	defer span.End()

	// Settle Outputs
	//   Any addresses sending or receiving tokens or bitcoin.
	//   Referenced from indices from within settlement data.
	//
	// Settle Inputs
	//   Any contracts involved.
	settleTx := txbuilder.NewTxBuilder(rk.Address, config.DustLimit, config.FeeRate)

	var err error
	addresses := make(map[protocol.PublicKeyHash]uint32)
	outputUsed := make([]bool, len(transferTx.Outputs))

	// Setup inputs from outputs of the Transfer tx. One from each contract involved.
	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.ContractIndex == uint16(0xffff) ||
			(assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()) {
			continue
		}

		if int(assetTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return nil, fmt.Errorf("Transfer contract index out of range %d", assetOffset)
		}

		if outputUsed[assetTransfer.ContractIndex] {
			continue
		}

		// Add input from contract to settlement tx so all involved contracts have to sign for a valid tx.
		err = settleTx.AddInput(wire.OutPoint{Hash: transferTx.Hash, Index: uint32(assetTransfer.ContractIndex)},
			transferTx.Outputs[assetTransfer.ContractIndex].UTXO.PkScript,
			uint64(transferTx.Outputs[assetTransfer.ContractIndex].Value))
		if err != nil {
			return nil, err
		}
		outputUsed[assetTransfer.ContractIndex] = true
	}

	// Setup outputs
	//   One to each receiver, including any bitcoins received, or dust.
	//   One to each sender with dust amount.
	for assetOffset, assetTransfer := range transfer.Assets {
		assetIsBitcoin := assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()
		assetBalance := uint64(0)

		// Add all senders
		for _, quantityIndex := range assetTransfer.AssetSenders {
			assetBalance += quantityIndex.Quantity

			if quantityIndex.Index >= uint16(len(transferTx.Inputs)) {
				return nil, fmt.Errorf("Transfer sender index out of range %d", assetOffset)
			}

			addressPKH, ok := transferTx.Inputs[quantityIndex.Index].Address.(*bitcoin.AddressPKH)
			if !ok {
				return nil, fmt.Errorf("Transfer sender not PKH %d", assetOffset)
			}
			address := protocol.PublicKeyHashFromBytes(addressPKH.PKH())
			_, exists := addresses[*address]
			if !exists {
				// Add output to sender
				addresses[*address] = uint32(len(settleTx.MsgTx.TxOut))

				err = settleTx.AddDustOutput(transferTx.Inputs[quantityIndex.Index].Address, false)
				if err != nil {
					return nil, err
				}
			}
		}

		var receiverAddress bitcoin.Address
		for _, assetReceiver := range assetTransfer.AssetReceivers {
			assetBalance -= assetReceiver.Quantity

			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if assetReceiver.Quantity > contractBalance {
					return nil, fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= assetReceiver.Quantity
			}

			outputIndex, exists := addresses[assetReceiver.Address]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, assetReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver
				addresses[assetReceiver.Address] = uint32(len(settleTx.MsgTx.TxOut))

				receiverAddress, err = bitcoin.NewAddressPKH(assetReceiver.Address.Bytes())
				if err != nil {
					return nil, err
				}
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
		feeAddress, err := bitcoin.NewAddressPKH(fee.Address.Bytes())
		if err != nil {
			return nil, err
		}
		settleTx.AddPaymentOutput(feeAddress, fee.Quantity, false)
	}

	// Add this contract's fee output
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
	if err != nil {
		return settleTx, errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.ContractFee > 0 {
		settleTx.AddPaymentOutput(config.FeeAddress, ct.ContractFee, false)

		// Add to settlement request
		feeAddressPKH, ok := config.FeeAddress.(*bitcoin.AddressPKH)
		if !ok {
			return settleTx, errors.New("Fee address not PKH")
		}
		feePKH := protocol.PublicKeyHashFromBytes(feeAddressPKH.PKH())
		settlementRequest.ContractFees = append(settlementRequest.ContractFees,
			protocol.TargetAddress{Address: *feePKH, Quantity: ct.ContractFee})
	}

	return settleTx, nil
}

// addBitcoinSettlements adds bitcoin settlement data to the Settlement data
func addBitcoinSettlements(ctx context.Context, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settleTx *txbuilder.TxBuilder) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addBitcoinSettlements")
	defer span.End()

	// Check for bitcoin transfers.
	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType != "CUR" || !assetTransfer.AssetCode.IsZero() {
			continue
		}

		sendBalance := uint64(0)

		// Process senders
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return fmt.Errorf("Sender input index out of range for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
			}

			input := transferTx.Inputs[sender.Index]

			// Get sender's balance
			if int64(sender.Quantity) >= input.Value {
				return fmt.Errorf("Sender bitcoin quantity higher than input amount for sender %d : %d/%d",
					senderOffset, input.Value, sender.Quantity)
			}

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Find output for receiver
			added := false
			for i, _ := range settleTx.MsgTx.TxOut {
				outputAddress, err := settleTx.OutputAddress(i)
				if err != nil {
					continue
				}
				outputAddressPKH, ok := outputAddress.(*bitcoin.AddressPKH)
				if !ok {
					continue
				}
				if bytes.Equal(receiver.Address.Bytes(), outputAddressPKH.PKH()) {
					// Add balance to receiver's output
					settleTx.AddValueToOutput(uint32(i), receiver.Quantity)
					added = true
					break
				}
			}

			if !added {
				return fmt.Errorf("Receiver bitcoin output missing output data for receiver %d", receiverOffset)
			}

			if receiver.Quantity >= sendBalance {
				return fmt.Errorf("Sending more bitcoin than received")
			}

			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return fmt.Errorf("Not sending all recieved bitcoins : %d remaining", sendBalance)
		}
	}

	// Add exchange fee
	if !transfer.ExchangeFeeAddress.IsZero() && transfer.ExchangeFee > 0 {
		// Find output for receiver
		added := false
		for i, _ := range settleTx.MsgTx.TxOut {
			outputAddress, err := settleTx.OutputAddress(i)
			if err != nil {
				continue
			}
			outputAddressPKH, ok := outputAddress.(*bitcoin.AddressPKH)
			if !ok {
				continue
			}
			if bytes.Equal(transfer.ExchangeFeeAddress.Bytes(), outputAddressPKH.PKH()) {
				// Add exchange fee to existing output
				settleTx.AddValueToOutput(uint32(i), transfer.ExchangeFee)
				added = true
				break
			}
		}

		if !added {
			// Add new output for exchange fee.
			exchangeAddress, err := bitcoin.NewAddressPKH(transfer.ExchangeFeeAddress.Bytes())
			if err != nil {
				return errors.Wrap(err, "Failed to create exchange address")
			}
			if err := settleTx.AddPaymentOutput(exchangeAddress, transfer.ExchangeFee, false); err != nil {
				return errors.Wrap(err, "Failed to add exchange fee output")
			}
		}
	}

	return nil
}

// addSettlementData appends data to a pending settlement action.
func addSettlementData(ctx context.Context, masterDB *db.DB, config *node.Config, rk *wallet.Key,
	transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *txbuilder.TxBuilder, settlement *protocol.Settlement, headers node.BitcoinHeaders,
	updates map[protocol.AssetCode]map[protocol.PublicKeyHash]*state.Holding) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addSettlementData")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	dataAdded := false

	ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		return rejectError{code: protocol.RejectContractFrozen}
	}

	// Generate public key hashes for all the outputs
	transferOutputAddresses := make([]bitcoin.Address, 0, len(transferTx.Outputs))
	for _, output := range transferTx.Outputs {
		transferOutputAddresses = append(transferOutputAddresses, output.Address)
	}

	// Generate public key hashes for all the inputs
	settleInputAddresses := make([]bitcoin.Address, 0, len(settleTx.Inputs))
	for _, input := range settleTx.Inputs {
		address, err := bitcoin.AddressFromLockingScript(input.LockScript)
		if err != nil {
			settleInputAddresses = append(settleInputAddresses, nil)
			continue
		}
		settleInputAddresses = append(settleInputAddresses, address)
	}

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]bitcoin.Address, 0, len(settleTx.MsgTx.TxOut))
	for _, output := range settleTx.MsgTx.TxOut {
		address, err := bitcoin.AddressFromLockingScript(output.PkScript)
		if err != nil {
			settleOutputAddresses = append(settleOutputAddresses, nil)
			continue
		}
		settleOutputAddresses = append(settleOutputAddresses, address)
	}

	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero() {
			node.LogVerbose(ctx, "Asset transfer for bitcoin")
			continue // Skip bitcoin transfers since they should be handled already
		}

		if len(transferTx.Outputs) <= int(assetTransfer.ContractIndex) {
			return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
		}

		contractOutputAddress := transferOutputAddresses[assetTransfer.ContractIndex]
		if contractOutputAddress == nil || !contractOutputAddress.Equal(rk.Address) {
			continue // This asset is not ours. Skip it.
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetTransfer.AssetCode)
		if err != nil {
			return fmt.Errorf("Asset ID not found : %s %s : %s", contractPKH, assetTransfer.AssetCode.String(), err)
		}
		if as.FreezePeriod.Nano() > v.Now.Nano() {
			node.LogWarn(ctx, "Asset frozen until %s", as.FreezePeriod.String())
			return rejectError{code: protocol.RejectAssetFrozen}
		}

		// Find contract input
		contractInputIndex := uint16(0xffff)
		for i, input := range settleInputAddresses {
			if input != nil && input.Equal(rk.Address) {
				contractInputIndex = uint16(i)
				break
			}
		}

		if contractInputIndex == uint16(0xffff) {
			return fmt.Errorf("Contract input not found: %s %s", contractPKH, assetTransfer.AssetCode.String())
		}

		node.LogVerbose(ctx, "Adding settlement data for asset : %s", assetTransfer.AssetCode.String())
		assetSettlement := protocol.AssetSettlement{
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
		updatedHoldings := make(map[protocol.PublicKeyHash]*state.Holding)
		updates[assetTransfer.AssetCode] = updatedHoldings

		// Process senders
		// assetTransfer.AssetSenders []QuantityIndex {Index uint16, Quantity uint64}
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return fmt.Errorf("Sender input index out of range for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
			}

			addressPKH, ok := transferTx.Inputs[sender.Index].Address.(*bitcoin.AddressPKH)
			if !ok {
				return fmt.Errorf("Sender input not PKH: %s %s", contractPKH, assetTransfer.AssetCode.String())
			}
			inputPKH := protocol.PublicKeyHashFromBytes(addressPKH.PKH())

			if bytes.Equal(inputPKH.Bytes(), ct.AdministrationPKH.Bytes()) {
				fromAdministration += sender.Quantity
			} else {
				fromNonAdministration += sender.Quantity
			}

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if outputAddress != nil && outputAddress.Equal(transferTx.Inputs[sender.Index].Address) {
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
				node.LogWarn(ctx, "Duplicate sender entry: contract=%s asset=%s party=%s",
					contractPKH, assetTransfer.AssetCode.String(), inputPKH.String())
				return rejectError{code: protocol.RejectMsgMalformed}
			}

			h, err := holdings.GetHolding(ctx, masterDB, contractPKH, &assetTransfer.AssetCode, inputPKH, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = &h
			updatedHoldings[*inputPKH] = &h

			if err := holdings.AddDebit(&h, txid, sender.Quantity, v.Now); err != nil {
				if err == holdings.ErrInsufficientHoldings {
					node.LogWarn(ctx, "Insufficient funds: contract=%s asset=%s party=%s",
						contractPKH, assetTransfer.AssetCode.String(), inputPKH.String())
					return rejectError{code: protocol.RejectInsufficientQuantity}
				}
				if err == holdings.ErrHoldingsFrozen {
					node.LogWarn(ctx, "Frozen funds: asset=%s party=%s",
						assetTransfer.AssetCode.String(), inputPKH.String())
					return rejectError{code: protocol.RejectHoldingsFrozen}
				}
				node.LogWarn(ctx, "Send failed : %s : contract=%s asset=%s party=%s",
					err, contractPKH, assetTransfer.AssetCode.String(), inputPKH.String())
				return rejectError{code: protocol.RejectMsgMalformed}
			}

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				receiverAddress, err := bitcoin.NewAddressPKH(receiver.Address.Bytes())
				if err == nil && outputAddress != nil && outputAddress.Equal(receiverAddress) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Receiver output not found in settle tx for asset %d receiver %d : %s",
					assetOffset, receiverOffset, receiver.Address.String())
			}

			if bytes.Equal(receiver.Address.Bytes(), ct.AdministrationPKH.Bytes()) {
				toAdministration += receiver.Quantity
			} else {
				toNonAdministration += receiver.Quantity
			}

			if hds[settleOutputIndex] != nil {
				node.LogWarn(ctx, "Duplicate receiver entry: contract=%s asset=%s party=%s",
					contractPKH, assetTransfer.AssetCode.String(), receiver.Address.String())
				return rejectError{code: protocol.RejectMsgMalformed}
			}

			h, err := holdings.GetHolding(ctx, masterDB, contractPKH, &assetTransfer.AssetCode, &receiver.Address, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = &h
			updatedHoldings[receiver.Address] = &h

			if err := holdings.AddDeposit(&h, txid, receiver.Quantity, v.Now); err != nil {
				node.LogWarn(ctx, "Send failed : %s : contract=%s asset=%s party=%s",
					err, contractPKH, assetTransfer.AssetCode.String(), receiver.Address.String())
				return rejectError{code: protocol.RejectMsgMalformed}
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
				return rejectError{code: protocol.RejectAssetNotPermitted}
			}
			if toNonAdministration > fromAdministration {
				node.LogWarn(ctx, "Transfers not permitted. Receiving tokens not all from administration : %d/%d",
					toNonAdministration, fromAdministration)
				return rejectError{code: protocol.RejectAssetNotPermitted}
			}
		}

		for index, holding := range hds {
			if holding != nil {
				assetSettlement.Settlements = append(assetSettlement.Settlements,
					protocol.QuantityIndex{Index: uint16(index), Quantity: holding.PendingBalance})
			}
		}

		// Check if settlement already exists for this asset.
		replaced := false
		for i, asset := range settlement.Assets {
			if asset.AssetType == assetSettlement.AssetType &&
				bytes.Equal(asset.AssetCode.Bytes(), assetSettlement.AssetCode.Bytes()) {
				replaced = true
				settlement.Assets[i] = assetSettlement
				break
			}
		}

		if !replaced {
			settlement.Assets = append(settlement.Assets, assetSettlement) // Append
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
		code, err := protocol.Code(output.PkScript, config.IsTest)
		if err != nil {
			continue
		}
		if code == protocol.CodeSettlement {
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
func findBoomerangIndex(transferTx *inspector.Transaction, transfer *protocol.Transfer, contractAddress bitcoin.Address) uint32 {
	outputUsed := make([]bool, len(transferTx.Outputs))
	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.ContractIndex == uint16(0xffff) ||
			(assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()) {
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
func sendToNextSettlementContract(ctx context.Context, w *node.ResponseWriter, rk *wallet.Key,
	itx *inspector.Transaction, transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *txbuilder.TxBuilder, settlement *protocol.Settlement, settlementRequest *protocol.SettlementRequest,
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
	nextContractIndex := uint16(0xffff)
	currentFound := false
	completedContracts := make(map[[20]byte]bool)
	for _, asset := range transfer.Assets {
		if asset.ContractIndex == uint16(0xffff) {
			continue // Asset transfer doesn't have a contract (probably BSV transfer).
		}

		if int(asset.ContractIndex) >= len(transferTx.Outputs) {
			return errors.New("Transfer contract index out of range")
		}

		addressPKH, ok := transferTx.Outputs[asset.ContractIndex].Address.(*bitcoin.AddressPKH)
		if !ok {
			continue
		}
		var pkh [20]byte
		copy(pkh[:], addressPKH.PKH())

		if !currentFound {
			completedContracts[pkh] = true
			if bytes.Equal(pkh[:], bitcoin.Hash160(rk.Key.PublicKey().Bytes())) {
				currentFound = true
			}
			continue
		}

		// Contracts can be used more than once, so ensure this contract wasn't referenced before
		//   the current contract.
		_, complete := completedContracts[pkh]
		if !complete {
			nextContractIndex = asset.ContractIndex
			break
		}
	}

	if nextContractIndex == 0xffff {
		return fmt.Errorf("Next contract not found in multi-contract transfer")
	}

	node.Log(ctx, "Sending settlement offer to %s",
		transferTx.Outputs[nextContractIndex].Address.String(wire.BitcoinNet(w.Config.ChainParams.Net)))

	// Setup M1 response
	var err error
	err = w.SetUTXOs(ctx, []inspector.UTXO{itx.Outputs[boomerangIndex].UTXO})
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
	var data []byte
	data, err = settlementRequest.Serialize()
	if err != nil {
		return err
	}
	message := protocol.Message{
		AddressIndexes: []uint16{0}, // First output is receiver of message
		MessageType:    settlementRequest.Type(),
		MessagePayload: data,
	}

	if err := node.RespondSuccess(ctx, w, itx, rk, &message); err != nil {
		return err
	}

	if bytes.Equal(itx.Hash[:], transferTx.Hash[:]) {
		outpoint := wire.OutPoint{Hash: itx.Hash, Index: boomerangIndex}
		tracer.Add(ctx, &outpoint)
	}
	return nil
}

// settlementIsComplete returns true if the settlement accounts for all assets in the transfer.
func settlementIsComplete(ctx context.Context, transfer *protocol.Transfer, settlement *protocol.Settlement) bool {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.settlementIsComplete")
	defer span.End()

	for _, assetTransfer := range transfer.Assets {
		found := false
		for _, assetSettle := range settlement.Assets {
			if assetTransfer.AssetType == assetSettle.AssetType &&
				bytes.Equal(assetTransfer.AssetCode.Bytes(), assetSettle.AssetCode.Bytes()) {
				node.LogVerbose(ctx, "Found settlement data for asset : %s", assetTransfer.AssetCode.String())
				found = true
				break
			}
		}

		if !found {
			return false // No settlement for this asset yet
		}
	}

	return true
}

// SettlementResponse handles an outgoing Settlement action and writes it to the state
func (t *Transfer) SettlementResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.SettlementResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Settlement)
	if !ok {
		return errors.New("Could not assert as *protocol.Settlement")
	}

	if itx.RejectCode != 0 {
		return errors.New("Settlement response invalid")
	}

	txid := protocol.TxIdFromBytes(itx.Inputs[0].UTXO.Hash[:])
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	ct, err := contract.Retrieve(ctx, t.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	assetUpdates := make(map[protocol.AssetCode]map[protocol.PublicKeyHash]state.Holding)
	for _, assetSettlement := range msg.Assets {
		if assetSettlement.AssetType == "CUR" && assetSettlement.AssetCode.IsZero() {
			continue // Bitcoin transaction
		}

		hds := make(map[protocol.PublicKeyHash]state.Holding)
		assetUpdates[assetSettlement.AssetCode] = hds

		if assetSettlement.ContractIndex == 0xffff {
			continue // No contract for this asset
		}

		if int(assetSettlement.ContractIndex) >= len(itx.Inputs) {
			return fmt.Errorf("Settlement contract index out of range : %s", assetSettlement.AssetCode.String())
		}

		if !itx.Inputs[assetSettlement.ContractIndex].Address.Equal(rk.Address) {
			continue // Asset not under this contract
		}

		// Finalize settlements
		for _, settlementQuantity := range assetSettlement.Settlements {
			if int(settlementQuantity.Index) >= len(itx.Outputs) {
				return fmt.Errorf("Settlement output index out of range %d/%d : %s",
					settlementQuantity.Index, len(itx.Outputs), assetSettlement.AssetCode.String())
			}

			addressPKH, ok := itx.Outputs[settlementQuantity.Index].Address.(*bitcoin.AddressPKH)
			if !ok {
				return fmt.Errorf("Settlement output not PKH %d/%d : %s",
					settlementQuantity.Index, len(itx.Outputs), assetSettlement.AssetCode.String())
			}
			pkh := protocol.PublicKeyHashFromBytes(addressPKH.PKH())

			h, err := holdings.GetHolding(ctx, t.MasterDB, contractPKH, &assetSettlement.AssetCode,
				pkh, msg.Timestamp)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}

			err = holdings.FinalizeTx(&h, txid, msg.Timestamp)
			if err != nil {
				return fmt.Errorf("Failed settlement finalize for holding : %s %s : %s",
					assetSettlement.AssetCode.String(), pkh.String(), err)
			}

			hds[*pkh] = h
		}
	}

	for assetCode, hds := range assetUpdates {
		for _, h := range hds {
			if err := holdings.Save(ctx, t.MasterDB, contractPKH, &assetCode, &h); err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
		}
	}

	return nil
}

// respondTransferReject sends a reject to all parties involved with a transfer request and refunds
//   any bitcoin involved. This can only be done by the first contract, because they hold the
//   bitcoin to be distributed.
func respondTransferReject(ctx context.Context, masterDB *db.DB, config *node.Config, w *node.ResponseWriter,
	transferTx *inspector.Transaction, transfer *protocol.Transfer, rk *wallet.Key, code uint8, removeBoomerang bool) error {

	// Determine UTXOs to fund the reject response.
	utxos, err := transferTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Transfer UTXOs not found")
	}

	if removeBoomerang {
		// Remove utxo spent by boomerang
		boomerangIndex := findBoomerangIndex(transferTx, transfer, rk.Address)
		if boomerangIndex == 0xffffffff {
			return errors.New("Boomerang output index not found")
		}

		if transferTx.Outputs[boomerangIndex].Address.Equal(rk.Address) {
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
	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == protocol.CodeCurrency && assetTransfer.AssetCode.IsZero() {
			// Process bitcoin senders refunds
			for _, sender := range assetTransfer.AssetSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				node.LogVerbose(ctx, "Bitcoin refund %d : %s", sender.Quantity,
					transferTx.Inputs[sender.Index].Address.String(wire.BitcoinNet(w.Config.ChainParams.Net)))
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
		}
	}

	if refundBalance > balance {
		contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
		ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}

		// Funding not enough to refund everyone, so don't refund to anyone. Send it to the administration to hold.
		administrationAddress, err := bitcoin.NewAddressPKH(ct.AdministrationPKH.Bytes())
		if err != nil {
			return errors.Wrap(err, "Failed to create admin address")
		}
		w.ClearRejectOutputValues(administrationAddress)
	}

	return node.RespondReject(ctx, w, transferTx, rk, code)
}
