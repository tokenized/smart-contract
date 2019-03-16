package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
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

	msg, ok := itx.MsgProto.(*protocol.Transfer)
	if !ok {
		return errors.New("Could not assert as *protocol.Transfer")
	}

	if len(msg.Assets) == 0 {
		return errors.New("protocol.Transfer has no asset transfers")
	}

	firstContractIndex := 0

	if msg.Assets[0].ContractIndex == 0xffff {
		// First asset transfer doesn't have a contract (probably BSV transfer).
		firstContractIndex++
	}

	if msg.Assets[firstContractIndex].ContractIndex >= len(itx.Outputs) {
		logger.Info(ctx, "Transfer first contract index out of range : %s", rk.Address.String())
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	if itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Address.String() != rk.Address.String() {
		logger.Info(ctx, "Not contract for first transfer : %s", rk.Address.String())
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	contractBalance := itx.Outputs[msg.Assets[firstContractIndex].ContractIndex].Value // Bitcoin balance of first (this) contract

	// TODO Verify input amounts are appropriate. Also verify the include any bitcoins required for bitcoin transfers.

	// Build initial settlement response transaction
	settleTx := wire.MsgTx{Version: wire.TxVersion, LockTime: 0}

	// Transfer Outputs
	//   Contract 1 : amount = calculated fee for settlement tx + any bitcoins being transfered
	//   Contract 2 : dust
	//   Boomerang to Contract 1 : amount = ((n-1) * 2) * (calculated fee for data passing tx)
	//     where n is number of contracts involved
	// Boomerang is only required when more than one contract is involved.
	//
	// Each contract can be involved in more than one asset in the transfer, but only needs to have
	//   one output since each asset transfer references the output of it's contract

	outputUsed := make([]bool, len(itx.Outputs))

	// Setup inputs from outputs of the Transfer tx. One from each contract involved.
	for assetOffset, assetTransfer := range msg.Assets {
		if assetTransfer.ContractIndex == 0xffff {
			// Asset transfer doesn't have a contract (probably BSV transfer).
			continue
		}

		if assetTransfer.ContractIndex >= len(itx.Outputs) {
			logger.Warn(ctx, "%s : Transfer contract index out of range %d", v.TraceID, assetOffset)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedTransfer)
		}

		if outputUsed[assetTransfer.ContractIndex] {
			continue
		}

		// Add input from contract to settlement tx so all involved contracts have to sign for a valid tx.
		input := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{Hash: itx.Hash, Index: uint32(assetTransfer.ContractIndex)},
			// SignatureScript: , Not ready to sign yet
			Sequence: wire.MaxTxInSequenceNum, // No sequence
		}
		settleTx.AddTxIn(&input)
		outputUsed[assetTransfer.ContractIndex] = true
	}

	type OutputData struct {
		isDust bool
		index  uint32
	}

	// Index to new output for specified address
	addresses := make(map[string]*OutputData)

	// Setup outputs
	//   One to each receiver, including any bitcoins received, or dust.
	//   One to each sender with dust amount.
	for assetOffset, assetTransfer := range msg.Assets {
		assetIsBitcoin := assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode == zeroCode
		assetBalance := uint64(0)

		// Add all senders
		for _, quantityIndex := range assetTransfer.AssetSenders {
			assetBalance += quantityIndex.Quantity

			if quantityIndex.Index >= len(itx.Inputs) {
				logger.Warn(ctx, "%s : Transfer sender index out of range %d", v.TraceID, assetOffset)
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			_, exists := addresses[itx.Inputs[quantityIndex.Index].Address.String()]
			if !exists {
				// Add output to sender
				outputData := OutputData{
					isDust: true,
					index:  len(settleTx.TxOut),
				}
				addresses[itx.Inputs[quantityIndex.Index].Address.String()] = &outputData

				pkScript, err := txscript.NewScriptBuilder().
					AddOp(txscript.OP_DUP).
					AddOp(txscript.OP_HASH160).
					AddData(itx.Inputs[quantityIndex.Index].Address.ScriptAddress()).
					AddOp(txscript.OP_EQUALVERIFY).
					AddOp(txscript.OP_CHECKSIG).
					Script()
				if err != nil {
					logger.Warn(ctx, "%s : Failed to build P2PKH script : %s", v.TraceID, err)
					return err
				}

				output := wire.TxOut{Value: t.Config.DustLimit, PkScript: pkScript}
				settleTx.AddTxOut(&output)
			}
		}

		for _, tokenReceiver := range assetTransfer.AssetReceivers {
			assetBalance -= tokenReceiver.Quantity
			// TODO Handle Registry : RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string

			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if quantityIndex.Quantity > contractBalance {
					logger.Warn(ctx, "%s : Transfer sent more bitcoin than was funded to contract", v.TraceID)
					return fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= quantityIndex.Quantity
			}

			address, exists := addresses[itx.Inputs[quantityIndex.Index].Address.String()]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if address.isDust { // If bitcoins are received then dust is not needed
						address.isDust = false
						settleTx.TxOut[address.index].Value = quantityIndex.Quantity
					} else {
						settleTx.TxOut[address.index].Value += quantityIndex.Quantity
					}
				}
			} else {
				// Add output to receiver
				outputData := OutputData{
					isDust: !assetIsBitcoin,
					index:  len(settleTx.TxOut),
				}
				addresses[itx.Outputs[tokenReceiver.Index].Address.String()] = &outputData

				pkScript, err := txscript.NewScriptBuilder().
					AddOp(txscript.OP_DUP).
					AddOp(txscript.OP_HASH160).
					AddData(itx.Outputs[tokenReceiver.Index].Address.ScriptAddress()).
					AddOp(txscript.OP_EQUALVERIFY).
					AddOp(txscript.OP_CHECKSIG).
					Script()
				if err != nil {
					logger.Warn(ctx, "%s : Failed to build P2PKH script : %s", v.TraceID, err)
					return err
				}

				output := wire.TxOut{PkScript: pkScript}

				if assetIsBitcoin {
					output.Value = quantityIndex.Quantity
				} else {
					output.Value = t.Config.DustLimit
				}
				settleTx.AddTxOut(&output)
			}
		}
	}

	// Empty op return data
	settlement := protocol.Settlement{}

	if err := AddSettlementData(ctx, mux, t.MasterDB, t.Config, itx, rk, &settleTx, &settlement); err != nil {
		return err
	}

	// TODO Check if settlement data is complete
	// if complete { // No additional data needed
	// // TODO Sign tx
	// return node.RespondSuccess(ctx, mux, itx, rk, &settlement, outs)
	// }

	// TODO Send M1 - 1001 to get data from another contract

	return nil
}

// AddSettlementData appends data to a pending settlement action.
func AddSettlementData(ctx context.Context, mux protomux.Handler, masterDB *db.DB,
	config *node.Config, itx *inspector.Transaction, rk *wallet.RootKey,
	settleTx *wire.MsgTx, settlement *protocol.Settlement) error {

	dbConn := masterDB
	contractAddr := rk.Address
	v := ctx.Value(node.KeyValues).(*node.Values)

	var transferTx *inspector.Transaction
	_, ok := itx.MsgProto.(*protocol.Transfer)
	if ok {
		transferTx = itx
	} else {
		// itx is a M1 that needs more data or a signature.
		// TODO Find transfer transaction for data
		// Transfer tx is in the inputs of the Settlement tx.

	}

	for assetOffset, assetTransfer := range msg.Assets {
		if len(settleTx.Outputs) <= assetTransfer.ContractIndex {
			logger.Warn(ctx, "%s : Not enough outputs for transfer %d", v.TraceID, assetOffset)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedSettle)
		}

		contractOutput := settleTx.Outputs[assetTransfer.ContractIndex]
		if contractOutput.Address.String() != contractAddr.String() {
			continue // This asset is not ours. Skip it.
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetTransfer.AssetID)
		if err != nil || as == nil {
			logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), assetTransfer.AssetID)
			return err
		}

		// Find contract input
		contractInputOffset := uint32(0xffffffff)
		for i, input := range settleTx.Inputs {
			if input.Address.String() == contractAddr.String() {
				contractInputOffset = i
				break
			}
		}

		if contractInputOffset == 0xffffffff {
			logger.Warn(ctx, "%s : Contract input not found: %s %s", v.TraceID, contractAddr.String(), assetTransfer.AssetID)
			return err
		}

		assetSettlement := protocol.AssetSettlement{
			ContractIndex: contractInputOffset,
			AssetType:     assetTransfer.AssetType,
			AssetID:       assetTransfer.AssetID,
		}

		sendBalance := uint64(0)

		// Process senders
		// assetTransfer.AssetSenders []QuantityIndex {Index uint16, Quantity uint64}
		for senderOffset, sender := range assetTransfer.AssetSenders {
			// Get sender address from transfer inputs[sender.Index]
			if sender.Index >= len(transferTx.Inputs) {
				logger.Warn(ctx, "%s : Sender input index out of range for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, senderOffset, sender.Index, len(transferTx.Inputs))
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			input := transferTx.Inputs[sender.Index]
			inputAddress = input.Address.String()

			// Find output in settle tx
			settleOutputIndex := uint32(0xffffffff)
			for i, output := range settleTx.Outputs {
				if output.Address.String() == inputAddress {
					settleOutputIndex = i
					break
				}
			}

			if settleOutputIndex == 0xffffffff {
				logger.Warn(ctx, "%s : Sender output not found in settle tx for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, senderOffset, sender.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedSettle)
			}

			// Check sender's available unfrozen balance
			if !asset.CheckBalanceFrozen(ctx, as, input.Address.String(), sender.Quantity, v.Now) {
				logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID,
					contractAddr.String(), assetTransfer.AssetID, input.Address.String())
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFrozen)
			}

			// Get sender's balance
			senderBalance := asset.GetBalance(ctx, as, input.Address.String())

			// assetSettlement.Settlements []QuantityIndex {Index uint16, Quantity uint64}
			assetSettlement.Settlements = append(assetSettlement.Settlements, protocol.QuantityIndex{Index: settleOutputIndex, Quantity: senderBalance - sender.Quantity})

			// Update total send balance
			sendBalance += sender.Quantity
		}

		totalSendBalance := sendBalance

		// Process receivers
		// assetTransfer.AssetReceivers []TokenReceiver {Index uint16, Quantity uint64, RegistrySigAlgorithm uint8, RegistryConfirmationSigToken string}
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Get receiver address from outputs[receiver.Index]
			if receiver.Index >= len(transferTx.Outputs) {
				logger.Warn(ctx, "%s : Receiver output index out of range for asset %d sender %d : %d/%d", v.TraceID,
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}

			output := transferTx.Outputs[receiver.Index]
			outputAddress := output.Address.String()

			// Find output in settle tx
			settleOutputIndex := uint32(0xffffffff)
			for i, output := range settleTx.Outputs {
				if output.Address.String() == outputAddress {
					settleOutputIndex = i
					break
				}
			}

			if settleOutputIndex == 0xffffffff {
				logger.Warn(ctx, "%s : Receiver output not found in settle tx for asset %d receiver %d : %d/%d", v.TraceID,
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedSettle)
			}

			// TODO Process RegistrySignatures

			// Get receiver's balance
			receiverBalance := asset.GetBalance(ctx, as, output.Address.String())

			assetSettlement.Settlements = append(assetSettlement.Settlements, protocol.QuantityIndex{Index: settleOutputIndex, Quantity: receiverBalance + receiver.Quantity})

			// Update asset balance
			if sender.Quantity > sendBalance {
				logger.Warn(ctx, "%s : Receiving more tokens than sending for asset %d", v.TraceID, assetOffset)
				return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeMalFormedTransfer)
			}
			sendBalance -= sender.Quantity
		}

		settlement.Assets = append(settlement.Assets, assetSettlement)
	}

	// // Validate transaction
	// if len(itx.Inputs) < 2 {
	// logger.Warn(ctx, "%s : Not enough inputs: %s %s", v.TraceID, contractAddr, assetID)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeReceiverUnspecified)
	// }

	// if len(itx.Outputs) < 3 {
	// logger.Warn(ctx, "%s : Not enough outputs: %s %s", v.TraceID, contractAddr, assetID)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeReceiverUnspecified)
	// }

	// // Party 1 (Sender), Party 2 (Receiver)
	// party1Addr := itx.Inputs[0].Address
	// party2Addr := itx.Inputs[1].Address

	// // Cannot transfer to self
	// if party1Addr.String() == party2Addr.String() {
	// logger.Warn(ctx, "%s : Cannot transfer to own self : contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, party1Addr)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeTransferSelf)
	// }

	// // Check available balance
	// if !asset.CheckBalance(ctx, as, party1Addr.String(), msg.Party1TokenQty) {
	// logger.Warn(ctx, "%s : Insufficient funds: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, party1Addr)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	// }

	// // Check available unfrozen balance
	// if !asset.CheckBalanceFrozen(ctx, as, party1Addr.String(), msg.Party1TokenQty, v.Now) {
	// logger.Warn(ctx, "%s : Frozen funds: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, party1Addr)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFrozen)
	// }

	// logger.Info(ctx, "%s : Accepting exchange request : %s %s %d tokens\n", v.TraceID, contractAddr, assetID, msg.Party1TokenQty)

	// // Find balances
	// party1Balance := asset.GetBalance(ctx, as, party1Addr.String())
	// party2Balance := asset.GetBalance(ctx, as, party2Addr.String())

	// // Modify balances
	// party1Balance -= msg.Party1TokenQty
	// party2Balance += msg.Party1TokenQty

	// // Settlement <- Exchange
	// settlement := protocol.NewSettlement()
	// settlement.AssetType = msg.Party1AssetType
	// settlement.AssetID = msg.Party1AssetID
	// settlement.Party1TokenQty = party1Balance
	// settlement.Party2TokenQty = party2Balance
	// settlement.Timestamp = uint64(v.Now.Unix())

	// // Build outputs
	// // 1 - Party 1 Address (Change + Value)
	// // 2 - Party 2 Address
	// // 3 - Contract Address
	// // 4 - Fee
	// outs := []node.Output{{
	// Address: party1Addr,
	// Value:   t.Config.DustLimit,
	// Change:  true,
	// }, {
	// Address: party2Addr,
	// Value:   t.Config.DustLimit,
	// }, {
	// Address: contractAddr,
	// Value:   t.Config.DustLimit,
	// }}

	// // Add fee output
	// if fee := node.OutputFee(ctx, t.Config); fee != nil {
	// outs = append(outs, *fee)
	// }

	// // Optional exchange fee.
	// if msg.ExchangeFeeFixed > 0 {
	// exAddr := string(msg.ExchangeFeeAddress)
	// addr, err := btcutil.DecodeAddress(exAddr, &chaincfg.MainNetParams)
	// if err != nil {
	// return err
	// }

	// // Convert BCH to Satoshi's
	// exo := node.Output{
	// Address: addr,
	// Value:   txbuilder.ConvertBCHToSatoshis(msg.ExchangeFeeFixed),
	// }

	// outs = append(outs, exo)
	// }

	// Respond with a settlement
	// TODO Specify Input addresses to use. inputs
	return node.RespondSuccess(ctx, mux, itx, rk, &settlement, outs)
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
	// assetID := string(msg.AssetID)
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

	// logger.Info(ctx, "%s : Settling transfer : %s %s\n", v.TraceID, contractAddr.String(), string(msg.AssetID))
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
}
