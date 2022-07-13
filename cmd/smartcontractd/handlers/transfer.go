package handlers

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/instrument"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
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
	Tracer          *filters.Tracer
	Scheduler       *scheduler.Scheduler
	HoldingsChannel *holdings.CacheChannel
}

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferRequest")
	defer span.End()
	start := time.Now()

	node.Log(ctx, "Transfer request")

	v := ctx.Value(node.KeyValues).(*node.Values)

	msg, ok := itx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Could not assert as *actions.Transfer")
	}

	// Check pre-processing reject code
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Transfer request invalid")
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			itx.RejectCode, false, itx.RejectText)
	}

	// Find "first" contract.
	first := firstContractOutputIndex(msg.Instruments, itx)

	if first == 0xffff {
		node.LogWarn(ctx, "Transfer first contract not found : %s",
			bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net))
		return errors.New("Transfer first contract not found")
	}

	if !itx.Outputs[first].Address.Equal(rk.Address) {
		node.Log(ctx, "Not contract for first transfer. Waiting for Message SettlementRequest : %s",
			bitcoin.NewAddressFromRawAddress(itx.Outputs[first].Address, w.Config.Net))
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	if msg.OfferExpiry != 0 && v.Now.Nano() > msg.OfferExpiry {
		node.LogWarn(ctx, "Transfer expired : %d", msg.OfferExpiry)
		return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
			actions.RejectionsTransferExpired, false, "Transfer expired")
	}

	if len(msg.Instruments) == 0 {
		node.LogWarn(ctx, "Transfer has no instrument transfers")
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
	//   Contract 1 : amount = calculated fee for settlement tx + contract fees + any bitcoins being
	//   transfered
	//   Contract 2 : contract fees if applicable or dust
	//   Boomerang to Contract 1 : amount = ((n-1) * 2) * (calculated fee for data passing tx)
	//     where n is number of contracts involved
	// Boomerang is only required when more than one contract is involved.
	// It is defined as an output from the transfer tx, that pays to the first contract of the
	//   transfer, but it's index is not referenced/spent by any of the instrument transfers of the
	//   transfer tx.
	// The first contract is defined by the first valid contract index of a transfer. Some of the
	//   transfers will not reference a contract, like a bitcoin transfer.
	//
	// Transfer Inputs
	//   Any addresses sending tokens or bitcoin.
	//
	// Each contract can be involved in more than one instrument in the transfer, but only needs to have
	//   one output since each instrument transfer references the output of it's contract
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
	// addSettlementData.
	var script []byte
	script, err = protocol.Serialize(&settlement, t.Config.IsTest)
	if err != nil {
		node.LogWarn(ctx, "Failed to serialize settlement : %s", err)
		return errors.Wrap(err, "serialize response")
	}

	if err := settleTx.AddOutput(script, 0, false, false); err != nil {
		return errors.Wrap(err, "add response output")
	}

	// Add this contract's data to the settlement op return data
	isSingleContract := transferIsSingleContract(ctx, itx, msg, rk)
	instrumentUpdates := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding)
	senderCount, receiverCount, err := addSettlementData(ctx, t.MasterDB, t.Config, rk, itx, msg,
		settleTx, &settlement, &instrumentUpdates, isSingleContract)
	if err != nil {
		rejectCode, ok := node.ErrorCode(err)
		if ok {
			node.LogWarn(ctx, "Rejecting Transfer : %s", err)
			return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg,
				rk, rejectCode, false, "")
		} else {
			return errors.Wrap(err, "add settlement data")
		}
	}

	// Check if settlement data is complete. No other contracts involved
	if isSingleContract {
		if _, err := settleTx.Sign([]bitcoin.Key{rk.Key}); err != nil {
			if errors.Cause(err) == txbuilder.ErrInsufficientValue {
				node.LogWarn(ctx, "Insufficient settlement tx funding : %s", err)
				return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx,
					msg, rk, actions.RejectionsInsufficientTxFeeFunding, false, err.Error())
			} else {
				return errors.Wrap(err, "sign response")
			}
		}

		responseItx, err := inspector.NewTransactionFromTxBuilder(ctx, settleTx, t.Config.IsTest)
		if err != nil {
			return errors.Wrap(err, "inspector from builder")
		}

		logger.InfoWithFields(ctx, []logger.Field{
			logger.Int("sender_count", senderCount),
			logger.Int("receiver_count", receiverCount),
			logger.MillisecondsFromNano("elapsed_ms", time.Since(start).Nanoseconds()),
		}, "Single contract settlement complete")

		if err := node.Respond(ctx, w, responseItx); err == nil {
			if err = saveHoldings(ctx, t.MasterDB, t.HoldingsChannel, instrumentUpdates,
				rk.Address); err != nil {
				return errors.Wrap(err, "save holdings")
			}
		}

		return errors.Wrap(err, "send response")
	}

	// Send to next contract
	if err := sendToNextSettlementContract(ctx, w, rk, itx, itx, msg, settleTx, &settlement,
		&settlementRequest, t.Tracer); err != nil {
		return errors.Wrap(err, "send to next contract")
	}

	// Save pending transfer
	timeout := protocol.NewTimestamp(v.Now.Nano() + t.Config.RequestTimeout)
	pendingTransfer := state.PendingTransfer{
		TransferTxId: itx.Hash,
		Timeout:      timeout,
	}
	if err := transfer.Save(ctx, t.MasterDB, rk.Address, &pendingTransfer); err != nil {
		return errors.Wrap(err, "Failed to save pending transfer")
	}

	// Schedule timeout for transfer in case the other contract(s) don't respond.
	if err := t.Scheduler.ScheduleJob(ctx, listeners.NewTransferTimeout(t.handler, itx,
		timeout)); err != nil {
		return errors.Wrap(err, "Failed to schedule transfer timeout")
	}

	if err := saveHoldings(ctx, t.MasterDB, t.HoldingsChannel, instrumentUpdates,
		rk.Address); err != nil {
		return errors.Wrap(err, "save holdings")
	}

	return nil
}

func saveHoldings(ctx context.Context, masterDB *db.DB, holdingsChannel *holdings.CacheChannel,
	updates map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding,
	contractAddress bitcoin.RawAddress) error {

	for instrumentCode, hds := range updates {
		for _, h := range *hds {
			cacheItem, err := holdings.Save(ctx, masterDB, contractAddress, &instrumentCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
			holdingsChannel.Add(cacheItem)
		}
	}

	return nil
}

func revertHoldings(ctx context.Context, masterDB *db.DB, holdingsChannel *holdings.CacheChannel,
	updates map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding,
	contractAddress bitcoin.RawAddress, txid *bitcoin.Hash32) error {

	for _, hds := range updates {
		for _, h := range *hds {
			if err := holdings.RevertStatus(h, txid); err != nil {
				return errors.Wrap(err, "Failed to revert holding status")
			}
		}
	}

	return saveHoldings(ctx, masterDB, holdingsChannel, updates, contractAddress)
}

// TransferTimeout is called when a multi-contract transfer times out because the other contracts
// are not responding.
func (t *Transfer) TransferTimeout(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferTimeout")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Could not assert as *actions.Transfer")
	}

	// Remove pending transfer
	if err := transfer.Remove(ctx, t.MasterDB, rk.Address, itx.Hash); err != nil {
		if err != transfer.ErrNotFound {
			return errors.Wrap(err, "Failed to remove pending transfer")
		}
	}

	// Remove tracer for this transfer.
	boomerangIndex := findBoomerangIndex(itx, msg, rk.Address)
	if boomerangIndex != 0xffffffff {
		outpoint := wire.OutPoint{Hash: *itx.Hash, Index: boomerangIndex}
		t.Tracer.Remove(ctx, &outpoint)
	}

	node.LogWarn(ctx, "Transfer timed out")
	return respondTransferReject(ctx, t.MasterDB, t.HoldingsChannel, t.Config, w, itx, msg, rk,
		actions.RejectionsTimeout, true, "")
}

// firstContractOutputIndex finds the "first" contract. The "first" contract of a transfer is the
// one responsible for creating the initial settlement data and passing it to the next contract if
// there are more than one.
func firstContractOutputIndex(instrumentTransfers []*actions.InstrumentTransferField,
	itx *inspector.Transaction) uint32 {

	for _, instrument := range instrumentTransfers {
		if instrument.InstrumentType != protocol.BSVInstrumentID && len(instrument.InstrumentCode) != 0 &&
			int(instrument.ContractIndex) < len(itx.Outputs) {
			return instrument.ContractIndex
		}
	}

	return 0x0000ffff
}

// buildSettlementTx builds the tx for a settlement action.
func buildSettlementTx(ctx context.Context, masterDB *db.DB, config *node.Config,
	transferTx *inspector.Transaction, transfer *actions.Transfer,
	settlementRequest *messages.SettlementRequest, contractBalance uint64,
	rk *wallet.Key) (*txbuilder.TxBuilder, error) {
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
	for instrumentOffset, instrumentTransfer := range transfer.Instruments {
		if instrumentTransfer.ContractIndex == uint32(0x0000ffff) ||
			(instrumentTransfer.InstrumentType == protocol.BSVInstrumentID && len(instrumentTransfer.InstrumentCode) == 0) {
			continue
		}

		if int(instrumentTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return nil, fmt.Errorf("Transfer contract index out of range %d", instrumentOffset)
		}

		if outputUsed[instrumentTransfer.ContractIndex] {
			continue
		}

		// Add input from contract to settlement tx so all involved contracts have to sign for a valid tx.
		err = settleTx.AddInput(wire.OutPoint{Hash: *transferTx.Hash, Index: uint32(instrumentTransfer.ContractIndex)},
			transferTx.Outputs[instrumentTransfer.ContractIndex].UTXO.LockingScript,
			transferTx.Outputs[instrumentTransfer.ContractIndex].UTXO.Value)
		if err != nil {
			return nil, err
		}
		outputUsed[instrumentTransfer.ContractIndex] = true
	}

	// Setup outputs
	//   One to each receiver, including any bitcoins received, or dust.
	//   One to each sender with dust amount.
	for instrumentOffset, instrumentTransfer := range transfer.Instruments {
		instrumentIsBitcoin := instrumentTransfer.InstrumentType == protocol.BSVInstrumentID &&
			len(instrumentTransfer.InstrumentCode) == 0
		instrumentBalance := uint64(0)

		// Add all senders
		for senderOffset, quantityIndex := range instrumentTransfer.InstrumentSenders {
			instrumentBalance += quantityIndex.Quantity

			if quantityIndex.Index >= uint32(len(transferTx.Inputs)) {
				return nil, fmt.Errorf("Transfer sender index out of range %d", instrumentOffset)
			}

			input := transferTx.Inputs[quantityIndex.Index]

			if instrumentIsBitcoin {
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
		for _, instrumentReceiver := range instrumentTransfer.InstrumentReceivers {
			if instrumentReceiver.Quantity > instrumentBalance {
				return nil, fmt.Errorf("Sending more than received")
			}

			instrumentBalance -= instrumentReceiver.Quantity

			if instrumentIsBitcoin {
				// Debit from contract's bitcoin balance
				if instrumentReceiver.Quantity > contractBalance {
					return nil, fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= instrumentReceiver.Quantity
			}

			receiverAddress, err = bitcoin.DecodeRawAddress(instrumentReceiver.Address)
			if err != nil {
				return nil, err
			}
			hash, err := receiverAddress.Hash()
			if err != nil {
				return nil, errors.Wrap(err, "Transfer receiver address invalid")
			}
			outputIndex, exists := addresses[*hash]
			if exists {
				if instrumentIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, instrumentReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver's address
				addresses[*hash] = uint32(len(settleTx.MsgTx.TxOut))
				if instrumentIsBitcoin {
					err = settleTx.AddPaymentOutput(receiverAddress, instrumentReceiver.Quantity, false)
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
	settlement *actions.Settlement,
	updates *map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding, isSingleContract bool) (int, int, error) {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addSettlementData")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	dataAdded := false
	var senderCount, receiverCount int

	ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
	if err != nil {
		return 0, 0, errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		return 0, 0, node.NewError(actions.RejectionsContractFrozen, "")
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
		address, err := bitcoin.RawAddressFromLockingScript(output.LockingScript)
		if err != nil {
			settleOutputAddresses = append(settleOutputAddresses, bitcoin.RawAddress{})
			continue
		}
		settleOutputAddresses = append(settleOutputAddresses, address)
	}

	for instrumentOffset, instrumentTransfer := range transfer.Instruments {
		if instrumentTransfer.InstrumentType == protocol.BSVInstrumentID && len(instrumentTransfer.InstrumentCode) == 0 {
			node.Log(ctx, "Instrument transfer for bitcoin")
			continue // Skip bitcoin transfers since they should be handled already
		}

		if int(instrumentTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return 0, 0, fmt.Errorf("Contract index %d out of range : %d >= %d", instrumentOffset,
				instrumentTransfer.ContractIndex, len(transferTx.Outputs))
		}

		contractOutputAddress := transferOutputAddresses[instrumentTransfer.ContractIndex]
		if contractOutputAddress.IsEmpty() || !contractOutputAddress.Equal(rk.Address) {
			continue // This instrument is not ours. Skip it.
		}

		instrumentCode, err := bitcoin.NewHash20(instrumentTransfer.InstrumentCode)
		if err != nil {
			return 0, 0, errors.Wrap(err, "invalid instrument code")
		}
		instrumentID := protocol.InstrumentID(instrumentTransfer.InstrumentType, *instrumentCode)

		// Locate Instrument
		as, err := instrument.Retrieve(ctx, masterDB, rk.Address, instrumentCode)
		if err != nil {
			return 0, 0, fmt.Errorf("Instrument ID not found : %s : %s", instrumentID, err)
		}

		if err := instrument.IsTransferable(ctx, as, v.Now); err != nil {
			return 0, 0, err
		}

		transfersPermitted := as.TransfersPermitted()

		// Find contract input
		contractInputIndex := uint32(0x0000ffff)
		for i, input := range settleInputAddresses {
			if !input.IsEmpty() && input.Equal(rk.Address) {
				contractInputIndex = uint32(i)
				break
			}
		}

		if contractInputIndex == uint32(0x0000ffff) {
			return 0, 0, fmt.Errorf("Contract input not found: %s", instrumentID)
		}

		node.Log(ctx, "Adding settlement data for instrument : %s", instrumentID)
		instrumentSettlement := actions.InstrumentSettlementField{
			ContractIndex:  contractInputIndex,
			InstrumentType: instrumentTransfer.InstrumentType,
			InstrumentCode: instrumentTransfer.InstrumentCode,
		}

		sendBalance := uint64(0)
		fromNonAdministration := uint64(0)
		fromAdministration := uint64(0)
		toNonAdministration := uint64(0)
		toAdministration := uint64(0)
		hds := make([]*state.Holding, len(settleTx.Outputs))
		updatedHoldings := make(map[bitcoin.Hash20]*state.Holding)
		(*updates)[*instrumentCode] = &updatedHoldings

		// Process senders
		// instrumentTransfer.InstrumentSenders []QuantityIndex {Index uint16, Quantity uint64}
		senderCount += len(instrumentTransfer.InstrumentSenders)
		for senderOffset, sender := range instrumentTransfer.InstrumentSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return 0, 0, fmt.Errorf("Sender input index out of range for instrument %d sender %d : %d/%d",
					instrumentOffset, senderOffset, sender.Index, len(transferTx.Inputs))
			}

			if transferTx.Inputs[sender.Index].Address.Equal(ct.AdminAddress) {
				fromAdministration += sender.Quantity
			} else {
				fromNonAdministration += sender.Quantity
			}

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if !outputAddress.IsEmpty() &&
					outputAddress.Equal(transferTx.Inputs[sender.Index].Address) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return 0, 0, fmt.Errorf("Sender output not found in settle tx for instrument %d sender %d : %d/%d",
					instrumentOffset, senderOffset, sender.Index, len(transferTx.Outputs))
			}

			// Check sender's available unfrozen balance
			if hds[settleOutputIndex] != nil {
				address := bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
					config.Net)
				node.LogWarn(ctx, "Duplicate sender entry: instrument=%s party=%s", instrumentID, address)
				return 0, 0, node.NewError(actions.RejectionsMsgMalformed, "")
			}

			h, err := holdings.GetHolding(ctx, masterDB, rk.Address, instrumentCode,
				transferTx.Inputs[sender.Index].Address, v.Now)
			if err != nil {
				return 0, 0, errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = h
			hash, err := transferTx.Inputs[sender.Index].Address.Hash()
			if err != nil {
				return 0, 0, errors.Wrap(err, "Invalid sender address")
			}
			updatedHoldings[*hash] = h

			address := bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
				config.Net)
			if err := holdings.AddDebit(h, transferTx.Hash, sender.Quantity, isSingleContract, v.Now); err != nil {
				if err == holdings.ErrInsufficientHoldings {
					node.LogWarn(ctx, "Insufficient funds: instrument=%s party=%s : %d/%d", instrumentID,
						address.String(), sender.Quantity, holdings.SafeBalance(h))
					return 0, 0, node.NewError(actions.RejectionsInsufficientQuantity, "")
				}
				if err == holdings.ErrHoldingsFrozen {
					node.LogWarn(ctx, "Frozen funds: instrument=%s party=%s", instrumentID, address)
					return 0, 0, node.NewError(actions.RejectionsHoldingsFrozen, "")
				}
				if err == holdings.ErrHoldingsLocked {
					node.LogWarn(ctx, "Locked funds: instrument=%s party=%s", instrumentID, address)
					return 0, 0, node.NewError(actions.RejectionsHoldingsLocked, "")
				}
				node.LogWarn(ctx, "Send failed : %s : instrument=%s party=%s", err, instrumentID, address)
				return 0, 0, node.NewError(actions.RejectionsMsgMalformed, "")
			} else {
				logger.Info(ctx, "Debit %d %s to %s", sender.Quantity, instrumentID, address)
			}

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		receiverCount += len(instrumentTransfer.InstrumentReceivers)
		for receiverOffset, receiver := range instrumentTransfer.InstrumentReceivers {
			receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
			if err != nil {
				return 0, 0, err
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
				return 0, 0, fmt.Errorf("Receiver output not found in settle tx for instrument %d receiver %d : %s",
					instrumentOffset, receiverOffset, address)
			}

			if receiverAddress.Equal(ct.AdminAddress) {
				toAdministration += receiver.Quantity
			} else {
				toNonAdministration += receiver.Quantity
			}

			if hds[settleOutputIndex] != nil {
				address := bitcoin.NewAddressFromRawAddress(receiverAddress,
					config.Net)
				node.LogWarn(ctx, "Duplicate receiver entry: instrument=%s party=%s", instrumentID, address)
				return 0, 0, node.NewError(actions.RejectionsMsgMalformed, "")
			}

			h, err := holdings.GetHolding(ctx, masterDB, rk.Address, instrumentCode, receiverAddress,
				v.Now)
			if err != nil {
				return 0, 0, errors.Wrap(err, "Failed to get holding")
			}
			hds[settleOutputIndex] = h
			hash, err := receiverAddress.Hash()
			if err != nil {
				return 0, 0, errors.Wrap(err, "Invalid receiver address")
			}
			updatedHoldings[*hash] = h

			address := bitcoin.NewAddressFromRawAddress(receiverAddress, config.Net)
			if err := holdings.AddDeposit(h, transferTx.Hash, receiver.Quantity, isSingleContract,
				v.Now); err != nil {
				if err == holdings.ErrHoldingsLocked {
					node.LogWarn(ctx, "Locked funds: instrument=%s party=%s", instrumentID, address)
					return 0, 0, node.NewError(actions.RejectionsHoldingsLocked, "")
				}
				node.LogWarn(ctx, "Send failed : %s : instrument=%s party=%s", err, instrumentID, address)
				return 0, 0, node.NewError(actions.RejectionsMsgMalformed, "")
			} else {
				logger.Info(ctx, "Deposit %d %s to %s", receiver.Quantity, instrumentID, address)
			}

			// Update instrument balance
			if receiver.Quantity > sendBalance {
				return 0, 0, fmt.Errorf("Receiving more tokens than sending for instrument %d", instrumentOffset)
			}
			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return 0, 0, fmt.Errorf("Not sending all input tokens for instrument %d : %d remaining",
				instrumentOffset, sendBalance)
		}

		if !transfersPermitted {
			if fromNonAdministration > toAdministration {
				node.LogWarn(ctx, "Transfers not permitted. Sending tokens not all to administration : %d/%d",
					fromNonAdministration, toAdministration)
				return 0, 0, node.NewError(actions.RejectionsInstrumentNotPermitted, "")
			}
			if toNonAdministration > fromAdministration {
				node.LogWarn(ctx, "Transfers not permitted. Receiving tokens not all from administration : %d/%d",
					toNonAdministration, fromAdministration)
				return 0, 0, node.NewError(actions.RejectionsInstrumentNotPermitted, "")
			}
		}

		for index, holding := range hds {
			if holding != nil {
				instrumentSettlement.Settlements = append(instrumentSettlement.Settlements,
					&actions.QuantityIndexField{
						Index:    uint32(index),
						Quantity: holding.PendingBalance,
					})
			}
		}

		// Check if settlement already exists for this instrument.
		replaced := false
		for i, instrument := range settlement.Instruments {
			if instrument.InstrumentType == instrumentSettlement.InstrumentType &&
				bytes.Equal(instrument.InstrumentCode, instrumentSettlement.InstrumentCode) {
				replaced = true
				settlement.Instruments[i] = &instrumentSettlement
				break
			}
		}

		if !replaced {
			settlement.Instruments = append(settlement.Instruments, &instrumentSettlement) // Append
		}
		dataAdded = true
	}

	if !dataAdded {
		return 0, 0, errors.New("No data added to settlement")
	}

	// Serialize settlement data back into OP_RETURN output.
	script, err := protocol.Serialize(settlement, config.IsTest)
	if err != nil {
		return 0, 0, fmt.Errorf("Failed to serialize empty settlement : %s", err)
	}

	// Find Settlement OP_RETURN.
	found := false
	settlementOutputIndex := 0
	for i, output := range settleTx.MsgTx.TxOut {
		action, err := protocol.Deserialize(output.LockingScript, config.IsTest)
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
		return 0, 0, fmt.Errorf("Settlement op return not found in settle tx")
	}

	settleTx.MsgTx.TxOut[settlementOutputIndex].LockingScript = script
	return senderCount, receiverCount, nil
}

// findBoomerangIndex returns the index to the "boomerang" output from transfer tx. It is the
//   output to the contract that is not referenced/spent by the transfers. It is used to fund the
//   offer and signature request messages required between multiple contracts to get a fully
//   approved settlement tx.
func findBoomerangIndex(transferTx *inspector.Transaction,
	transfer *actions.Transfer,
	contractAddress bitcoin.RawAddress) uint32 {

	outputUsed := make([]bool, len(transferTx.Outputs))
	for _, instrumentTransfer := range transfer.Instruments {
		if instrumentTransfer.ContractIndex == uint32(0x0000ffff) ||
			(instrumentTransfer.InstrumentType == protocol.BSVInstrumentID && len(instrumentTransfer.InstrumentCode) == 0) {
			continue
		}

		if int(instrumentTransfer.ContractIndex) >= len(transferTx.Outputs) {
			return 0xffffffff
		}

		// Output will be spent by settlement tx.
		outputUsed[instrumentTransfer.ContractIndex] = true
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
	for _, instrument := range transfer.Instruments {
		if instrument.ContractIndex == uint32(0x0000ffff) {
			continue // Instrument transfer doesn't have a contract (probably BSV transfer).
		}

		if int(instrument.ContractIndex) >= len(transferTx.Outputs) {
			return errors.New("Transfer contract index out of range")
		}

		hash, err := transferTx.Outputs[instrument.ContractIndex].Address.Hash()
		if err != nil {
			return errors.Wrap(err, "Transfer contract address invalid")
		}

		if !currentFound {
			completedContracts[*hash] = true
			if transferTx.Outputs[instrument.ContractIndex].Address.Equal(rk.Address) {
				currentFound = true
			}
			continue
		}

		// Contracts can be used more than once, so ensure this contract wasn't referenced before
		//   the current contract.
		_, complete := completedContracts[*hash]
		if !complete {
			nextContractIndex = instrument.ContractIndex
			break
		}
	}

	if nextContractIndex == 0xffff {
		return fmt.Errorf("Next contract not found in multi-contract transfer")
	}

	node.Log(ctx, "Sending settlement offer to %s",
		bitcoin.NewAddressFromRawAddress(transferTx.Outputs[nextContractIndex].Address,
			w.Config.Net))

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

// transferIsSingleContract returns true if this contract can settle all instruments in the transfer.
func transferIsSingleContract(ctx context.Context, itx *inspector.Transaction,
	transfer *actions.Transfer, rk *wallet.Key) bool {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.transferIsSingleContract")
	defer span.End()

	for _, instrumentTransfer := range transfer.Instruments {
		if instrumentTransfer.InstrumentType == protocol.BSVInstrumentID {
			continue // All contracts can handle bitcoin transfers
		}

		if int(instrumentTransfer.ContractIndex) >= len(itx.Outputs) {
			return false // Invalid contract index
		}

		if !itx.Outputs[instrumentTransfer.ContractIndex].Address.Equal(rk.Address) {
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

	node.Log(ctx, "Settlement response")

	msg, ok := itx.MsgProto.(*actions.Settlement)
	if !ok {
		return errors.New("Could not assert as *actions.Settlement")
	}

	ct, err := contract.Retrieve(ctx, t.MasterDB, rk.Address, t.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	instrumentUpdates := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding)
	for instrumentIndex, instrumentSettlement := range msg.Instruments {
		if instrumentSettlement.InstrumentType == protocol.BSVInstrumentID && len(instrumentSettlement.InstrumentCode) == 0 {
			continue // Bitcoin transaction
		}

		if instrumentSettlement.ContractIndex == 0x0000ffff {
			continue // No contract for this instrument
		}

		if int(instrumentSettlement.ContractIndex) >= len(itx.Inputs) {
			return fmt.Errorf("Settlement contract index %d out of range : %d >= %d", instrumentIndex,
				instrumentSettlement.ContractIndex, len(itx.Inputs))
		}

		if !itx.Inputs[instrumentSettlement.ContractIndex].Address.Equal(rk.Address) {
			continue // Instrument not under this contract
		}

		instrumentCode, err := bitcoin.NewHash20(instrumentSettlement.InstrumentCode)
		if err != nil {
			return errors.Wrap(err, "invalid instrument code")
		}
		instrumentID := protocol.InstrumentID(instrumentSettlement.InstrumentType, *instrumentCode)

		hds := make(map[bitcoin.Hash20]*state.Holding)
		instrumentUpdates[*instrumentCode] = &hds

		timestamp := protocol.NewTimestamp(msg.Timestamp)

		// Finalize settlements
		for _, settlementQuantity := range instrumentSettlement.Settlements {
			if int(settlementQuantity.Index) >= len(itx.Outputs) {
				return fmt.Errorf("Settlement output index out of range %d >= %d : %s",
					settlementQuantity.Index, len(itx.Outputs), instrumentID)
			}

			h, err := holdings.GetHolding(ctx, t.MasterDB, rk.Address, instrumentCode,
				itx.Outputs[settlementQuantity.Index].Address, timestamp)
			if err != nil {
				return errors.Wrap(err, "Failed to get holding")
			}

			ra := itx.Outputs[settlementQuantity.Index].Address
			address := bitcoin.NewAddressFromRawAddress(ra, w.Config.Net)
			if err = holdings.FinalizeTx(h, &itx.Inputs[0].UTXO.Hash, settlementQuantity.Quantity,
				timestamp); err != nil {
				return fmt.Errorf("Failed settlement finalize for holding : %s %s : %s", instrumentID,
					address, err)
			}

			logger.Info(ctx, "Settled %s balance of %d for %s", instrumentID, h.FinalizedBalance,
				address)

			hash, err := itx.Outputs[settlementQuantity.Index].Address.Hash()
			if err != nil {
				return errors.Wrap(err, "Invalid settlement address")
			}
			hds[*hash] = h
		}
	}

	// Now that no errors were found we can save all the data.
	for instrumentCode, hds := range instrumentUpdates {
		for _, h := range *hds {
			cacheItem, err := holdings.Save(ctx, t.MasterDB, rk.Address, &instrumentCode, h)
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

	// Determine UTXOs to fund the reject response.
	utxos, err := transferTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		node.LogWarn(ctx, "Transfer UTXOs not found")
		return nil
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
				node.LogWarn(ctx, "Boomerang output not found")
				return nil
			}
		}
	}

	balance := uint64(0)
	for _, utxo := range utxos {
		balance += uint64(utxo.Value)
	}

	updates := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*state.Holding)

	w.SetRejectUTXOs(ctx, utxos)

	// Add refund amounts for all bitcoin senders (if "first" contract, first contract receives bitcoin funds to be distributed)
	first := firstContractOutputIndex(transfer.Instruments, transferTx)
	if first == 0xffff {
		node.LogWarn(ctx, "First contract output index not found")
		return nil
	}

	// Determine if this contract is the first contract and needs to send a refund.
	if !transferTx.Outputs[first].Address.Equal(rk.Address) {
		node.LogWarn(ctx, "This is not the first contract")
		return nil
	}

	refundBalance := uint64(0)
	for instrumentOffset, instrumentTransfer := range transfer.Instruments {
		if instrumentTransfer.InstrumentType == protocol.BSVInstrumentID && len(instrumentTransfer.InstrumentCode) == 0 {
			// Process bitcoin senders refunds
			for _, sender := range instrumentTransfer.InstrumentSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				node.LogVerbose(ctx, "Bitcoin refund %d : %s", sender.Quantity,
					bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
						w.Config.Net))
				w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, sender.Quantity)
				refundBalance += sender.Quantity
			}
		} else {
			// Add all other senders to be notified
			for _, sender := range instrumentTransfer.InstrumentSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, 0)
			}

			if started { // Revert holding statuses
				if len(transferTx.Outputs) <= int(instrumentTransfer.ContractIndex) {
					node.LogWarn(ctx, "Contract index out of range for instrument %d", instrumentOffset)
					return nil
				}

				if !transferTx.Outputs[instrumentTransfer.ContractIndex].Address.Equal(rk.Address) {
					continue // This instrument is not ours. Skip it.
				}

				instrumentCode, err := bitcoin.NewHash20(instrumentTransfer.InstrumentCode)
				if err != nil {
					node.LogWarn(ctx, "Invalid instrument code %d", instrumentOffset)
					return nil
				}
				updatedHoldings := make(map[bitcoin.Hash20]*state.Holding)
				updates[*instrumentCode] = &updatedHoldings

				// Revert sender pending statuses
				for _, sender := range instrumentTransfer.InstrumentSenders {
					// Revert holding status
					h, err := holdings.GetHolding(ctx, masterDB, rk.Address, instrumentCode,
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
					err = holdings.RevertStatus(h, transferTx.Hash)
					if err != nil {
						return errors.Wrap(err, "revert status")
					}
				}

				// Revert receiver pending statuses
				for _, receiver := range instrumentTransfer.InstrumentReceivers {
					receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
					if err != nil {
						node.LogWarn(ctx, "Invalid receiver address : %s", err)
						return nil
					}

					h, err := holdings.GetHolding(ctx, masterDB, rk.Address, instrumentCode,
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
					err = holdings.RevertStatus(h, transferTx.Hash)
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
