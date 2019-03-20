package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
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

	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

type Transfer struct {
	MasterDB *db.DB
	Config   *node.Config
}

var (
	frozenError = errors.New("Holdings Frozen")
)

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return errors.New("Transfer Request Not Implemented")
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Transfer")
	defer span.End()

	var err error
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
		logger.Info(ctx, "Transfer contract index out of range : %s", rk.Address.String())
		return errors.New("Transfer contract index out of range")
	}

	if itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Address.String() != rk.Address.String() {
		logger.Info(ctx, "Not contract for first transfer : %s", rk.Address.String())
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	contractBalance := uint64(itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Value) // Bitcoin balance of first (this) contract

	// TODO Verify input amounts are appropriate. Also verify the include any bitcoins required for bitcoin transfers.

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
	settleTx, err := BuildSettlementTx(ctx, t.Config, itx, msg, rk, contractBalance)
	if err != nil {
		logger.Warn(ctx, "Failed to build settlement tx : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Update outputs to pay bitcoin receivers.
	err = AddBitcoinSettlements(ctx, itx, msg, settleTx)
	if err != nil {
		logger.Warn(ctx, "Failed to add bitcoin settlements : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalFormedTransfer)
	}

	// Create initial settlement data
	settlement := protocol.Settlement{Timestamp: v.Now}

	// Serialize emtpy settlement data into OP_RETURN output.
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
	err = AddSettlementData(ctx, t.MasterDB, itx, msg, &settleTx.MsgTx, rk)
	if err == frozenError {
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeFrozen)
	}
	if err != nil {
		return err
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, msg, &settlement) {
		if err := settleTx.Sign([]*btcec.PrivateKey{rk.PrivateKey}); err != nil {
			return err
		}

		return node.Respond(ctx, w, &settleTx.MsgTx)
	}

	// Send to next contract
	return SendToNextContract(ctx, w, rk, itx, itx, msg, settleTx, &settlement)
}

// BuildSettlementTx builds the tx for a settlement action.
func BuildSettlementTx(ctx context.Context, config *node.Config, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, rk *wallet.RootKey, contractBalance uint64) (*txbuilder.Tx, error) {
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
			_, exists := addresses[address]
			if !exists {
				// Add output to sender
				addresses[address] = uint32(len(settleTx.MsgTx.TxOut))

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
			outputIndex, exists := addresses[address]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, tokenReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver
				addresses[address] = uint32(len(settleTx.MsgTx.TxOut))

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
	}

	return settleTx, nil
}

// AddBitcoinSettlements adds bitcoin settlement data to the Settlement data
func AddBitcoinSettlements(ctx context.Context, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settleTx *txbuilder.Tx) error {
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

// AddSettlementData appends data to a pending settlement action.
func AddSettlementData(ctx context.Context, masterDB *db.DB, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settleTx *wire.MsgTx, rk *wallet.RootKey) error {
	v := ctx.Value(node.KeyValues).(*node.Values)

	var settlement *protocol.Settlement
	found := false
	settlementOutputIndex := 0
	for i, output := range settleTx.TxOut {
		msg, err := protocol.Deserialize(output.PkScript)
		if err != nil {
			continue
		}
		ok := false
		settlement, ok = msg.(*protocol.Settlement)
		if ok {
			settlementOutputIndex = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Settlement op return not found in settle tx")
	}

	dbConn := masterDB
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	dataAdded := false

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]protocol.PublicKeyHash, 0, len(settleTx.TxOut))
	for _, output := range settleTx.TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			settleOutputAddresses = append(settleOutputAddresses, protocol.PublicKeyHashFromBytes(pkh))
		} else {
			settleOutputAddresses = append(settleOutputAddresses, protocol.PublicKeyHash{})
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
		if !bytes.Equal(contractOutputAddress.Bytes(), contractAddr.Bytes()) {
			continue // This asset is not ours. Skip it.
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr, assetTransfer.AssetCode)
		if err != nil || as == nil {
			return fmt.Errorf("Asset ID not found : %s %s : %s", contractAddr, assetTransfer.AssetCode, err)
		}

		// Find contract input
		contractInputOffset := uint16(0xffff)
		for i, input := range settleInputAddresses {
			if bytes.Equal(input.Bytes(), contractAddr.Bytes()) {
				contractInputOffset = uint16(i)
				break
			}
		}

		if contractInputOffset == uint16(0xffff) {
			return fmt.Errorf("Contract input not found: %s %s", contractAddr, assetTransfer.AssetCode)
		}

		assetSettlement := protocol.AssetSettlement{
			ContractIndex: contractInputOffset,
			AssetType:     assetTransfer.AssetType,
			AssetCode:     assetTransfer.AssetCode,
		}

		sendBalance := uint64(0)

		// Process senders
		// assetTransfer.AssetSenders []QuantityIndex {Index uint16, Quantity uint64}
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(transferTx.Inputs) {
				return fmt.Errorf("Sender input index out of range for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
			}

			input := transferTx.Inputs[sender.Index]

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if bytes.Equal(outputAddress.Bytes(), input.Address.ScriptAddress()) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Sender output not found in settle tx for asset %d sender %d : %d/%d",
					assetOffset, senderOffset, sender.Index, len(transferTx.Outputs))
			}

			// Check sender's available unfrozen balance
			inputAddress := protocol.PublicKeyHashFromBytes(input.Address.ScriptAddress())
			if !asset.CheckBalanceFrozen(ctx, as, inputAddress, sender.Quantity, v.Now) {
				logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr, assetTransfer.AssetCode, inputAddress)
				return frozenError
			}

			// Get sender's balance
			senderBalance := asset.GetBalance(ctx, as, inputAddress)

			// assetSettlement.Settlements []QuantityIndex {Index uint16, Quantity uint64}
			assetSettlement.Settlements = append(assetSettlement.Settlements,
				protocol.QuantityIndex{Index: settleOutputIndex, Quantity: senderBalance - sender.Quantity})

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
				if bytes.Equal(outputAddress.Bytes(), transferOutputAddress.Bytes()) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Receiver output not found in settle tx for asset %d receiver %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			// TODO Process RegistrySignatures

			// Get receiver's balance
			receiverBalance := asset.GetBalance(ctx, as, transferOutputAddress)

			assetSettlement.Settlements = append(assetSettlement.Settlements,
				protocol.QuantityIndex{Index: settleOutputIndex, Quantity: receiverBalance + receiver.Quantity})

			// Update asset balance
			if receiver.Quantity > sendBalance {
				return fmt.Errorf("Receiving more tokens than sending for asset %d", assetOffset)
			}
			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			return fmt.Errorf("Not sending all input tokens for asset %d : %d remaining", assetOffset, sendBalance)
		}

		settlement.Assets = append(settlement.Assets, assetSettlement)
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
	settleTx.TxOut[settlementOutputIndex].PkScript = script
	return nil
}

func SendToNextContract(ctx context.Context, w *node.ResponseWriter, rk *wallet.RootKey,
	itx *inspector.Transaction, transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *txbuilder.Tx, settlement *protocol.Settlement) error {
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

	v := ctx.Value(node.KeyValues).(*node.Values)

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
		MessageType:    protocol.CodeOffer,
		MessagePayload: data,
	}

	return node.RespondSuccess(ctx, w, itx, rk, &message)
}

// TODO CheckSettlement verifies that all settlement data related to this contract and bitcoin transfers are correct.
func CheckSettlement(ctx context.Context, masterDB *db.DB, transferTx *inspector.Transaction,
	transfer *protocol.Transfer, settleTx *wire.MsgTx, rk *wallet.RootKey) error {

	return nil
}

// SettlementResponse handles an outgoing Settlement action and writes it to the state
func (t *Transfer) SettlementResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return errors.New("Settlement Response Not Implemented")
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Settlement")
	defer span.End()

	// msg, ok := itx.MsgProto.(*protocol.Settlement)
	// if !ok {
	// return errors.New("Could not assert as *protocol.Settlement")
	// }

	// dbConn := t.MasterDB

	// v := ctx.Value(node.KeyValues).(*node.Values)

	// // Locate Asset
	// contractAddr := rk.Address
	// assetID := string(msg.AssetCode)
	// as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	// if err != nil {
	// return err
	// }

	// // Asset could not be found
	// if as == nil {
	// logger.Warn(ctx, "%s : Asset ID not found: %s %s\n", v.TraceID, contractAddr, assetID)
	// return node.ErrNoResponse
	// }

	// // Validate transaction
	// if len(itx.Outputs) < 2 {
	// logger.Warn(ctx, "%s : Not enough outputs: %s %s\n", v.TraceID, contractAddr, assetID)
	// return node.ErrNoResponse
	// }

	// // Party 1 (Sender), Party 2 (Receiver)
	// party1PKH := itx.Outputs[0].Address.String()
	// party2PKH := itx.Outputs[1].Address.String()

	// logger.Info(ctx, "%s : Settling transfer : %s %s\n", v.TraceID, contractAddr.String(), string(msg.AssetCode))
	// logger.Info(ctx, "%s : Party 1 %s : %d tokens\n", v.TraceID, party1PKH, msg.Party1TokenQty)
	// logger.Info(ctx, "%s : Party 2 %s : %d tokens\n", v.TraceID, party2PKH, msg.Party2TokenQty)

	// newBalances := map[string]uint64{
	// party1PKH: msg.Party1TokenQty,
	// party2PKH: msg.Party2TokenQty,
	// }

	// // Update asset
	// ua := asset.UpdateAsset{
	// NewBalances: newBalances,
	// }

	// if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
	// return err
	// }

	// return nil
	return errors.New("Not implemented")
}

// settlementIsComplete returns true if the settlement accounts for all assets in the transfer.
func settlementIsComplete(ctx context.Context, transfer *protocol.Transfer, settlement *protocol.Settlement) bool {
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
