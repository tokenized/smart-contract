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
	"github.com/tokenized/smart-contract/internal/platform/protomux"
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

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
	//   creating the initial settlement transaction and passing it to the next contract if there
	//   are more than one.
	firstContractIndex := uint16(0)

	if msg.Assets[0].ContractIndex == uint16(0xffff) {
		// First asset transfer doesn't have a contract (probably BSV transfer).
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

	// Settle Outputs
	//   Any addresses sending or receiving tokens or bitcoin.
	//   Referenced from indices from within settlement data.
	//
	// Settle Inputs
	//   Any contracts involved.

	// Build initial settlement response transaction
	settleTx := txbuilder.NewTx(rk.Address.ScriptAddress(), t.Config.DustLimit, t.Config.FeeRate)

	outputUsed := make([]bool, len(itx.Outputs))

	// Setup inputs from outputs of the Transfer tx. One from each contract involved.
	for assetOffset, assetTransfer := range msg.Assets {
		if assetTransfer.ContractIndex == uint16(0xffff) ||
			(assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()) {
			continue
		}

		if int(assetTransfer.ContractIndex) >= len(itx.Outputs) {
			logger.Warn(ctx, "%s : Transfer contract index out of range %d", v.TraceID, assetOffset)
			return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
		}

		if outputUsed[assetTransfer.ContractIndex] {
			continue
		}

		// Add input from contract to settlement tx so all involved contracts have to sign for a valid tx.
		err = settleTx.AddInput(wire.OutPoint{Hash: itx.Hash, Index: uint32(assetTransfer.ContractIndex)},
			itx.Outputs[assetTransfer.ContractIndex].UTXO.PkScript, uint64(itx.Outputs[assetTransfer.ContractIndex].Value))
		if err != nil {
			return err
		}
		outputUsed[assetTransfer.ContractIndex] = true
	}

	// Index to new output for specified address
	addressesMap := make(map[protocol.PublicKeyHash]uint32)

	// Setup outputs
	//   One to each receiver, including any bitcoins received, or dust.
	//   One to each sender with dust amount.
	for assetOffset, assetTransfer := range msg.Assets {
		assetIsBitcoin := assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero()
		assetBalance := uint64(0)

		// Add all senders
		for _, quantityIndex := range assetTransfer.AssetSenders {
			assetBalance += quantityIndex.Quantity

			if quantityIndex.Index >= uint16(len(itx.Inputs)) {
				logger.Warn(ctx, "%s : Transfer sender index out of range %d", v.TraceID, assetOffset)
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			address := protocol.PublicKeyHashFromBytes(itx.Inputs[quantityIndex.Index].Address.ScriptAddress())
			_, exists := addressesMap[address]
			if !exists {
				// Add output to sender
				addressesMap[address] = uint32(len(settleTx.MsgTx.TxOut))

				err = settleTx.AddP2PKHDustOutput(itx.Inputs[quantityIndex.Index].Address.ScriptAddress(), false)
				if err != nil {
					return err
				}
			}
		}

		for _, tokenReceiver := range assetTransfer.AssetReceivers {
			assetBalance -= tokenReceiver.Quantity
			// TODO Handle Registry : RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string

			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if tokenReceiver.Quantity > contractBalance {
					logger.Warn(ctx, "%s : Transfer sent more bitcoin than was funded to contract", v.TraceID)
					return fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= tokenReceiver.Quantity
			}

			address := protocol.PublicKeyHashFromBytes(itx.Inputs[tokenReceiver.Index].Address.ScriptAddress())
			outputIndex, exists := addressesMap[address]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err := settleTx.AddValueToOutput(outputIndex, tokenReceiver.Quantity); err != nil {
						return err
					}
				}
			} else {
				// Add output to receiver
				addressesMap[address] = uint32(len(settleTx.MsgTx.TxOut))

				if assetIsBitcoin {
					err = settleTx.AddP2PKHOutput(itx.Outputs[tokenReceiver.Index].Address.ScriptAddress(), tokenReceiver.Quantity, false)
				} else {
					err = settleTx.AddP2PKHDustOutput(itx.Outputs[tokenReceiver.Index].Address.ScriptAddress(), false)
				}
				if err != nil {
					return err
				}
			}
		}
	}

	// Create initial settlement data
	settlement := protocol.Settlement{Timestamp: v.Now}

	// Check for bitcoin transfers.
	for assetOffset, assetTransfer := range msg.Assets {
		if assetTransfer.AssetType != "CUR" || !assetTransfer.AssetCode.IsZero() {
			continue
		}

		sendBalance := uint64(0)

		// Process senders
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if int(sender.Index) >= len(itx.Inputs) {
				logger.Warn(ctx, "%s : Sender input index out of range for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, senderOffset, sender.Index, len(itx.Inputs))
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			input := itx.Inputs[sender.Index]

			// Get sender's balance
			if int64(sender.Quantity) >= input.Value {
				logger.Warn(ctx, "%s : Sender bitcoin quantity higher than input amount for sender %d : %d/%d", v.TraceID,
					senderOffset, input.Value, sender.Quantity)
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Get receiver address from outputs[receiver.Index]
			if int(receiver.Index) >= len(itx.Outputs) {
				logger.Warn(ctx, "%s : Receiver output index out of range for asset %d receiver %d : %d/%d", v.TraceID,
					assetOffset, receiverOffset, receiver.Index, len(itx.Outputs))
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			// Find output in settle tx
			address := protocol.PublicKeyHashFromBytes(itx.Outputs[receiver.Index].Address.ScriptAddress())
			outputIndex, ok := addressesMap[address]
			if !ok {
				logger.Warn(ctx, "%s : Receiver bitcoin output missing output data for receiver %d", v.TraceID, receiverOffset)
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			// Add balance to receiver's output
			settleTx.AddValueToOutput(outputIndex, receiver.Quantity)

			// TODO Process RegistrySignatures
			// RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string

			if receiver.Quantity >= sendBalance {
				logger.Warn(ctx, "%s : Sending more bitcoin than received")
				return node.RespondReject(ctx, mux, t.Config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			logger.Warn(ctx, "%s : Not sending all recieved bitcoins : %d remaining", v.TraceID, sendBalance)
		}
	}

	// Serialize settlement data into OP_RETURN output.
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
	if err := AddSettlementData(ctx, mux, t.MasterDB, t.Config, itx, rk, &settleTx.MsgTx); err != nil {
		return err
	}

	// Check if settlement data is complete. No other contracts involved
	if settlementIsComplete(ctx, msg, &settlement) {
		if err := settleTx.Sign([]*btcec.PrivateKey{rk.PrivateKey}); err != nil {
			return err
		}

		return node.Respond(ctx, mux, t.Config, &settleTx.MsgTx)
	}

	// Find "boomerang" input. The input to the first contract that will fund the passing around of
	//   the settlement data to get all contract signatures.
	boomerangIndex := uint32(0xffffffff)
	for i, used := range outputUsed {
		if !used && bytes.Equal(itx.Outputs[i].Address.ScriptAddress(), rk.Address.ScriptAddress()) {
			boomerangIndex = uint32(i)
			break
		}
	}

	if boomerangIndex == 0xffffffff {
		logger.Warn(ctx, "%s : Multi-Contract Transfer missing boomerang output", v.TraceID)
		return fmt.Errorf("Multi-Contract Transfer missing boomerang output")
	}

	// Find next contract
	nextContractIndex := uint16(0xffff)

	for _, assetTransfer := range msg.Assets {
		if assetTransfer.ContractIndex == uint16(0xffff) {
			// Asset transfer doesn't have a contract (probably BSV transfer).
			continue
		}

		if assetTransfer.ContractIndex == firstContractIndex {
			continue
		}

		if int(assetTransfer.ContractIndex) >= len(itx.Outputs) {
			logger.Warn(ctx, "Transfer contract index out of range")
			return errors.New("Transfer contract index out of range")
		}

		nextContractIndex = msg.Assets[firstContractIndex].ContractIndex
		break
	}

	if nextContractIndex == 0xffff {
		logger.Warn(ctx, "%s : Next contract not found in multi-contract transfer", v.TraceID)
		return fmt.Errorf("Next contract not found in multi-contract transfer")
	}

	// Create message to next contract
	m1Tx := txbuilder.NewTx(itx.Outputs[nextContractIndex].Address.ScriptAddress(),
		t.Config.DustLimit, t.Config.FeeRate)

	// Add boomerang funding as input.
	err = m1Tx.AddInput(wire.OutPoint{Hash: itx.Hash, Index: boomerangIndex},
		itx.Outputs[boomerangIndex].UTXO.PkScript, uint64(itx.Outputs[boomerangIndex].UTXO.Value))
	if err != nil {
		return err
	}

	// Add output to next contract.
	// Mark as change so the tx fee will be taken from it.
	err = m1Tx.AddP2PKHOutput(itx.Outputs[nextContractIndex].Address.ScriptAddress(),
		uint64(itx.Outputs[boomerangIndex].UTXO.Value), true)
	if err != nil {
		return err
	}

	// TODO Serialize settlement tx into M1 1001 OP_RETURN output
	var buf bytes.Buffer
	err = settleTx.MsgTx.Serialize(&buf)
	if err != nil {
		return err
	}
	m1Payload := protocol.Offer{
		Version:   0,
		Timestamp: protocol.CurrentTimestamp(),
		Offer:     buf.Bytes(),
	}

	// Setup M1 Payload Data
	var data []byte
	data, err = m1Payload.Serialize()
	if err != nil {
		return err
	}
	m1Data := protocol.Message{
		AddressIndexes: []uint16{0}, // First output is receiver of message
		MessageType:    protocol.CodeOffer,
		MessagePayload: data,
	}

	// Setup M1 Data
	script, err = protocol.Serialize(&m1Data)
	if err != nil {
		return err
	}

	// Add output to M1 tx for message.
	err = m1Tx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	// Sign input
	err = m1Tx.Sign([]*btcec.PrivateKey{rk.PrivateKey})
	if err != nil {
		return err
	}

	return node.Respond(ctx, mux, t.Config, &m1Tx.MsgTx)
}

// AddSettlementData appends data to a pending settlement action.
func AddSettlementData(ctx context.Context, mux protomux.Handler, masterDB *db.DB,
	config *node.Config, itx *inspector.Transaction, rk *wallet.RootKey,
	settleTx *wire.MsgTx) error {

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
		logger.Warn(ctx, "%s : Settlement op return not found in settle tx", v.TraceID)
		return fmt.Errorf("Settlement op return not found in settle tx")
	}

	dbConn := masterDB
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	var transferTx *inspector.Transaction
	transfer, ok := itx.MsgProto.(*protocol.Transfer)
	if ok {
		transferTx = itx
	} else {
		// itx is a M1 that needs more data or a signature.
		// TODO Find transfer transaction for data
		// Transfer tx is in the inputs of the Settlement tx.

	}

	dataAdded := false

	// Generate public key hashes for all the outputs
	settleOutputAddresses := make([]protocol.PublicKeyHash, 0, len(settleTx.TxOut))
	for _, output := range settleTx.TxOut {
		class, addresses, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, &config.ChainParams)
		if err != nil {
			continue
		}
		if class == txscript.PubKeyHashTy {
			settleOutputAddresses = append(settleOutputAddresses, protocol.PublicKeyHashFromBytes(addresses[0].ScriptAddress()))
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
			logger.Warn(ctx, "%s : Not enough outputs for transfer %d", v.TraceID, assetOffset)
			return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedSettle)
		}

		contractOutputAddress := settleOutputAddresses[assetTransfer.ContractIndex]
		if !bytes.Equal(contractOutputAddress.Bytes(), contractAddr.Bytes()) {
			continue // This asset is not ours. Skip it.
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr, assetTransfer.AssetCode)
		if err != nil || as == nil {
			logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetTransfer.AssetCode)
			return err
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
			logger.Warn(ctx, "%s : Contract input not found: %s %s", v.TraceID, contractAddr, assetTransfer.AssetCode)
			return err
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
				logger.Warn(ctx, "%s : Sender input index out of range for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
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
				logger.Warn(ctx, "%s : Sender output not found in settle tx for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, senderOffset, sender.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedSettle)
			}

			// Check sender's available unfrozen balance
			inputAddress := protocol.PublicKeyHashFromBytes(input.Address.ScriptAddress())
			if !asset.CheckBalanceFrozen(ctx, as, inputAddress, sender.Quantity, v.Now) {
				logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr, assetTransfer.AssetCode, inputAddress)
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeFrozen)
			}

			// Get sender's balance
			senderBalance := asset.GetBalance(ctx, as, inputAddress)

			// assetSettlement.Settlements []QuantityIndex {Index uint16, Quantity uint64}
			assetSettlement.Settlements = append(assetSettlement.Settlements, protocol.QuantityIndex{Index: settleOutputIndex, Quantity: senderBalance - sender.Quantity})

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		// assetTransfer.AssetReceivers []TokenReceiver {Index uint16, Quantity uint64, RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string}
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Get receiver address from outputs[receiver.Index]
			if int(receiver.Index) >= len(transferTx.Outputs) {
				logger.Warn(ctx, "%s : Receiver output index out of range for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
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
				logger.Warn(ctx, "%s : Receiver output not found in settle tx for asset %d receiver %d : %d/%d", v.TraceID,
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedSettle)
			}

			// TODO Process RegistrySignatures

			// Get receiver's balance
			receiverBalance := asset.GetBalance(ctx, as, transferOutputAddress)

			assetSettlement.Settlements = append(assetSettlement.Settlements, protocol.QuantityIndex{Index: settleOutputIndex, Quantity: receiverBalance + receiver.Quantity})

			// Update asset balance
			if receiver.Quantity > sendBalance {
				logger.Warn(ctx, "%s : Receiving more tokens than sending for asset %d", v.TraceID, assetOffset)
				return node.RespondReject(ctx, mux, config, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}
			sendBalance -= receiver.Quantity
		}

		if sendBalance != 0 {
			logger.Warn(ctx, "%s : Not sending all input tokens for asset %d : %d remaining", v.TraceID, assetOffset, sendBalance)
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
		logger.Warn(ctx, "%s : Failed to serialize empty settlement : %s", v.TraceID, err)
		return err
	}
	settleTx.TxOut[settlementOutputIndex].PkScript = script
	return nil
}

// SettlementResponse handles an outgoing Settlement action and writes it to the state
func (t *Transfer) SettlementResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
