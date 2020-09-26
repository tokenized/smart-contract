package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
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
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/messages"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Message struct {
	MasterDB        *db.DB
	Config          *node.Config
	Headers         node.BitcoinHeaders
	Tracer          *filters.Tracer
	Scheduler       *scheduler.Scheduler
	UTXOs           *utxos.UTXOs
	HoldingsChannel *holdings.CacheChannel
}

// ProcessMessage handles an incoming Message OP_RETURN.
func (m *Message) ProcessMessage(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Message.ProcessMessage")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Message)
	if !ok {
		return errors.New("Could not assert as *actions.Message")
	}

	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Message invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Check if message is addressed to contract.
	found := false
	if len(msg.ReceiverIndexes) == 0 {
		found = itx.Outputs[0].Address.Equal(rk.Address)
	} else {
		for _, outputIndex := range msg.ReceiverIndexes {
			if int(outputIndex) >= len(itx.Outputs) {
				return fmt.Errorf("Message output index out of range : %d/%d", outputIndex, len(itx.Outputs))
			}

			if itx.Outputs[outputIndex].Address.Equal(rk.Address) {
				found = true
				break
			}
		}
	}

	if !found {
		node.Log(ctx, "Message not addressed to this contract")
		return nil // Message not addressed to contract.
	}

	messagePayload, err := messages.Deserialize(msg.MessageCode, msg.MessagePayload)
	if err != nil {
		return errors.Wrap(err, "Failed to deserialize message payload")
	}

	if err := messagePayload.Validate(); err != nil {
		node.LogWarn(ctx, "Message %04d payload is invalid : %s", msg.MessageCode, err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	switch payload := messagePayload.(type) {
	case *messages.SettlementRequest:
		node.LogVerbose(ctx, "Processing Settlement Request")
		return m.processSettlementRequest(ctx, w, itx, payload, rk)
	case *messages.SignatureRequest:
		node.LogVerbose(ctx, "Processing Signature Request")
		return m.processSigRequest(ctx, w, itx, payload, rk)
	default:
		return fmt.Errorf("Unknown message payload type : %04d", msg.MessageCode)
	}
}

// ProcessRejection handles an incoming Rejection OP_RETURN.
func (m *Message) ProcessRejection(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Message.ProcessRejection")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Rejection)
	if !ok {
		return errors.New("Could not assert as *actions.Rejection")
	}

	// Check if message is addressed to this contract.
	found := false
	if len(msg.AddressIndexes) == 0 {
		found = itx.Outputs[0].Address.Equal(rk.Address)
	} else {
		for _, outputIndex := range msg.AddressIndexes {
			if int(outputIndex) >= len(itx.Outputs) {
				return fmt.Errorf("Reject message output index out of range : %d/%d", outputIndex,
					len(itx.Outputs))
			}

			if itx.Outputs[outputIndex].Address.Equal(rk.Address) {
				found = true
				break
			}
		}
	}

	if !found {
		node.Log(ctx, "Reject message not addressed to this contract")
		return nil // Message not addressed to this contract.
	}

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		node.LogWarn(ctx, "Reject invalid : %s", err)
		return errors.Wrap(err, "Invalid rejection tx")
	}

	node.LogWarn(ctx, "Rejection received (%d) : %s", msg.RejectionCode, msg.Message)

	// Trace back to original request tx if necessary.
	hash := m.Tracer.Retrace(ctx, itx.MsgTx)
	var problemTx *inspector.Transaction
	var err error
	if hash != nil {
		problemTx, err = transactions.GetTx(ctx, m.MasterDB, hash, m.Config.IsTest)
	} else {
		problemTx, err = transactions.GetTx(ctx, m.MasterDB, &itx.Inputs[0].UTXO.Hash, m.Config.IsTest)
	}
	if err != nil {
		return nil
	}

	switch problemMsg := problemTx.MsgProto.(type) {
	case *actions.Transfer:
		// Refund any funds from the transfer tx that were sent to the this contract.
		return refundTransferFromReject(ctx, m.Config, m.MasterDB, m.HoldingsChannel, m.Scheduler,
			w, itx, msg, problemTx, problemMsg, rk)

	default:
	}

	return nil
}

// ProcessRevert handles a tx that has been reverted either through a reorg or zero conf double spend.
func (m *Message) ProcessRevert(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Message.ProcessRevert")
	defer span.End()

	// Serialize tx for Message OP_RETURN.
	var txBuf bytes.Buffer
	err := itx.MsgTx.Serialize(&txBuf)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize revert tx")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	messagePayload := messages.RevertedTx{
		Timestamp:   v.Now.Nano(),
		Transaction: txBuf.Bytes(),
	}

	// Setup Message
	var payBuf bytes.Buffer
	err = messagePayload.Serialize(&payBuf)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize revert payload")
	}
	message := actions.Message{
		ReceiverIndexes: []uint32{0}, // First receiver is administration
		MessageCode:     messagePayload.Code(),
		MessagePayload:  payBuf.Bytes(),
	}

	ct, err := contract.Retrieve(ctx, m.MasterDB, rk.Address, m.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Create tx
	tx := txbuilder.NewTxBuilder(m.Config.FeeRate, m.Config.DustFeeRate)
	tx.SetChangeAddress(rk.Address, "")

	// Add outputs to administration/operator
	tx.AddDustOutput(ct.AdminAddress, false)
	outputAmount := tx.MsgTx.TxOut[len(tx.MsgTx.TxOut)-1].Value
	if !ct.OperatorAddress.IsEmpty() {
		// Add operator
		tx.AddDustOutput(ct.OperatorAddress, false)
		message.ReceiverIndexes = append(message.ReceiverIndexes, uint32(1))
		outputAmount += tx.MsgTx.TxOut[len(tx.MsgTx.TxOut)-1].Value
	}

	// Serialize payload
	payload, err := protocol.Serialize(&message, m.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize revert message")
	}
	tx.AddOutput(payload, 0, false, false)

	// Estimate fee with 2 inputs
	amount := tx.EstimatedFee() + outputAmount + (2 * txbuilder.MaximumP2PKHInputSize)

	for {
		utxos, err := m.UTXOs.Get(amount, rk.Address)
		if err != nil {
			return errors.Wrap(err, "Failed to get UTXOs")
		}

		for _, utxo := range utxos {
			if err := tx.AddInput(utxo.OutPoint, utxo.Output.PkScript, uint64(utxo.Output.Value)); err != nil {
				return errors.Wrap(err, "Failed add input")
			}
		}

		err = tx.Sign([]bitcoin.Key{rk.Key})
		if err == nil {
			break
		}
		if errors.Cause(err) == txbuilder.ErrInsufficientValue {
			// Get more utxos
			amount = uint64(float32(amount) * 1.25)
			utxos, err = m.UTXOs.Get(amount, rk.Address)
			if err != nil {
				return errors.Wrap(err, "Failed to get UTXOs")
			}

			// Clear inputs
			tx.Inputs = nil
		}
	}

	// Send tx
	return node.Respond(ctx, w, tx.MsgTx)
}

// processSettlementRequest handles an incoming Message SettlementRequest payload.
func (m *Message) processSettlementRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, settlementRequest *messages.SettlementRequest, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Message.processSettlementRequest")
	defer span.End()

	action, err := protocol.Deserialize(settlementRequest.Settlement, w.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to deserialize settlement from settlement request")
	}

	settlement, ok := action.(*actions.Settlement)
	if !ok {
		return errors.New("Settlement Request payload not a settlement")
	}

	// Get transfer tx
	var transferTxId *bitcoin.Hash32
	transferTxId, err = bitcoin.NewHash32(settlementRequest.TransferTxId)
	if err != nil {
		return err
	}

	var transferTx *inspector.Transaction
	transferTx, err = transactions.GetTx(ctx, m.MasterDB, transferTxId, m.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Transfer tx not found")
	}

	// Get transfer from it
	transfer, ok := transferTx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Transfer invalid for transfer tx")
	}

	// Find "first" contract. The "first" contract of a transfer is the one responsible for
	//   creating the initial settlement data and passing it to the next contract if there
	//   are more than one.
	firstContractIndex := uint16(0)
	for _, asset := range transfer.Assets {
		if asset.ContractIndex != uint32(0x0000ffff) {
			break
		}
		// Asset transfer doesn't have a contract (probably BSV transfer).
		firstContractIndex++
	}

	if int(transfer.Assets[firstContractIndex].ContractIndex) >= len(transferTx.Outputs) {
		node.LogWarn(ctx, "Transfer contract index out of range : %x", rk.Address.Bytes())
		return errors.New("Transfer contract index out of range")
	}

	// Bitcoin balance of first contract
	contractBalance := transferTx.Outputs[transfer.Assets[firstContractIndex].ContractIndex].UTXO.Value

	// Build settle tx
	settleTx, err := buildSettlementTx(ctx, m.MasterDB, m.Config, transferTx, transfer,
		settlementRequest, contractBalance, rk)
	if err != nil {
		return errors.Wrap(err, "Failed to build settle tx")
	}

	// Serialize settlement data into OP_RETURN output as a placeholder to be updated by addSettlementData.
	var script []byte
	script, err = protocol.Serialize(settlement, m.Config.IsTest)
	if err != nil {
		node.LogWarn(ctx, "Failed to serialize settlement : %s", err)
		return err
	}
	err = settleTx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	ct, err := contract.Retrieve(ctx, m.MasterDB, rk.Address, m.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transfer, rk,
			actions.RejectionsContractMoved, "Contract address changed")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen")
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transfer, rk,
			actions.RejectionsContractFrozen, "Contract frozen")
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transfer, rk,
			actions.RejectionsContractExpired, "Contract expired")
	}

	// Check Oracle Signature
	if transferTx.RejectCode != 0 {
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transfer, rk,
			transferTx.RejectCode, transferTx.RejectText)
	}

	// Add this contract's data to the settlement op return data
	assetUpdates := make(map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding)
	err = addSettlementData(ctx, m.MasterDB, m.Config, rk, transferTx, transfer, settleTx,
		settlement, m.Headers, &assetUpdates, false)
	if err != nil {
		rejectCode, ok := node.ErrorCode(err)
		if ok {
			node.LogWarn(ctx, "Rejecting Transfer : %s", err)
			return m.respondTransferMessageReject(ctx, w, itx, transferTx, transfer, rk, rejectCode,
				"")
		} else {
			return errors.Wrap(err, "Failed to add settlement data")
		}
	}

	for assetCode, hds := range assetUpdates {
		for _, h := range *hds {
			cacheItem, err := holdings.Save(ctx, m.MasterDB, rk.Address, &assetCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holding")
			}
			m.HoldingsChannel.Add(cacheItem)
		}
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, transfer, settlement) {
		// Sign this contracts input of the settle tx.
		signed := false
		var sigHashCache txbuilder.SigHashCache
		for i, _ := range settleTx.Inputs {
			err = settleTx.SignP2PKHInput(i, rk.Key, &sigHashCache)
			if errors.Cause(err) == txbuilder.ErrWrongPrivateKey {
				continue
			}
			if err != nil {
				return err
			}
			node.LogVerbose(ctx, "Signed settlement input %d", i)
			signed = true
		}

		if !signed {
			return errors.New("Failed to find input to sign")
		}

		// This shouldn't happen because we recieved this from another contract and they couldn't
		//   have signed it yet since it was incomplete.
		if settleTx.AllInputsAreSigned() {
			// Remove tracer for this request.
			boomerangIndex := findBoomerangIndex(transferTx, transfer, rk.Address)
			if boomerangIndex != 0xffffffff {
				outpoint := wire.OutPoint{Hash: *transferTx.Hash, Index: boomerangIndex}
				m.Tracer.Remove(ctx, &outpoint)
			}

			node.Log(ctx, "Broadcasting settlement tx")
			// Send complete settlement tx as response
			return node.Respond(ctx, w, settleTx.MsgTx)
		}

		// Send back to previous contract via a M1 - 1002 Signature Request
		return sendToPreviousSettlementContract(ctx, m.Config, w, rk, itx, settleTx)
	}

	// Save tx
	if err := transactions.AddTx(ctx, m.MasterDB, transferTx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Send to next contract
	return sendToNextSettlementContract(ctx, w, rk, itx, transferTx, transfer, settleTx, settlement, settlementRequest, m.Tracer)
}

// settlementIsComplete returns true if the settlement accounts for all assets in the transfer.
func settlementIsComplete(ctx context.Context, transfer *actions.Transfer, settlement *actions.Settlement) bool {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.settlementIsComplete")
	defer span.End()

	for _, assetTransfer := range transfer.Assets {
		found := false
		for _, assetSettle := range settlement.Assets {
			if assetTransfer.AssetType == assetSettle.AssetType &&
				bytes.Equal(assetTransfer.AssetCode, assetSettle.AssetCode) {
				assetCode := protocol.AssetCodeFromBytes(assetTransfer.AssetCode)
				assetID := protocol.AssetID(assetTransfer.AssetType, *assetCode)
				node.LogVerbose(ctx, "Found settlement data for asset : %s", assetID)
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

// processSigRequest handles an incoming Message SignatureRequest payload.
func (m *Message) processSigRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, sigRequest *messages.SignatureRequest, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.processSigRequest")
	defer span.End()

	var tx wire.MsgTx
	buf := bytes.NewBuffer(sigRequest.Payload)
	err := tx.Deserialize(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to deserialize sig request payload tx")
	}

	// Find OP_RETURN
	for _, output := range tx.TxOut {
		action, err := protocol.Deserialize(output.PkScript, m.Config.IsTest)
		if err == nil {
			switch msg := action.(type) {
			case *actions.Settlement:
				node.LogVerbose(ctx, "Processing Settlement Signature Request")
				return m.processSigRequestSettlement(ctx, w, itx, rk, sigRequest, &tx, msg)
			default:
				return fmt.Errorf("Unsupported signature request tx payload type : %s", action.Code())
			}
		}
	}

	return fmt.Errorf("Tokenized OP_RETURN not found in Sig Request Tx : %s", tx.TxHash())
}

// processSigRequestSettlement handles an incoming Message SignatureRequest payload containing a
//   Settlement tx.
func (m *Message) processSigRequestSettlement(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key, sigRequest *messages.SignatureRequest,
	settleWireTx *wire.MsgTx, settlement *actions.Settlement) error {
	// Get transfer tx
	transferTx, err := transactions.GetTx(ctx, m.MasterDB, &settleWireTx.TxIn[0].PreviousOutPoint.Hash,
		m.Config.IsTest)
	if err != nil {
		return errors.New("Failed to get transfer tx")
	}

	// Get transfer from tx
	transferMsg, ok := transferTx.MsgProto.(*actions.Transfer)
	if !ok {
		return errors.New("Transfer invalid for transfer tx")
	}

	ct, err := contract.Retrieve(ctx, m.MasterDB, rk.Address, m.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transferMsg, rk,
			actions.RejectionsContractMoved, "Contract address changed")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen")
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transferMsg, rk,
			actions.RejectionsContractFrozen, "Contract frozen")
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return m.respondTransferMessageReject(ctx, w, itx, transferTx, transferMsg, rk,
			actions.RejectionsContractExpired, "Contract expired")
	}

	// Verify all the data for this contract is correct.
	err = verifySettlement(ctx, w.Config, m.MasterDB, rk, transferTx, transferMsg, settleWireTx,
		settlement, m.Headers)
	if err != nil {
		rejectCode, ok := node.ErrorCode(err)
		if ok {
			node.LogWarn(ctx, "Rejecting Transfer : %s", err)
			return m.respondTransferMessageReject(ctx, w, itx, transferTx, transferMsg, rk,
				rejectCode, "")
		} else {
			return errors.Wrap(err, "Failed to verify settlement data")
		}
	}

	// Convert settle tx to a txbuilder tx
	var settleTx *txbuilder.TxBuilder
	settleTx, err = txbuilder.NewTxBuilderFromWire(m.Config.FeeRate, m.Config.DustFeeRate,
		settleWireTx, []*wire.MsgTx{transferTx.MsgTx})
	settleTx.SetChangeAddress(rk.Address, "")
	if err != nil {
		return errors.Wrap(err, "Failed to compose settle tx")
	}

	// Sign this contracts input of the settle tx.
	signed := false
	var hashCache txbuilder.SigHashCache
	for i, _ := range settleTx.Inputs {
		err = settleTx.SignP2PKHInput(i, rk.Key, &hashCache)
		if errors.Cause(err) == txbuilder.ErrWrongPrivateKey {
			continue
		}
		if err != nil {
			return err
		}
		node.LogVerbose(ctx, "Signed settlement input %d", i)
		signed = true
	}

	if !signed {
		return errors.New("Failed to find input to sign")
	}

	// This shouldn't happen because we recieved this from another contract and they couldn't
	//   have signed it yet since it was incomplete.
	if settleTx.AllInputsAreSigned() {
		// Remove pending transfer
		if err := transfer.Remove(ctx, m.MasterDB, rk.Address,
			protocol.TxIdFromBytes(transferTx.Hash[:])); err != nil {
			return errors.Wrap(err, "Failed to save pending transfer")
		}

		// Cancel transfer timeout
		err := m.Scheduler.CancelJob(ctx, listeners.NewTransferTimeout(nil, transferTx,
			protocol.NewTimestamp(0)))
		if err != nil {
			if err == scheduler.NotFound {
				node.LogWarn(ctx, "Transfer timeout job not found to cancel")
			} else {
				return errors.Wrap(err, "Failed to cancel transfer timeout")
			}
		}

		node.Log(ctx, "Broadcasting settlement tx")
		// Send complete settlement tx as response
		return node.Respond(ctx, w, settleTx.MsgTx)
	}

	// Send back to previous contract via a M1 - 1002 Signature Request
	return sendToPreviousSettlementContract(ctx, m.Config, w, rk, itx, settleTx)
}

// sendToPreviousSettlementContract sends the completed settlement tx to the previous contract involved so it can sign it.
func sendToPreviousSettlementContract(ctx context.Context, config *node.Config, w *node.ResponseWriter,
	rk *wallet.Key, itx *inspector.Transaction, settleTx *txbuilder.TxBuilder) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.sendToPreviousSettlementContract")
	defer span.End()

	// Find previous input that still needs a signature
	inputIndex := 0xffffffff
	for i, _ := range settleTx.MsgTx.TxIn {
		if !settleTx.InputIsSigned(i) {
			inputIndex = i
		}
	}

	// This only happens if this function was called in error with a completed tx.
	if inputIndex == 0xffffffff {
		return errors.New("Could not find input that needs signature")
	}

	address, err := settleTx.InputAddress(inputIndex)
	if err != nil {
		return err
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	node.Log(ctx, "Sending settlement SignatureRequest to %x", address.Bytes())

	// Add output to previous contract.
	// Mark as change so it gets everything except the tx fee.
	err = w.AddChangeOutput(ctx, address)
	if err != nil {
		return err
	}

	// Serialize settlement data for Message OP_RETURN.
	var txBuf bytes.Buffer
	err = settleTx.MsgTx.Serialize(&txBuf)
	if err != nil {
		return err
	}

	messagePayload := messages.SignatureRequest{
		Timestamp: v.Now.Nano(),
		Payload:   txBuf.Bytes(),
	}

	// Setup Message
	var payBuf bytes.Buffer
	err = messagePayload.Serialize(&payBuf)
	if err != nil {
		return err
	}
	message := actions.Message{
		ReceiverIndexes: []uint32{0}, // First output is receiver of message
		MessageCode:     messagePayload.Code(),
		MessagePayload:  payBuf.Bytes(),
	}

	return node.RespondSuccess(ctx, w, itx, rk, &message)
}

// verifySettlement verifies that all settlement data related to this contract and bitcoin transfers are correct.
func verifySettlement(ctx context.Context, config *node.Config, masterDB *db.DB, rk *wallet.Key,
	transferTx *inspector.Transaction, transfer *actions.Transfer, settleTx *wire.MsgTx,
	settlement *actions.Settlement, headers node.BitcoinHeaders) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.verifySettlement")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]bitcoin.RawAddress, 0, len(settleTx.TxOut))
	settleOpReturnFound := false
	for i, output := range settleTx.TxOut {
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err == nil {
			settleOutputAddresses = append(settleOutputAddresses, address)
		} else {
			settleOutputAddresses = append(settleOutputAddresses, bitcoin.RawAddress{})
		}

		action, err := protocol.Deserialize(output.PkScript, config.IsTest)
		if err != nil {
			continue
		}
		if action.Code() == actions.CodeSettlement && !settleOpReturnFound {
			settleOpReturnFound = true
			continue
		}
		if action.Code() == actions.CodeMessage {
			message, ok := action.(*actions.Message)
			if !ok {
				return fmt.Errorf("Invalid Tokenized OP_RETURN message script : output %d", i)
			}
			if message.MessageCode != messages.CodeOutputMetadata {
				return fmt.Errorf("Invalid Tokenized OP_RETURN non-metadata message script : output %d", i)
			}
			continue
		}
		return fmt.Errorf("Unexpected Tokenized OP_RETURN script : output %d", i)
	}

	// Generate public key hashes for all the inputs
	settleInputAddresses := make([]bitcoin.RawAddress, 0, len(settleTx.TxIn))
	for _, input := range settleTx.TxIn {
		address, err := bitcoin.RawAddressFromUnlockingScript(input.SignatureScript)
		if err != nil {
			settleInputAddresses = append(settleInputAddresses, bitcoin.RawAddress{})
		} else {
			settleInputAddresses = append(settleInputAddresses, address)
		}
	}

	txid := protocol.TxIdFromBytes(transferTx.Hash[:])
	ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	for assetOffset, assetTransfer := range transfer.Assets {
		assetCode := protocol.AssetCodeFromBytes(assetTransfer.AssetCode)
		assetID := protocol.AssetID(assetTransfer.AssetType, *assetCode)
		assetIsBitcoin := assetTransfer.AssetType == "BSV" && assetCode.IsZero()

		var as *state.Asset
		if !assetIsBitcoin {
			if len(settleTx.TxOut) <= int(assetTransfer.ContractIndex) {
				return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
			}

			contractOutputAddress := settleOutputAddresses[assetTransfer.ContractIndex]
			if !contractOutputAddress.IsEmpty() && !contractOutputAddress.Equal(rk.Address) {
				continue // This asset is not for this contract.
			}
			if ct.FreezePeriod.Nano() > v.Now.Nano() {
				return node.NewError(actions.RejectionsContractFrozen, "")
			}

			// Locate Asset
			as, err = asset.Retrieve(ctx, masterDB, rk.Address, assetCode)
			if err != nil {
				return fmt.Errorf("Asset code not found : %s : %s", assetID, err)
			}
			if as.FreezePeriod.Nano() > v.Now.Nano() {
				return node.NewError(actions.RejectionsAssetFrozen, "")
			}
			if !as.TransfersPermitted {
				return node.NewError(actions.RejectionsAssetNotPermitted, "")
			}
		}

		// Find settlement for asset.
		var assetSettlement *actions.AssetSettlementField
		for i, asset := range settlement.Assets {
			if asset.AssetType == assetTransfer.AssetType &&
				bytes.Equal(asset.AssetCode, assetTransfer.AssetCode) {
				assetSettlement = settlement.Assets[i]
				break
			}
		}

		if assetSettlement == nil {
			return fmt.Errorf("Asset settlement not found during verify")
		}

		sendBalance := uint64(0)
		settlementQuantities := make([]*uint64, len(settleTx.TxOut))

		// Process senders
		// assetTransfer.AssetSenders []QuantityIndex {Index uint16, Quantity uint64}
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return fmt.Errorf("Sender input index out of range for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
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

			// Check send
			var settlementQuantity uint64
			if !assetIsBitcoin {
				h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode,
					transferTx.Inputs[sender.Index].Address, v.Now)
				if err != nil {
					return errors.Wrap(err, "Failed to get sender holding")
				}

				settlementQuantity, err = holdings.CheckDebit(h, txid, sender.Quantity)
				if err != nil {
					address := bitcoin.NewAddressFromRawAddress(transferTx.Inputs[sender.Index].Address,
						config.Net)
					node.LogWarn(ctx, "Send invalid : %s %s : %s", assetID, address, err)
					return node.NewError(actions.RejectionsMsgMalformed, "")
				}
			}

			settlementQuantities[settleOutputIndex] = &settlementQuantity

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
			if err != nil {
				return fmt.Errorf("Receiver address invalid for asset %d receiver %d : %s",
					assetOffset, receiverOffset, err)
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
					assetOffset, receiverOffset, address.String())
			}

			// Check receive
			var settlementQuantity uint64
			if !assetIsBitcoin {
				h, err := holdings.GetHolding(ctx, masterDB, rk.Address, assetCode, receiverAddress,
					v.Now)
				if err != nil {
					return errors.Wrap(err, "Failed to get reciever holding")
				}

				settlementQuantity, err = holdings.CheckDeposit(h, txid, receiver.Quantity)
				if err != nil {
					address := bitcoin.NewAddressFromRawAddress(receiverAddress,
						config.Net)
					node.LogWarn(ctx, "Receive invalid : %x %s : %s",
						assetTransfer.AssetCode, address.String(), err)
					return node.NewError(actions.RejectionsMsgMalformed, "")
				}
			}

			settlementQuantities[settleOutputIndex] = &settlementQuantity

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

		// Check ending balances
		for index, quantity := range settlementQuantities {
			if quantity != nil {
				found := false
				for _, settlementQuantity := range assetSettlement.Settlements {
					if index == int(settlementQuantity.Index) {
						if *quantity != settlementQuantity.Quantity {
							node.LogWarn(ctx, "Incorrect settlment quantity for output %d : %d != %d : %s",
								index, *quantity, settlementQuantity.Quantity, assetID)
							return fmt.Errorf("Asset settlement quantity wrong")
						}
						found = true
						break
					}
				}

				if !found {
					node.LogWarn(ctx, "Missing settlment for output %d : %s", index, assetID)
					return fmt.Errorf("Asset settlement missing")
				}
			}
		}
	}

	// Verify contract fee
	if ct.ContractFee > 0 {
		found := false
		for i, outputAddress := range settleOutputAddresses {
			if !outputAddress.IsEmpty() && outputAddress.Equal(config.FeeAddress) {
				if uint64(settleTx.TxOut[i].Value) < ct.ContractFee {
					return fmt.Errorf("Contract fee too low")
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Contract fee missing")
		}
	}

	return nil
}

// respondTransferMessageReject responds to an M1 SettlementRequest or SignatureRequest with a
//   rejection message.
// If this is the first contract, it will send a full refund/reject to all parties involved.
// If this is not the first contract, it will send a reject message to the first contract so that
//   it can send the refund/reject to everyone.
func (m *Message) respondTransferMessageReject(ctx context.Context, w *node.ResponseWriter,
	messageTx *inspector.Transaction, transferTx *inspector.Transaction, transfer *actions.Transfer,
	rk *wallet.Key, code uint32, text string) error {

	// Determine if first contract
	first := firstContractOutputIndex(transfer.Assets, transferTx)
	if first == 0xffff {
		return errors.New("First contract output index not found")
	}

	if !transferTx.Outputs[first].Address.Equal(rk.Address) {
		// This is not the first contract. Send reject to only the first contract.
		w.AddRejectValue(ctx, transferTx.Outputs[first].Address, 0)
		return node.RespondRejectText(ctx, w, messageTx, rk, code, text)
	}

	// Determine UTXOs from transfer tx to fund the reject response.
	utxos, err := transferTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Transfer UTXOs not found")
	}

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

	// Add utxo from message tx
	messageUTXOs, err := messageTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Message UTXOs not found")
	}

	utxos = append(utxos, messageUTXOs...)

	balance := uint64(0)
	for _, utxo := range utxos {
		balance += uint64(utxo.Value)
	}

	w.SetRejectUTXOs(ctx, utxos)

	// Add refund amounts for all bitcoin senders
	refundBalance := uint64(0)
	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			// Process bitcoin senders refunds
			for _, sender := range assetTransfer.AssetSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

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
		ct, err := contract.Retrieve(ctx, m.MasterDB, rk.Address, m.Config.IsTest)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}

		// Funding not enough to refund everyone, so don't refund to anyone. Send it to the administration to hold.
		w.ClearRejectOutputValues(ct.AdminAddress)
	}

	return node.RespondRejectText(ctx, w, transferTx, rk, code, text)
}

// refundTransferFromReject responds to an M2 Reject, from another contract involved in a multi-contract
//   transfer with a tx refunding any bitcoin sent to the contract that was requested to be
//   transferred.
func refundTransferFromReject(ctx context.Context, config *node.Config, masterDB *db.DB,
	holdingsChannel *holdings.CacheChannel, sch *scheduler.Scheduler, w *node.ResponseWriter,
	rejectionTx *inspector.Transaction, rejection *actions.Rejection,
	transferTx *inspector.Transaction, transferMsg *actions.Transfer, rk *wallet.Key) error {

	v := ctx.Value(node.KeyValues).(*node.Values)
	transferTxId := protocol.TxIdFromBytes(transferTx.Hash[:])

	// Remove pending transfer
	if err := transfer.Remove(ctx, masterDB, rk.Address, transferTxId); err != nil {
		return errors.Wrap(err, "Failed to save pending transfer")
	}

	// Cancel transfer timeout
	err := sch.CancelJob(ctx, listeners.NewTransferTimeout(nil, transferTx, protocol.NewTimestamp(0)))
	if err != nil {
		if err == scheduler.NotFound {
			node.LogWarn(ctx, "Transfer timeout job not found to cancel")
		} else {
			return errors.Wrap(err, "Failed to cancel transfer timeout")
		}
	}

	// Find first contract index.
	first := firstContractOutputIndex(transferMsg.Assets, transferTx)
	if first == 0x0000ffff {
		return errors.New("First contract output index not found")
	}

	// Determine UTXOs from transfer tx to fund the reject response.
	utxos, err := transferTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Transfer UTXOs not found")
	}

	// Determine if this contract is the first contract and needs to send a refund.
	refund := false
	if transferTx.Outputs[first].Address.Equal(rk.Address) {
		refund = true

		// Remove utxo spent by boomerang
		boomerangIndex := findBoomerangIndex(transferTx, transferMsg, rk.Address)
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

	// Add utxo from message tx
	messageUTXOs, err := rejectionTx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return errors.Wrap(err, "Message UTXOs not found")
	}

	utxos = append(utxos, messageUTXOs...)

	balance := uint64(0)
	for _, utxo := range utxos {
		balance += uint64(utxo.Value)
	}

	updates := make(map[protocol.AssetCode]*map[bitcoin.Hash20]*state.Holding)

	w.SetRejectUTXOs(ctx, utxos)

	// Add refund amounts for all bitcoin senders
	refundBalance := uint64(0)
	for assetOffset, assetTransfer := range transferMsg.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			if refund {
				// Process bitcoin senders refunds
				for _, sender := range assetTransfer.AssetSenders {
					if int(sender.Index) >= len(transferTx.Inputs) {
						continue
					}

					w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, sender.Quantity)
					refundBalance += sender.Quantity
				}
			}
		} else {
			if len(transferTx.Outputs) <= int(assetTransfer.ContractIndex) {
				return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
			}

			if !transferTx.Outputs[assetTransfer.ContractIndex].Address.Equal(rk.Address) {
				continue // This asset is not ours. Skip it.
			}

			assetCode := protocol.AssetCodeFromBytes(assetTransfer.AssetCode)
			updatedHoldings := make(map[bitcoin.Hash20]*state.Holding)
			updates[*assetCode] = &updatedHoldings

			// Add all other senders to be notified
			// Revert sender pending statuses
			for _, sender := range assetTransfer.AssetSenders {
				if int(sender.Index) >= len(transferTx.Inputs) {
					continue
				}

				w.AddRejectValue(ctx, transferTx.Inputs[sender.Index].Address, 0)

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

	err = saveHoldings(ctx, masterDB, holdingsChannel, updates, rk.Address)
	if err != nil {
		return errors.Wrap(err, "save holdings")
	}

	if refund && refundBalance > balance {
		ct, err := contract.Retrieve(ctx, masterDB, rk.Address, config.IsTest)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}

		// Funding not enough to refund everyone, so don't refund to anyone.
		w.ClearRejectOutputValues(ct.AdminAddress)
	}

	// Set rejection address from previous rejection
	if int(rejection.RejectAddressIndex) < len(rejectionTx.Outputs) {
		w.RejectAddress = rejectionTx.Outputs[rejection.RejectAddressIndex].Address
	}

	return node.RespondReject(ctx, w, transferTx, rk, rejection.RejectionCode)
}
