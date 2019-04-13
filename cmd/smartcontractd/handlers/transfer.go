package handlers

import (
	"bytes"
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"fmt"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

type Transfer struct {
	handler   *node.App
	MasterDB  *db.DB
	Config    *node.Config
	Headers   BitcoinHeaders
	Tracer    *listeners.Tracer
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
		return value.Text
	}
	return fmt.Sprintf("%s - %s", value.Text, err.text)
}

// TransferRequest handles an incoming Transfer request.
func (t *Transfer) TransferRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
		logger.Warn(ctx, "%s : Transfer first contract not found : %s", v.TraceID, rk.Address.String())
		return errors.New("Transfer first contract not found")
	}

	if !bytes.Equal(itx.Outputs[first].Address.ScriptAddress(), rk.Address.ScriptAddress()) {
		logger.Verbose(ctx, "%s : Not contract for first transfer. Waiting for Message Offer : %s", v.TraceID,
			itx.Outputs[first].Address.String())
		if err := transactions.AddTx(ctx, t.MasterDB, itx); err != nil {
			return errors.Wrap(err, "Failed to save tx")
		}
		return nil // Wait for M1 - 1001 requesting data to complete Settlement tx.
	}

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Transfer invalid : %s", v.TraceID, err)
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectMsgMalformed, false)
	}

	if msg.OfferExpiry.Nano() > v.Now.Nano() {
		logger.Warn(ctx, "%s : Transfer expired : %s", v.TraceID, msg.OfferExpiry.String())
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectTransferExpired, false)
	}

	if len(msg.Assets) == 0 {
		logger.Warn(ctx, "%s : Transfer has no asset transfers", v.TraceID)
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectTransferExpired, false)
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
	var settleTx *txbuilder.Tx
	settleTx, err = buildSettlementTx(ctx, t.MasterDB, t.Config, itx, msg, &settlementRequest, contractBalance, rk)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to build settlement tx : %s", v.TraceID, err)
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectMsgMalformed, false)
	}

	// Update outputs to pay bitcoin receivers.
	err = addBitcoinSettlements(ctx, itx, msg, settleTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to add bitcoin settlements : %s", v.TraceID, err)
		return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, protocol.RejectMsgMalformed, false)
	}

	// Create initial settlement data
	settlement := protocol.Settlement{Timestamp: v.Now}

	// Serialize empty settlement data into OP_RETURN output as a placeholder to be updated by addSettlementData.
	var script []byte
	script, err = protocol.Serialize(&settlement)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to serialize settlement : %s", v.TraceID, err)
		return err
	}
	err = settleTx.AddOutput(script, 0, false, false)
	if err != nil {
		return err
	}

	// Add this contract's data to the settlement op return data
	err = addSettlementData(ctx, t.MasterDB, rk, itx, msg, settleTx.MsgTx, &settlement, t.Headers)
	if err != nil {
		reject, ok := err.(rejectError)
		if ok {
			logger.Warn(ctx, "%s : Rejecting Transfer : %s", v.TraceID, err)
			return respondTransferReject(ctx, t.MasterDB, t.Config, w, itx, msg, rk, reject.code, false)
		} else {
			return errors.Wrap(err, "Failed to add settlement data")
		}
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
	if err := transactions.AddTx(ctx, t.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Send to next contract
	if err := sendToNextSettlementContract(ctx, w, rk, itx, itx, msg, settleTx, &settlement, &settlementRequest, t.Tracer); err != nil {
		return err
	}

	// Save pending transfer
	timeout := protocol.NewTimestamp(v.Now.Nano() + t.Config.RequestTimeout)
	pendingTransfer := state.PendingTransfer{TransferTxId: *protocol.TxIdFromBytes(itx.Hash[:]), Timeout: timeout}
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if err := transfer.Save(ctx, t.MasterDB, contractPKH, &pendingTransfer); err != nil {
		return errors.Wrap(err, "Failed to save pending transfer")
	}

	// Schedule timeout for transfer in case the other contract(s) don't respond.
	if err := t.Scheduler.ScheduleJob(ctx, listeners.NewTransferTimeout(t.handler, itx, timeout)); err != nil {
		return errors.Wrap(err, "Failed to schedule transfer timeout")
	}

	return nil
}

// TransferTimeout is called when a multi-contract transfer times out because the other contracts are not responding.
func (t *Transfer) TransferTimeout(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.TransferTimeout")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Transfer)
	if !ok {
		return errors.New("Could not assert as *protocol.Transfer")
	}

	// Remove pending transfer
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if err := transfer.Remove(ctx, t.MasterDB, contractPKH, protocol.TxIdFromBytes(itx.Hash[:])); err != nil {
		if err != transfer.ErrNotFound {
			return errors.Wrap(err, "Failed to remove pending transfer")
		}
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	logger.Warn(ctx, "%s : Transfer timed out", v.TraceID)
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
	transfer *protocol.Transfer, settlementRequest *protocol.SettlementRequest, contractBalance uint64, rk *wallet.RootKey) (*txbuilder.Tx, error) {
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

			receiverPKH := protocol.PublicKeyHashFromBytes(transferTx.Inputs[tokenReceiver.Index].Address.ScriptAddress())
			if assetIsBitcoin {
				// Debit from contract's bitcoin balance
				if tokenReceiver.Quantity > contractBalance {
					return nil, fmt.Errorf("Transfer sent more bitcoin than was funded to contract")
				}
				contractBalance -= tokenReceiver.Quantity
			}

			outputIndex, exists := addresses[*receiverPKH]
			if exists {
				if assetIsBitcoin {
					// Add bitcoin quantity to receiver's output
					if err = settleTx.AddValueToOutput(outputIndex, tokenReceiver.Quantity); err != nil {
						return nil, err
					}
				}
			} else {
				// Add output to receiver
				addresses[*receiverPKH] = uint32(len(settleTx.MsgTx.TxOut))

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

	// Add other contract's fees
	for _, fee := range settlementRequest.ContractFees {
		settleTx.AddP2PKHOutput(fee.Address.Bytes(), fee.Quantity, false)
	}

	// Add this contract's fee output
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
	if err != nil {
		return settleTx, errors.Wrap(err, "Failed to retrieve contract")
	}
	if ct.ContractFee > 0 {
		settleTx.AddP2PKHOutput(config.FeePKH.Bytes(), ct.ContractFee, false)

		// Add to settlement request
		settlementRequest.ContractFees = append(settlementRequest.ContractFees, protocol.TargetAddress{Address: *config.FeePKH, Quantity: ct.ContractFee})
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
			outputPKH, err := settleTx.OutputPKH(i)
			if err != nil {
				continue
			}
			if bytes.Equal(transfer.ExchangeFeeAddress.Bytes(), outputPKH) {
				// Add exchange fee to existing output
				settleTx.AddValueToOutput(uint32(i), transfer.ExchangeFee)
				added = true
				break
			}
		}

		if !added {
			// Add new output for exchange fee.
			if err := settleTx.AddP2PKHOutput(transfer.ExchangeFeeAddress.Bytes(), transfer.ExchangeFee, false); err != nil {
				return errors.Wrap(err, "Failed to add exchange fee output")
			}
		}
	}

	return nil
}

// addSettlementData appends data to a pending settlement action.
func addSettlementData(ctx context.Context, masterDB *db.DB, rk *wallet.RootKey,
	transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *wire.MsgTx, settlement *protocol.Settlement, headers BitcoinHeaders) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.addSettlementData")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
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
		if contractOutputAddress != nil && !bytes.Equal(contractOutputAddress.Bytes(), contractPKH.Bytes()) {
			continue // This asset is not ours. Skip it.
		}

		ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}
		if ct.FreezePeriod.Nano() > v.Now.Nano() {
			return rejectError{code: protocol.RejectContractFrozen}
		}

		// Locate Asset
		as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetTransfer.AssetCode)
		if err != nil {
			return fmt.Errorf("Asset ID not found : %s %s : %s", contractPKH, assetTransfer.AssetCode, err)
		}
		if as.FreezePeriod.Nano() > v.Now.Nano() {
			return rejectError{code: protocol.RejectAssetFrozen}
		}
		if !as.TransfersPermitted {
			return rejectError{code: protocol.RejectAssetNotPermitted}
		}

		// Find contract input
		contractInputIndex := uint16(0xffff)
		for i, input := range settleInputAddresses {
			if input != nil && bytes.Equal(input.Bytes(), contractPKH.Bytes()) {
				contractInputIndex = uint16(i)
				break
			}
		}

		if contractInputIndex == uint16(0xffff) {
			return fmt.Errorf("Contract input not found: %s %s", contractPKH, assetTransfer.AssetCode)
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
					contractPKH, assetTransfer.AssetCode, inputAddress)
				return rejectError{code: protocol.RejectHoldingsFrozen}
			}

			if settlementQuantities[settleOutputIndex] == nil {
				// Get sender's balance
				senderBalance := asset.GetBalance(ctx, as, inputAddress)
				settlementQuantities[settleOutputIndex] = &senderBalance
			}

			if *settlementQuantities[settleOutputIndex] < sender.Quantity {
				logger.Warn(ctx, "%s : Insufficient funds: contract=%s asset=%s party=%s", v.TraceID,
					contractPKH, assetTransfer.AssetCode, inputAddress)
				return rejectError{code: protocol.RejectInsufficientQuantity}
			}

			*settlementQuantities[settleOutputIndex] -= sender.Quantity

			// Update total send balance
			sendBalance += sender.Quantity
		}

		// Process receivers
		for receiverOffset, receiver := range assetTransfer.AssetReceivers {
			// Get receiver address from outputs[receiver.Index]
			if int(receiver.Index) >= len(transferTx.Outputs) {
				return fmt.Errorf("Receiver output index out of range for asset %d sender %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			receiverPKH := protocol.PublicKeyHashFromBytes(transferTx.Outputs[receiver.Index].Address.ScriptAddress())

			// Find output in settle tx
			settleOutputIndex := uint16(0xffff)
			for i, outputAddress := range settleOutputAddresses {
				if outputAddress != nil && bytes.Equal(outputAddress.Bytes(), receiverPKH.Bytes()) {
					settleOutputIndex = uint16(i)
					break
				}
			}

			if settleOutputIndex == uint16(0xffff) {
				return fmt.Errorf("Receiver output not found in settle tx for asset %d receiver %d : %d/%d",
					assetOffset, receiverOffset, receiver.Index, len(transferTx.Outputs))
			}

			// Check Oracle Signature
			if err := validateOracle(ctx, contractPKH, ct, &assetTransfer.AssetCode,
				receiverPKH, &receiver, headers); err != nil {
				return rejectError{code: protocol.RejectInvalidSignature, text: err.Error()}
			}

			if settlementQuantities[settleOutputIndex] == nil {
				// Get receiver's balance
				receiverBalance := asset.GetBalance(ctx, as, receiverPKH)
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

// findBoomerangIndex returns the index to the "boomerang" output from transfer tx. It is the
//   output to the contract that is not referenced/spent by the transfers. It is used to fund the
//   offer and signature request messages required between multiple contracts to get a fully
//   approved settlement tx.
func findBoomerangIndex(transferTx *inspector.Transaction, transfer *protocol.Transfer, contractAddress btcutil.Address) uint32 {
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
		if bytes.Equal(output.Address.ScriptAddress(), contractAddress.ScriptAddress()) {
			return uint32(index)
		}
	}

	return 0xffffffff
}

// sendToNextSettlementContract sends settlement data to the next contract involved so it can add its data.
func sendToNextSettlementContract(ctx context.Context, w *node.ResponseWriter, rk *wallet.RootKey,
	itx *inspector.Transaction, transferTx *inspector.Transaction, transfer *protocol.Transfer,
	settleTx *txbuilder.Tx, settlement *protocol.Settlement, settlementRequest *protocol.SettlementRequest,
	tracer *listeners.Tracer) error {
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

	// Serialize settlement tx for Message payload.
	var buf bytes.Buffer
	err = settleTx.MsgTx.Serialize(&buf)
	if err != nil {
		return err
	}

	settlementRequest.Settlement = buf.Bytes()

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
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

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
		err := asset.Update(ctx, dbConn, contractPKH, &assetSettlement.AssetCode, &ua, msg.Timestamp)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateOracle(ctx context.Context, contractPKH *protocol.PublicKeyHash, ct *state.Contract,
	assetCode *protocol.AssetCode, receiverPKH *protocol.PublicKeyHash, tokenReceiver *protocol.TokenReceiver,
	headers BitcoinHeaders) error {

	if tokenReceiver.OracleSigAlgorithm == 0 {
		if len(ct.Oracles) > 0 {
			return fmt.Errorf("Missing signature")
		}
		return nil // No signature required
	}

	// Parse signature
	oracleSig, err := btcec.ParseSignature(tokenReceiver.OracleConfirmationSig, elliptic.P256())
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	expire := (v.Now.Seconds()) - 3600 // Hour ago, unix timestamp in seconds

	// Check all oracles
	for _, oracle := range ct.Oracles {
		oraclePubKey, err := btcec.ParsePubKey(oracle.PublicKey, btcec.S256())
		if err != nil {
			return errors.Wrap(err, "Failed to parse oracle pub key")
		}

		// Check block headers until they are beyond expiration
		previousExpired := 0
		for blockHeight := headers.LastHeight(ctx); ; blockHeight-- {
			hash, err := headers.Hash(ctx, blockHeight)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d", blockHeight))
			}
			sigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, assetCode,
				receiverPKH, tokenReceiver.Quantity, hash)
			if err != nil {
				return errors.Wrap(err, "Failed to calculate oracle sig hash")
			}

			if oracleSig.Verify(sigHash, oraclePubKey) {
				return nil // Valid signature found
			}

			if previousExpired == 4 {
				break
			} else {
				blockTime, err := headers.Time(ctx, blockHeight)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to retrieve time for block height %d", blockHeight))
				}
				if blockTime < expire {
					previousExpired++
				}
			}
		}
	}

	return fmt.Errorf("Signature invalid")
}

// respondTransferReject sends a reject to all parties involved with a transfer request and refunds
//   any bitcoin involved. This can only be done by the first contract, because they hold the
//   bitcoin to be distributed.
func respondTransferReject(ctx context.Context, masterDB *db.DB, config *node.Config, w *node.ResponseWriter,
	transferTx *inspector.Transaction, transfer *protocol.Transfer, rk *wallet.RootKey, code uint8, removeBoomerang bool) error {

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

		if bytes.Equal(transferTx.Outputs[boomerangIndex].Address.ScriptAddress(), rk.Address.ScriptAddress()) {
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
	if !bytes.Equal(transferTx.Outputs[first].Address.ScriptAddress(), rk.Address.ScriptAddress()) {
		return errors.New("This is not the first contract")
	}

	refundBalance := uint64(0)
	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero() {
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
		contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
		ct, err := contract.Retrieve(ctx, masterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve contract")
		}

		// Funding not enough to refund everyone, so don't refund to anyone. Send it to the issuer to hold.
		issuerAddress, err := btcutil.NewAddressPubKeyHash(ct.IssuerPKH.Bytes(), &config.ChainParams)
		w.ClearRejectOutputValues(issuerAddress)
	}

	return node.RespondReject(ctx, w, transferTx, rk, code)
}
