package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

type Message struct {
	MasterDB *db.DB
	Config   *node.Config
	TxCache  InspectorTxCache
}

// ProcessMessage handles an incoming Message OP_RETURN.
func (m *Message) ProcessMessage(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.ProcessMessage")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Message)
	if !ok {
		return errors.New("Could not assert as *protocol.Message")
	}

	messagePayload, err := msg.PayloadMessage()
	if err != nil {
		return err
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	switch payload := messagePayload.(type) {
	case *protocol.PublicMessage:
		return errors.New("PublicMessage not implemented")
	case *protocol.PrivateMessage:
		return errors.New("PrivateMessage not implemented")
	case *protocol.Offer:
		logger.Verbose(ctx, "%s : Processing Offer", v.TraceID)
		return m.processOffer(ctx, w, itx, rk, payload)
	case *protocol.SignatureRequest:
		logger.Verbose(ctx, "%s : Processing SignatureRequest", v.TraceID)
		return m.processSigRequest(ctx, w, itx, rk, payload)
	default:
		return fmt.Errorf("Unknown message payload type : %s", messagePayload.Type())
	}
}

// processOffer handles an incoming Message Offer payload.
func (m *Message) processOffer(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.RootKey, offer *protocol.Offer) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.processOffer")
	defer span.End()

	offerPayload, err := protocol.Deserialize(offer.Payload)
	if err != nil {
		return err
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	switch payload := offerPayload.(type) {
	case *protocol.Settlement:
		logger.Verbose(ctx, "%s : Processing Settlement Offer", v.TraceID)
		return m.processSettlementOffer(ctx, w, itx, rk, offer, payload)
	default:
		return fmt.Errorf("Unsupported offer payload type : %s", offerPayload.Type())
	}

	return nil
}

// processSettlementOffer processes an Offer message containing a Settlement.
// It is a request to add settlement data relating to this contract.
func (m *Message) processSettlementOffer(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.RootKey, offer *protocol.Offer,
	settlement *protocol.Settlement) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.processSettlementOffer")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Get transfer tx
	var err error
	var refTxId *chainhash.Hash
	refTxId, err = chainhash.NewHash(offer.RefTxId.Bytes())
	if err != nil {
		return err
	}

	var transferTx *inspector.Transaction
	transferTx = m.TxCache.GetTx(ctx, refTxId)
	if transferTx == nil {
		return errors.New("Transfer tx not found")
	}

	// Get transfer from it
	var transfer *protocol.Transfer
	for _, output := range transferTx.MsgTx.TxOut {
		opReturn, err := protocol.Deserialize(output.PkScript)
		if err != nil {
			continue
		}

		var ok bool
		transfer, ok = opReturn.(*protocol.Transfer)
		if ok {
			break
		}
	}

	if transfer == nil {
		return errors.New("Could not find Transfer in transfer tx")
	}

	// Find "first" contract. The "first" contract of a transfer is the one responsible for
	//   creating the initial settlement data and passing it to the next contract if there
	//   are more than one.
	firstContractIndex := uint16(0)
	for _, asset := range transfer.Assets {
		if asset.ContractIndex != uint16(0xffff) {
			break
		}
		// Asset transfer doesn't have a contract (probably BSV transfer).
		firstContractIndex++
	}

	if int(transfer.Assets[firstContractIndex].ContractIndex) >= len(transferTx.Outputs) {
		logger.Warn(ctx, "%s : Transfer contract index out of range : %s", v.TraceID, rk.Address.String())
		return errors.New("Transfer contract index out of range")
	}

	// Bitcoin balance of first contract. Funding for bitcoin transfers.
	contractBalance := uint64(transferTx.Outputs[transfer.Assets[firstContractIndex].ContractIndex].Value)

	// Build settlement tx
	var settleTx *txbuilder.Tx
	settleTx, err = buildSettlementTx(ctx, w.Config, transferTx, transfer, rk, contractBalance)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to build settlement tx : %s", v.TraceID, err)
		// TODO How exactly do these rejects work. Do you send a reject based on the Transfer Tx?
		return node.RespondReject(ctx, w, transferTx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Update outputs to pay bitcoin receivers.
	err = addBitcoinSettlements(ctx, transferTx, transfer, settleTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to add bitcoin settlements : %s", v.TraceID, err)
		// TODO How exactly do these rejects work. Do you send a reject based on the Transfer Tx?
		return node.RespondReject(ctx, w, transferTx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Serialize received settlement data into OP_RETURN output.
	var script []byte
	script, err = protocol.Serialize(settlement)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to serialize empty settlement : %s", v.TraceID, err)
		return err
	}
	err = settleTx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	// Add this contract's data to the settlement op return data
	err = addSettlementData(ctx, m.MasterDB, rk, transferTx, transfer, settleTx.MsgTx, settlement)
	if err == frozenError {
		logger.Warn(ctx, "%s : Assets Frozen. Rejecting Transfer", v.TraceID)
		return node.RespondReject(ctx, w, transferTx, rk, protocol.RejectionCodeFrozen)
	}
	if err != nil {
		return err
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, transfer, settlement) {
		// Sign this contracts input of the settle tx.
		signed := false
		for i, _ := range settleTx.Inputs {
			err = settleTx.SignInput(i, rk.PrivateKey)
			if txbuilder.IsErrorCode(err, txbuilder.ErrorCodeWrongPrivateKey) {
				continue
			}
			if err != nil {
				return err
			}
			logger.Verbose(ctx, "%s : Signed settlement input %d", v.TraceID, i)
			signed = true
		}

		if !signed {
			return errors.New("Failed to find input to sign")
		}

		// This shouldn't happen because we recieved this from another contract and they couldn't
		//   have signed it yet since it was incomplete.
		if settleTx.AllInputsAreSigned() {
			logger.Info(ctx, "%s : Broadcasting settlement tx", v.TraceID)
			// Send complete settlement tx as response
			return node.Respond(ctx, w, settleTx.MsgTx)
		}

		// Send back to previous contract via a M1 - 1002 Signature Request
		return sendToPreviousSettlementContract(ctx, m.Config, w, rk, itx, settleTx)
	}

	// Save tx
	err = m.TxCache.SaveTx(ctx, transferTx)
	if err != nil {
		return err
	}

	// Send to next contract
	return sendToNextSettlementContract(ctx, w, rk, itx, transferTx, transfer, settleTx, settlement)
}

// processSigRequest handles an incoming Message SignatureRequest payload.
func (m *Message) processSigRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.RootKey, sigRequest *protocol.SignatureRequest) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Message.processSigRequest")
	defer span.End()

	var tx wire.MsgTx
	buf := bytes.NewBuffer(sigRequest.Payload)
	err := tx.Deserialize(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to deserialize sig request payload tx")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Find OP_RETURN
	for _, output := range tx.TxOut {
		opReturn, err := protocol.Deserialize(output.PkScript)
		if err == nil {
			switch msg := opReturn.(type) {
			case *protocol.Settlement:
				logger.Verbose(ctx, "%s : Processing Settlement Signature Request", v.TraceID)
				return m.processSigRequestSettlement(ctx, w, itx, rk, sigRequest, &tx, msg)
			default:
				return fmt.Errorf("Unsupported signature request tx payload type : %s", opReturn.Type())
			}
		}
	}

	return fmt.Errorf("Tokenized OP_RETURN not found in Sig Request Tx : %s", tx.TxHash())
}

// processSigRequestSettlement handles an incoming Message SignatureRequest payload containing a
//   Settlement tx.
func (m *Message) processSigRequestSettlement(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.RootKey, sigRequest *protocol.SignatureRequest,
	settleWireTx *wire.MsgTx, settlement *protocol.Settlement) error {
	// Get transfer tx
	transferTx := m.TxCache.GetTx(ctx, &settleWireTx.TxIn[0].PreviousOutPoint.Hash)
	if transferTx == nil {
		return errors.New("Failed to get transfer tx")
	}

	// Find transfer OP_RETURN
	var transfer *protocol.Transfer
	found := false
	for _, output := range transferTx.Outputs {
		opReturn, err := protocol.Deserialize(output.UTXO.PkScript)
		if err == nil {
			transfer, found = opReturn.(*protocol.Transfer)
			if found {
				break
			}
		}
	}

	if !found {
		return errors.New("Failed fo find transfer OP_RETURN in transfer tx")
	}

	// Verify all the data for this contract is correct.
	err := verifySettlement(ctx, m.MasterDB, rk, transferTx, transfer, settleWireTx, settlement)
	if err != nil {
		return errors.Wrap(err, "Failed to validate settlement data")
	}

	// Convert settle tx to a txbuilder tx
	var settleTx *txbuilder.Tx
	settleTx, err = txbuilder.NewTxFromWire(rk.Address.ScriptAddress(), m.Config.DustLimit, m.Config.FeeRate,
		settleWireTx, []*wire.MsgTx{transferTx.MsgTx})
	if err != nil {
		return errors.Wrap(err, "Failed to compose settle tx")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Sign this contracts input of the settle tx.
	signed := false
	for i, _ := range settleTx.Inputs {
		err = settleTx.SignInput(i, rk.PrivateKey)
		if txbuilder.IsErrorCode(err, txbuilder.ErrorCodeWrongPrivateKey) {
			continue
		}
		if err != nil {
			return err
		}
		logger.Verbose(ctx, "%s : Signed settlement input %d", v.TraceID, i)
		signed = true
	}

	if !signed {
		return errors.New("Failed to find input to sign")
	}

	// This shouldn't happen because we recieved this from another contract and they couldn't
	//   have signed it yet since it was incomplete.
	if settleTx.AllInputsAreSigned() {
		logger.Info(ctx, "%s : Broadcasting settlement tx", v.TraceID)
		// Send complete settlement tx as response
		return node.Respond(ctx, w, settleTx.MsgTx)
	}

	// Send back to previous contract via a M1 - 1002 Signature Request
	return sendToPreviousSettlementContract(ctx, m.Config, w, rk, itx, settleTx)
}

// sendToPreviousSettlementContract sends the completed settlement tx to the previous contract involved so it can sign it.
func sendToPreviousSettlementContract(ctx context.Context, config *node.Config, w *node.ResponseWriter,
	rk *wallet.RootKey, itx *inspector.Transaction, settleTx *txbuilder.Tx) error {
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

	pkh, err := settleTx.InputPKH(inputIndex)
	if err != nil {
		return err
	}

	address, err := btcutil.NewAddressPubKeyHash(pkh, &config.ChainParams)
	if err != nil {
		return err
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	logger.Info(ctx, "%s : Sending settlement SignatureRequest to %s", v.TraceID, address.String())

	// Add output to previous contract.
	// Mark as change so it gets everything except the tx fee.
	err = w.AddChangeOutput(ctx, address)
	if err != nil {
		return err
	}

	// Serialize settlement data for Message OP_RETURN.
	var buf bytes.Buffer
	err = settleTx.MsgTx.Serialize(&buf)
	if err != nil {
		return err
	}

	messagePayload := protocol.SignatureRequest{
		Version:   0,
		Timestamp: v.Now,
		Payload:   buf.Bytes(),
	}

	// Setup Message
	var data []byte
	data, err = messagePayload.Serialize()
	if err != nil {
		return err
	}
	message := protocol.Message{
		AddressIndexes: []uint16{0}, // First output is receiver of message
		MessageType:    messagePayload.Type(),
		MessagePayload: data,
	}

	return node.RespondSuccess(ctx, w, itx, rk, &message)
}

// verifySettlement verifies that all settlement data related to this contract and bitcoin transfers are correct.
func verifySettlement(ctx context.Context, masterDB *db.DB, rk *wallet.RootKey,
	transferTx *inspector.Transaction, transfer *protocol.Transfer, settleTx *wire.MsgTx,
	settlement *protocol.Settlement) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.verifySettlement")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]*protocol.PublicKeyHash, 0, len(settleTx.TxOut))
	for _, output := range settleTx.TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err == nil {
			settleOutputAddresses = append(settleOutputAddresses, protocol.PublicKeyHashFromBytes(pkh))
		} else {
			settleOutputAddresses = append(settleOutputAddresses, nil)
		}
	}

	// Generate public key hashes for all the inputs
	hash256 := sha256.New()
	hash160 := ripemd160.New()
	settleInputAddresses := make([]protocol.PublicKeyHash, 0, len(settleTx.TxIn))
	for _, input := range settleTx.TxIn {
		pushes, err := txscript.PushedData(input.SignatureScript)
		if err != nil {
			settleInputAddresses = append(settleInputAddresses, protocol.PublicKeyHash{})
			continue
		}
		if len(pushes) != 2 {
			settleInputAddresses = append(settleInputAddresses, protocol.PublicKeyHash{})
			continue
		}

		// Calculate RIPEMD160(SHA256(PublicKey))
		hash256.Reset()
		hash256.Write(pushes[1])
		hash160.Reset()
		hash160.Write(hash256.Sum(nil))
		settleInputAddresses = append(settleInputAddresses, *protocol.PublicKeyHashFromBytes(hash160.Sum(nil)))
	}

	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	for assetOffset, assetTransfer := range transfer.Assets {
		assetIsBitcoin := assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()

		if len(settleTx.TxOut) <= int(assetTransfer.ContractIndex) {
			return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
		}

		contractOutputAddress := settleOutputAddresses[assetTransfer.ContractIndex]
		if contractOutputAddress != nil && !bytes.Equal(contractOutputAddress.Bytes(), contractAddr.Bytes()) {
			continue // This asset is not for this contract.
		}

		// Locate Asset
		var as *state.Asset
		var err error
		if !assetIsBitcoin {
			as, err = asset.Retrieve(ctx, masterDB, contractAddr, &assetTransfer.AssetCode)
			if err != nil || as == nil {
				return fmt.Errorf("Asset ID not found : %s %s : %s", contractAddr, assetTransfer.AssetCode, err)
			}
		}

		// Find settlement for asset.
		var assetSettlement *protocol.AssetSettlement
		for i, asset := range settlement.Assets {
			if asset.AssetType == assetTransfer.AssetType &&
				bytes.Equal(asset.AssetCode.Bytes(), assetTransfer.AssetCode.Bytes()) {
				assetSettlement = &settlement.Assets[i]
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

			inputPKH := transferTx.Inputs[sender.Index].Address.ScriptAddress()

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if outputAddress != nil && bytes.Equal(outputAddress.Bytes(), inputPKH) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Sender output not found in settle tx for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Outputs))
			}

			// Check sender's available unfrozen balance
			inputAddress := protocol.PublicKeyHashFromBytes(inputPKH)
			if !assetIsBitcoin && !asset.CheckBalanceFrozen(ctx, as, inputAddress, sender.Quantity, v.Now) {
				logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr, assetTransfer.AssetCode, inputAddress)
				return frozenError
			}

			if settlementQuantities[settleOutputIndex] == nil {
				// Get sender's balance
				if assetIsBitcoin {
					senderBalance := uint64(0)
					settlementQuantities[settleOutputIndex] = &senderBalance
				} else {
					senderBalance := asset.GetBalance(ctx, as, inputAddress)
					settlementQuantities[settleOutputIndex] = &senderBalance
				}
			}

			if *settlementQuantities[settleOutputIndex] < sender.Quantity {
				logger.Warn(ctx, "%s : Insufficient funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr, assetTransfer.AssetCode, inputAddress)
				return insufficientError
			}

			*settlementQuantities[settleOutputIndex] -= sender.Quantity

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		// assetTransfer.AssetReceivers []TokenReceiver {Index uint16, Quantity uint64,
		//   RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string}
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Get receiver address from outputs[receiver.Index]
			if int(receiver.Index) >= len(transferTx.Outputs) {
				return fmt.Errorf("Receiver output index out of range for asset %d sender %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			transferOutputAddress := protocol.PublicKeyHashFromBytes(transferTx.Outputs[receiver.Index].Address.ScriptAddress())

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if outputAddress != nil && bytes.Equal(outputAddress.Bytes(), transferOutputAddress.Bytes()) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Receiver output not found in settle tx for asset %d receiver %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			// TODO Process RegistrySignatures

			if settlementQuantities[settleOutputIndex] == nil {
				// Get receiver's balance
				if assetIsBitcoin {
					receiverBalance := uint64(0)
					settlementQuantities[settleOutputIndex] = &receiverBalance
				} else {
					receiverBalance := asset.GetBalance(ctx, as, transferOutputAddress)
					settlementQuantities[settleOutputIndex] = &receiverBalance
				}
			}

			*settlementQuantities[settleOutputIndex] += receiver.Quantity

			// Update asset balance
			if receiver.Quantity > sendBalance {
				return fmt.Errorf("Receiving more tokens than sending for asset %d", assetOffset)
			}
			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return fmt.Errorf("Not sending all input tokens for asset %d : %d remaining", assetOffset, sendBalance)
		}

		// Check ending balances
		for index, quantity := range settlementQuantities {
			if quantity != nil {
				found := false
				for _, settlementQuantity := range assetSettlement.Settlements {
					if index == int(settlementQuantity.Index) {
						if *quantity != settlementQuantity.Quantity {
							logger.Warn(ctx, "%s : Incorrect settlment quantity for output %d : %d != %d : %x",
								v.TraceID, index, *quantity, settlementQuantity.Quantity, assetTransfer.AssetCode)
							return fmt.Errorf("Asset settlement quantity wrong")
						}
						found = true
						break
					}
				}

				if !found {
					logger.Warn(ctx, "%s : missing settlment for output %d : %x",
						v.TraceID, index, assetTransfer.AssetCode)
					return fmt.Errorf("Asset settlement missing")
				}
			}
		}
	}

	return nil
}
