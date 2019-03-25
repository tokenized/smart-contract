package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

type Transfer struct {
	MasterDB *db.DB
	Config   *node.Config
	TxCache  InspectorTxCache
}

var (
	frozenError       = errors.New("Holdings Frozen")
	insufficientError = errors.New("Holdings Insufficient")
)

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferRequest")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	msg, ok := itx.MsgProto.(*protocol.Transfer)
	if !ok {
		return errors.New("Could not assert as *protocol.Transfer")
	}

	if len(msg.Assets) == 0 {
		return errors.New("protocol.Transfer has no asset transfers")
	}

	// Find "first" contract. The "first" contract of a transfer is the one responsible for
	//   creating the initial settlement data and passing it to the next contract if there
	//   are more than one.
	firstContractIndex := uint16(0)
	for _, asset := range msg.Assets {
		if asset.ContractIndex != uint16(0xffff) {
			break
		}
		// Asset transfer doesn't have a contract (probably BSV transfer).
		firstContractIndex++
	}

	if int(msg.Assets[firstContractIndex].ContractIndex) >= len(itx.Outputs) {
		logger.Warn(ctx, "%s : Transfer contract index out of range : %s", v.TraceID, rk.Address.String())
		return errors.New("Transfer contract index out of range")
	}

	if itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Address.String() != rk.Address.String() {
		logger.Verbose(ctx, "%s : Not contract for first transfer. Waiting for Message Offer : %s", v.TraceID,
			itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Address.String())
		t.TxCache.SaveTx(ctx, itx)
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	// Bitcoin balance of first (this) contract. Funding for bitcoin transfers.
	contractBalance := uint64(itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Value)

	// TODO Verify input amounts for fees.

	// Transfer Outputs
	//   Contract 1 : amount = calculated fee for settlement tx + any bitcoins being transfered
	//   Contract 2 : dust
	//   Boomerang to Contract 1 : amount = ((n-1) * 2) * (calculated fee for data passing tx)
	//     where n is number of contracts involved
	// Boomerang is only required when more than one contract is involved.
	//
	// Transfer Inputs
	//   Any addresses sending tokens or bitcoin.
	//
	// Each contract can be involved in more than one asset in the transfer, but only needs to have
	//   one output since each asset transfer references the output of it's contract
	var err error
	var settleTx *txbuilder.Tx
	settleTx, err = buildSettlementTx(ctx, t.Config, itx, msg, rk, contractBalance)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to build settlement tx : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Update outputs to pay bitcoin receivers.
	err = addBitcoinSettlements(ctx, itx, msg, settleTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to add bitcoin settlements : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Create initial settlement data
	settlement := protocol.Settlement{Timestamp: v.Now}

	// Serialize empty settlement data into OP_RETURN output.
	var script []byte
	script, err = protocol.Serialize(&settlement)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to serialize empty settlement : %s", v.TraceID, err)
		return err
	}
	err = settleTx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	// Add this contract's data to the settlement op return data
	err = addSettlementData(ctx, t.MasterDB, rk, itx, msg, settleTx.MsgTx, &settlement)
	if err == frozenError {
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeFrozen)
	}
	if err != nil {
		return err
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, msg, &settlement) {
		logger.Info(ctx, "%s : Single contract settlement complete", v.TraceID)
		if err := settleTx.Sign([]*btcec.PrivateKey{rk.PrivateKey}); err != nil {
			return err
		}
		return node.Respond(ctx, w, settleTx.MsgTx)
	}

	// Save tx
	err = t.TxCache.SaveTx(ctx, itx)
	if err != nil {
		return err
	}

	// Send to next contract
	return sendToNextSettlementContract(ctx, w, rk, itx, itx, msg, settleTx, &settlement)
}

// buildSettlementTx builds the tx for a settlement action.
func buildSettlementTx(ctx context.Context, config *node.Config, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, rk *wallet.RootKey, contractBalance uint64) (*txbuilder.Tx, error) {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.buildSettlementTx")
	defer span.End()

	// Settle Outputs
	//   Any addresses sending or receiving tokens or bitcoin.
	//   Referenced from indices from within settlement data.
	//
	// Settle Inputs
	//   Any contracts involved.
	settleTx := txbuilder.NewTx(rk.Address.ScriptAddress(), config.DustLimit, config.FeeRate)

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
			transferTx.Outputs[assetTransfer.ContractIndex].UTXO.PkScript, uint64(transferTx.Outputs[assetTransfer.ContractIndex].Value))
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

			address := protocol.PublicKeyHashFromBytes(transferTx.Inputs[quantityIndex.Index].Address.ScriptAddress())
			_, exists := addresses[*address]
			if !exists {
				// Add output to sender
				addresses[*address] = uint32(len(settleTx.MsgTx.TxOut))

				err = settleTx.AddP2PKHDustOutput(transferTx.Inputs[quantityIndex.Index].Address.ScriptAddress(), false)
				if err != nil {
					return nil, err
				}
			}
		}

		for _, tokenReceiver := range assetTransfer.AssetReceivers {
			assetBalance -= tokenReceiver.Quantity
			// TODO Handle Registry : RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string

			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if tokenReceiver.Quantity > contractBalance {
					return nil, fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= tokenReceiver.Quantity
			}

			address := protocol.PublicKeyHashFromBytes(transferTx.Inputs[tokenReceiver.Index].Address.ScriptAddress())
			outputIndex, exists := addresses[*address]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, tokenReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver
				addresses[*address] = uint32(len(settleTx.MsgTx.TxOut))

				if assetIsBitcoin {
					err = settleTx.AddP2PKHOutput(transferTx.Outputs[tokenReceiver.Index].Address.ScriptAddress(), tokenReceiver.Quantity, false)
				} else {
					err = settleTx.AddP2PKHDustOutput(transferTx.Outputs[tokenReceiver.Index].Address.ScriptAddress(), false)
				}
				if err != nil {
					return nil, err
				}
			}
		}

		// TODO Add asset fee
	}

	return settleTx, nil
}

// addBitcoinSettlements adds bitcoin settlement data to the Settlement data
func addBitcoinSettlements(ctx context.Context, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settleTx *txbuilder.Tx) error {
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
			// Get receiver address from outputs[receiver.Index]
			if int(receiver.Index) >= len(transferTx.Outputs) {
				return fmt.Errorf("Receiver output index out of range for asset %d receiver %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			// Find output in settle tx
			pkh := transferTx.Outputs[receiver.Index].Address.ScriptAddress()

			// Find output for receiver
			added := false
			for i, _ := range settleTx.MsgTx.TxOut {
				outputPKH, err := settleTx.OutputPKH(i)
				if err != nil {
					continue
				}
				if bytes.Equal(pkh, outputPKH) {
					// Add balance to receiver's output
					settleTx.AddValueToOutput(uint32(i), receiver.Quantity)
					added = true
					break
				}
			}

			if !added {
				return fmt.Errorf("Receiver bitcoin output missing output data for receiver %d", receiverOffset)
			}

			// TODO Process RegistrySignatures
			// RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string

			if receiver.Quantity >= sendBalance {
				return fmt.Errorf("Sending more bitcoin than received")
			}

			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return fmt.Errorf("Not sending all recieved bitcoins : %d remaining", sendBalance)
		}
	}

	return nil
}

// addSettlementData appends data to a pending settlement action.
func addSettlementData(ctx context.Context, masterDB *db.DB, rk *wallet.RootKey,
	transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *wire.MsgTx, settlement *protocol.Settlement) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addSettlementData")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	dbConn := masterDB
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	dataAdded := false

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
	settleInputAddresses := make([]*protocol.PublicKeyHash, 0, len(settleTx.TxIn))
	for _, input := range settleTx.TxIn {
		pushes, err := txscript.PushedData(input.SignatureScript)
		if err != nil {
			settleInputAddresses = append(settleInputAddresses, nil)
			continue
		}
		if len(pushes) != 2 {
			settleInputAddresses = append(settleInputAddresses, nil)
			continue
		}

		// Calculate RIPEMD160(SHA256(PublicKey))
		hash256.Reset()
		hash256.Write(pushes[1])
		hash160.Reset()
		hash160.Write(hash256.Sum(nil))
		settleInputAddresses = append(settleInputAddresses, protocol.PublicKeyHashFromBytes(hash160.Sum(nil)))
	}

	for assetOffset, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero() {
			continue // Skip bitcoin transfers since they should be handled already
		}

		if len(settleTx.TxOut) <= int(assetTransfer.ContractIndex) {
			return fmt.Errorf("Contract index out of range for asset %d", assetOffset)
		}

		contractOutputAddress := settleOutputAddresses[assetTransfer.ContractIndex]
		if contractOutputAddress != nil && !bytes.Equal(contractOutputAddress.Bytes(), contractAddr.Bytes()) {
			continue // This asset is not ours. Skip it.
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr, &assetTransfer.AssetCode)
		if err != nil || as == nil {
			return fmt.Errorf("Asset ID not found : %s %s : %s", contractAddr, assetTransfer.AssetCode, err)
		}

		// Find contract input
		contractInputIndex := uint16(0xffff)
		for i, input := range settleInputAddresses {
			if input != nil && bytes.Equal(input.Bytes(), contractAddr.Bytes()) {
				contractInputIndex = uint16(i)
				break
			}
		}

		if contractInputIndex == uint16(0xffff) {
			return fmt.Errorf("Contract input not found: %s %s", contractAddr, assetTransfer.AssetCode)
		}

		assetSettlement := protocol.AssetSettlement{
			ContractIndex: contractInputIndex,
			AssetType:     assetTransfer.AssetType,
			AssetCode:     assetTransfer.AssetCode,
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
			if !asset.CheckBalanceFrozen(ctx, as, inputAddress, sender.Quantity, v.Now) {
				logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr, assetTransfer.AssetCode, inputAddress)
				return frozenError
			}

			if settlementQuantities[settleOutputIndex] == nil {
				// Get sender's balance
				senderBalance := asset.GetBalance(ctx, as, inputAddress)
				settlementQuantities[settleOutputIndex] = &senderBalance
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
				receiverBalance := asset.GetBalance(ctx, as, transferOutputAddress)
				settlementQuantities[settleOutputIndex] = &receiverBalance
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

		for index, quantity := range settlementQuantities {
			if quantity != nil {
				assetSettlement.Settlements = append(assetSettlement.Settlements,
					protocol.QuantityIndex{Index: uint16(index), Quantity: *quantity})
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
	script, err := protocol.Serialize(settlement)
	if err != nil {
		return fmt.Errorf("Failed to serialize empty settlement : %s", err)
	}

	// Find Settlement OP_RETURN.
	found := false
	settlementOutputIndex := 0
	for i, output := range settleTx.TxOut {
		code, err := protocol.Code(output.PkScript)
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

	settleTx.TxOut[settlementOutputIndex].PkScript = script
	return nil
}

// sendToNextSettlementContract sends settlement data to the next contract involved so it can add its data.
func sendToNextSettlementContract(ctx context.Context, w *node.ResponseWriter, rk *wallet.RootKey,
	itx *inspector.Transaction, transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *txbuilder.Tx, settlement *protocol.Settlement) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.sendToNextSettlementContract")
	defer span.End()

	boomerangIndex := uint32(0xffffffff)
	if len(itx.Outputs) == 1 {
		// If already an M1, use only output
		boomerangIndex = 0
	} else {
		// Find "boomerang" input. The input to the first contract that will fund the passing around of
		//   the settlement data to get all contract signatures.
		for i, output := range itx.Outputs {
			if bytes.Equal(output.Address.ScriptAddress(), rk.Address.ScriptAddress()) {
				used := false
				for _, input := range settleTx.MsgTx.TxIn {
					if input.PreviousOutPoint.Index == uint32(i) {
						used = true
						break
					}
				}

				if !used {
					boomerangIndex = uint32(i)
					break
				}
			}
		}
	}

	if boomerangIndex == 0xffffffff {
		return fmt.Errorf("Multi-Contract Transfer missing boomerang output")
	}

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

		var pkh [20]byte
		copy(pkh[:], transferTx.Outputs[asset.ContractIndex].Address.ScriptAddress())

		if !currentFound {
			completedContracts[pkh] = true
			if bytes.Equal(pkh[:], rk.Address.ScriptAddress()) {
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

	v := ctx.Value(node.KeyValues).(*node.Values)

	logger.Info(ctx, "%s : Sending settlement Offer to %s", v.TraceID,
		transferTx.Outputs[nextContractIndex].Address.String())

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

	// Serialize settlement data for Message OP_RETURN.
	var payload []byte
	payload, err = protocol.Serialize(settlement)
	if err != nil {
		return err
	}
	messagePayload := protocol.Offer{
		Version:   0,
		Timestamp: v.Now,
		Payload:   payload,
	}

	err = messagePayload.RefTxId.Set(transferTx.Hash[:])
	if err != nil {
		return err
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

// settlementIsComplete returns true if the settlement accounts for all assets in the transfer.
func settlementIsComplete(ctx context.Context, transfer *protocol.Transfer, settlement *protocol.Settlement) bool {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.settlementIsComplete")
	defer span.End()

	for _, assetTransfer := range transfer.Assets {
		found := false
		for _, assetSettle := range settlement.Assets {
			if assetTransfer.AssetType == assetSettle.AssetType &&
				bytes.Equal(assetTransfer.AssetCode.Bytes(), assetSettle.AssetCode.Bytes()) {
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
	itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.SettlementResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Settlement)
	if !ok {
		return errors.New("Could not assert as *protocol.Settlement")
	}

	dbConn := t.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	for _, assetSettlement := range msg.Assets {
		if assetSettlement.AssetType == "CUR" && assetSettlement.AssetCode.IsZero() {
			continue // Bitcoin transaction
		}

		if assetSettlement.ContractIndex == 0xffff {
			continue // No contract for this asset
		}

		if int(assetSettlement.ContractIndex) >= len(itx.Inputs) {
			return fmt.Errorf("Settlement contract index out of range : %x", assetSettlement.AssetCode)
		}

		if !bytes.Equal(itx.Inputs[assetSettlement.ContractIndex].Address.ScriptAddress(), rk.Address.ScriptAddress()) {
			continue // Asset not under this contract
		}

		as, err := asset.Retrieve(ctx, dbConn, contractAddr, &assetSettlement.AssetCode)
		if err != nil || as == nil {
			logger.Warn(ctx, "%s : Asset not found: %x %x", v.TraceID, contractAddr, assetSettlement.AssetCode)
			return node.ErrNoResponse
		}

		// Create update record
		ua := asset.UpdateAsset{
			NewBalances: make(map[protocol.PublicKeyHash]uint64),
		}

		for _, settlementQuantity := range assetSettlement.Settlements {
			if int(settlementQuantity.Index) >= len(itx.Outputs) {
				return fmt.Errorf("Settlement output index out of range %d/%d : %x",
					settlementQuantity.Index, len(itx.Outputs), assetSettlement.AssetCode)
			}

			pkh := protocol.PublicKeyHashFromBytes(itx.Outputs[settlementQuantity.Index].Address.ScriptAddress())
			ua.NewBalances[*pkh] = settlementQuantity.Quantity
			logger.Verbose(ctx, "%s : Updating balance for %x to %d for asset %x", v.TraceID,
				itx.Outputs[settlementQuantity.Index].Address.String(), settlementQuantity.Quantity,
				assetSettlement.AssetCode)
		}

		// Update database
		err = asset.Update(ctx, dbConn, contractAddr, &assetSettlement.AssetCode, &ua, v.Now)
		if err != nil {
			return err
		}
	}

	return nil
}
