package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Enforcement struct {
	MasterDB *db.DB
	Config   *node.Config
}

// OrderRequest handles an incoming Order request and prepares a Confiscation response
func (e *Enforcement) OrderRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Order")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	// Validate all fields have valid values.
	if !itx.DataIsValid {
		node.LogWarn(ctx, "Order request invalid")
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		node.LogWarn(ctx, "Contract address changed : %s", ct.MovedTo.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractMoved)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractExpired)
	}

	senderPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, senderPKH) {
		node.LogWarn(ctx, "Requestor PKH is not administration or operator : %s", contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	// Validate enforcement authority public key and signature
	if msg.AuthorityIncluded {
		if msg.SignatureAlgorithm != 1 {
			node.LogWarn(ctx, "Invalid authority sig algo : %s : %02x", contractPKH.String(), msg.SignatureAlgorithm)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		authorityPubKey, err := btcec.ParsePubKey(msg.AuthorityPublicKey, btcec.S256())
		if err != nil {
			node.LogWarn(ctx, "Failed to parse authority pub key : %s : %s", contractPKH.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		authoritySig, err := btcec.ParseSignature(msg.OrderSignature, btcec.S256())
		if err != nil {
			node.LogWarn(ctx, "Failed to parse authority signature : %s : %s", contractPKH.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		sigHash, err := protocol.OrderAuthoritySigHash(ctx, contractPKH, msg)
		if err != nil {
			return errors.Wrap(err, "Failed to calculate authority sig hash")
		}

		if !authoritySig.Verify(sigHash, authorityPubKey) {
			node.LogWarn(ctx, "Authority Sig Verify Failed : %s", contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInvalidSignature)
		}
	}

	// Apply logic based on Compliance Action type
	switch msg.ComplianceAction {
	case protocol.ComplianceActionFreeze:
		return e.OrderFreezeRequest(ctx, w, itx, rk)
	case protocol.ComplianceActionThaw:
		return e.OrderThawRequest(ctx, w, itx, rk)
	case protocol.ComplianceActionConfiscation:
		return e.OrderConfiscateRequest(ctx, w, itx, rk)
	case protocol.ComplianceActionReconciliation:
		return e.OrderReconciliationRequest(ctx, w, itx, rk)
	default:
		return fmt.Errorf("Unknown compliance action : %s", string(msg.ComplianceAction))
	}
}

// OrderFreezeRequest is a helper of Order
func (e *Enforcement) OrderFreezeRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderFreezeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		node.LogVerbose(ctx, "Requestor is not operator : %s", requestorPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	// Freeze <- Order
	freeze := protocol.Freeze{
		Timestamp: v.Now,
	}

	err = node.Convert(ctx, msg, &freeze)
	if err != nil {
		return errors.Wrap(err, "Failed to convert freeze order to freeze")
	}

	full := false
	if len(msg.TargetAddresses) == 0 {
		node.LogWarn(ctx, "No freeze target addresses specified")
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	} else if len(msg.TargetAddresses) == 1 && bytes.Equal(msg.TargetAddresses[0].Address.Bytes(), contractPKH.Bytes()) {
		full = true
		freeze.Quantities = append(freeze.Quantities, protocol.QuantityIndex{Index: 0, Quantity: 0})
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)
	if msg.AssetCode.IsZero() {
		if !full {
			node.LogWarn(ctx, "Zero asset code in non-full freeze")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	} else {
		as, err := asset.Retrieve(ctx, e.MasterDB, contractPKH, &msg.AssetCode)
		if err != nil {
			node.LogWarn(ctx, "Asset ID not found: %s : %s", msg.AssetCode.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
		}

		if !as.EnforcementOrdersPermitted {
			node.LogWarn(ctx, "Enforcement orders not permitted on asset : %s", msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotPermitted)
		}

		if !full {
			outputIndex := uint16(0)

			// Validate target addresses
			for _, target := range msg.TargetAddresses {
				// Holdings check
				if !asset.CheckHolding(ctx, as, &target.Address) {
					node.LogWarn(ctx, "Holding not found: %s %s", msg.AssetCode.String(), target.Address.String())
					return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
				}

				if target.Quantity == 0 {
					node.LogWarn(ctx, "Zero quantity order is invalid : %s %s", msg.AssetCode.String(), target.Address.String())
					return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
				}

				node.Log(ctx, "Freeze order request : %s %s", msg.AssetCode.String(), target.Address.String())

				targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
				if err != nil {
					return errors.Wrap(err, "Failed to convert target PKH to address")
				}

				// Notify target address
				w.AddOutput(ctx, targetAddr, 0)

				freeze.Quantities = append(freeze.Quantities, protocol.QuantityIndex{Index: outputIndex, Quantity: target.Quantity})
				outputIndex++
			}
		}
	}

	// Add contract output
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &e.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to convert contract PKH to address")
	}
	w.AddOutput(ctx, contractAddress, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a freeze action
	return node.RespondSuccess(ctx, w, itx, rk, &freeze)
}

// OrderThawRequest is a helper of Order
func (e *Enforcement) OrderThawRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderThawRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		node.LogVerbose(ctx, "Requestor is not operator : %s", requestorPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	// Get Freeze Tx
	hash, err := chainhash.NewHash(msg.FreezeTxId.Bytes())
	freezeTx, err := transactions.GetTx(ctx, e.MasterDB, hash, &e.Config.ChainParams, e.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Failed to retrieve freeze tx for thaw : %s : %s", msg.FreezeTxId.String(), err)
	}

	// Get Freeze Op Return
	freeze, ok := freezeTx.MsgProto.(*protocol.Freeze)
	if !ok {
		return fmt.Errorf("Failed to assert freeze tx op return : %s", msg.FreezeTxId.String())
	}

	as, err := asset.Retrieve(ctx, e.MasterDB, contractPKH, &freeze.AssetCode)
	if err != nil {
		node.LogWarn(ctx, "Asset not found: %s", freeze.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
	}

	if !as.EnforcementOrdersPermitted {
		node.LogWarn(ctx, "Enforcement orders not permitted on asset : %s", freeze.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotPermitted)
	}

	// Thaw <- Order
	thaw := protocol.Thaw{
		FreezeTxId: msg.FreezeTxId,
		Timestamp:  v.Now,
	}

	full := false
	if len(freeze.Quantities) == 0 {
		node.LogWarn(ctx, "No freeze target addresses specified : %s", contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	} else if len(freeze.Quantities) == 1 && bytes.Equal(freezeTx.Outputs[freeze.Quantities[0].Index].Address.ScriptAddress(), contractPKH.Bytes()) {
		full = true
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)
	if freeze.AssetCode.IsZero() {
		if !full {
			node.LogWarn(ctx, "Zero asset code in non-full freeze : %s", contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	} else {
		if !full {
			// Validate target addresses
			for _, quantity := range freeze.Quantities {
				node.Log(ctx, "Thaw order request : %s %x", freeze.AssetCode.String(),
					freezeTx.Outputs[quantity.Index].Address.ScriptAddress())

				// Notify target address
				w.AddOutput(ctx, freezeTx.Outputs[quantity.Index].Address, 0)
			}
		}
	}

	// Add contract output
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &e.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to convert contract PKH to address")
	}
	w.AddOutput(ctx, contractAddress, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a thaw action
	return node.RespondSuccess(ctx, w, itx, rk, &thaw)
}

// OrderConfiscateRequest is a helper of Order
func (e *Enforcement) OrderConfiscateRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderConfiscateRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		node.LogVerbose(ctx, "Requestor is not operator : %s", requestorPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	as, err := asset.Retrieve(ctx, e.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil {
		node.LogWarn(ctx, "Asset not found: %s %s", contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
	}

	if !as.EnforcementOrdersPermitted {
		node.LogWarn(ctx, "Enforcement orders not permitted on asset : %s", msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotPermitted)
	}

	// Confiscation <- Order
	confiscation := protocol.Confiscation{}

	err = node.Convert(ctx, msg, &confiscation)
	if err != nil {
		return errors.Wrap(err, "Failed to convert confiscation order to confiscation")
	}

	confiscation.Timestamp = v.Now
	confiscation.Quantities = make([]protocol.QuantityIndex, 0, len(msg.TargetAddresses))

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Deposit Address
	// n+2  - Contract Address
	// n+3  - Contract Fee (change)

	// Validate deposit address, and increase balance by confiscation.DepositQty and increase DepositQty by previous balance
	depositAddr, err := btcutil.NewAddressPubKeyHash(msg.DepositAddress.Bytes(), &e.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to convert deposit PKH to address")
	}

	// Holdings check
	confiscation.DepositQty = asset.GetBalance(ctx, as, &msg.DepositAddress)

	// Validate target addresses
	outputIndex := uint16(0)
	for _, target := range msg.TargetAddresses {
		if target.Quantity == 0 {
			node.LogWarn(ctx, "Zero quantity confiscation order is invalid : %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		balance := asset.GetBalance(ctx, as, &target.Address)
		if target.Quantity > balance {
			node.LogWarn(ctx, "Holding not found: %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
		}

		confiscation.Quantities = append(confiscation.Quantities, protocol.QuantityIndex{Index: outputIndex, Quantity: balance - target.Quantity})
		confiscation.DepositQty += target.Quantity

		node.Log(ctx, "Confiscation order request : %s %s", msg.AssetCode.String(), target.Address.String())

		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
		if err != nil {
			node.LogWarn(ctx, "Invalid target address: %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectUnauthorizedAddress)
		}

		// Notify target address
		w.AddOutput(ctx, targetAddr, 0)
		outputIndex++
	}

	// Notify deposit address
	w.AddOutput(ctx, depositAddr, 0)

	// Add contract output
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &e.Config.ChainParams)
	if err != nil {
		return errors.New("Failed to convert contract pkh to address")
	}
	w.AddOutput(ctx, contractAddress, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a confiscation action
	return node.RespondSuccess(ctx, w, itx, rk, &confiscation)
}

// OrderReconciliationRequest is a helper of Order
func (e *Enforcement) OrderReconciliationRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderReconciliationRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		node.LogVerbose(ctx, "Requestor is not operator : %s", requestorPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	as, err := asset.Retrieve(ctx, e.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil {
		node.LogWarn(ctx, "Asset not found: %s %s", contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
	}

	if !as.EnforcementOrdersPermitted {
		node.LogWarn(ctx, "Enforcement orders not permitted on asset : %s", msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotPermitted)
	}

	// Reconciliation <- Order
	reconciliation := protocol.Reconciliation{}

	err = node.Convert(ctx, msg, &reconciliation)
	if err != nil {
		return errors.Wrap(err, "Failed to convert reconciliation order to reconciliation")
	}

	reconciliation.Timestamp = v.Now
	reconciliation.Quantities = make([]protocol.QuantityIndex, 0, len(msg.TargetAddresses))

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)

	// Validate target addresses
	outputIndex := uint16(0)
	addressOutputIndex := make([]uint16, 0, len(msg.TargetAddresses))
	outputs := make([]node.Output, 0, len(msg.TargetAddresses))
	for _, target := range msg.TargetAddresses {
		if target.Quantity == 0 {
			node.LogWarn(ctx, "Zero quantity reconciliation order is invalid : %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		balance := asset.GetBalance(ctx, as, &target.Address)
		if target.Quantity > balance {
			node.LogWarn(ctx, "Holding not found: %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
		}

		reconciliation.Quantities = append(reconciliation.Quantities, protocol.QuantityIndex{Index: outputIndex, Quantity: balance - target.Quantity})

		node.Log(ctx, "Reconciliation order request : %s %s", msg.AssetCode.String(), target.Address.String())

		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
		if err != nil {
			node.LogWarn(ctx, "Invalid target address : %s %s", msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectUnauthorizedAddress)
		}

		// Notify target address
		outputs = append(outputs, node.Output{Address: targetAddr, Value: 0})
		addressOutputIndex = append(addressOutputIndex, outputIndex)
		outputIndex++
	}

	// Update outputs with bitcoin dispersions
	for _, quantity := range msg.BitcoinDispersions {
		if int(quantity.Index) >= len(msg.TargetAddresses) {
			node.LogWarn(ctx, "Invalid bitcoin dispersion index : %s %d", msg.AssetCode.String(), quantity.Index)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		outputs[addressOutputIndex[quantity.Index]].Value += quantity.Quantity
	}

	// Add outputs to response writer
	for _, output := range outputs {
		w.AddOutput(ctx, output.Address, output.Value)
	}

	// Add contract output
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &e.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to convert contract address")
	}
	w.AddOutput(ctx, contractAddress, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a reconciliation action
	return node.RespondSuccess(ctx, w, itx, rk, &reconciliation)
}

// FreezeResponse handles an outgoing Freeze action and writes it to the state
func (e *Enforcement) FreezeResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Freeze")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Freeze)
	if !ok {
		return errors.New("Could not assert as *protocol.Freeze")
	}

	if !itx.DataIsValid {
		return errors.New("Freeze response invalid")
	}

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Freeze not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	full := false
	if len(msg.Quantities) == 0 {
		return fmt.Errorf("No freeze addresses specified : %s", contractPKH.String())
	} else if len(msg.Quantities) == 1 && bytes.Equal(itx.Outputs[msg.Quantities[0].Index].Address.ScriptAddress(), contractPKH.Bytes()) {
		full = true
	}

	if msg.AssetCode.IsZero() {
		if !full {
			return fmt.Errorf("Zero asset code in non-full freeze : %s", contractPKH.String())
		} else {
			// Contract wide freeze
			uc := contract.UpdateContract{FreezePeriod: &msg.FreezePeriod}
			if err := contract.Update(ctx, e.MasterDB, contractPKH, &uc, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to update contract freeze period")
			}
		}
	} else {
		if full {
			// Asset wide freeze
			ua := asset.UpdateAsset{FreezePeriod: &msg.FreezePeriod}
			if err := asset.Update(ctx, e.MasterDB, contractPKH, &msg.AssetCode, &ua, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to update asset freeze period")
			}
		} else {
			ua := asset.UpdateAsset{NewHoldingStatuses: make(map[protocol.PublicKeyHash]state.HoldingStatus)}

			// Validate target addresses
			for _, quantity := range msg.Quantities {
				if int(quantity.Index) >= len(itx.Outputs) {
					return fmt.Errorf("Freeze quantity index out of range : %d/%d", quantity.Index, len(itx.Outputs))
				}

				userPKH := protocol.PublicKeyHashFromBytes(itx.Outputs[quantity.Index].Address.ScriptAddress())
				ua.NewHoldingStatuses[*userPKH] = state.HoldingStatus{
					Code:    byte('F'), // Freeze
					Expires: msg.FreezePeriod,
					Balance: quantity.Quantity,
					TxId:    *protocol.TxIdFromBytes(itx.Hash[:]),
				}
			}

			if err := asset.Update(ctx, e.MasterDB, contractPKH, &msg.AssetCode, &ua, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to update asset holding freezes")
			}
		}
	}

	// Save Tx for thaw action.
	if err := transactions.AddTx(ctx, e.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	node.Log(ctx, "Processed Freeze : %s", msg.AssetCode.String())
	return nil
}

// ThawResponse handles an outgoing Thaw action and writes it to the state
func (e *Enforcement) ThawResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Thaw")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Thaw)
	if !ok {
		return errors.New("Could not assert as *protocol.Thaw")
	}

	if !itx.DataIsValid {
		return errors.New("Thaw response invalid")
	}

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Thaw not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	// Get Freeze Tx
	hash, _ := chainhash.NewHash(msg.FreezeTxId.Bytes())
	freezeTx, err := transactions.GetTx(ctx, e.MasterDB, hash, &e.Config.ChainParams, e.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Failed to retrieve freeze tx for thaw : %s : %s", msg.FreezeTxId.String(), err)
	}

	// Get Freeze Op Return
	freeze, ok := freezeTx.MsgProto.(*protocol.Freeze)
	if !ok {
		return fmt.Errorf("Failed to assert freeze tx op return : %s", msg.FreezeTxId.String())
	}

	full := false
	if len(freeze.Quantities) == 0 {
		return fmt.Errorf("No freeze addresses specified : %s", contractPKH.String())
	} else if len(freeze.Quantities) == 1 && bytes.Equal(freezeTx.Outputs[freeze.Quantities[0].Index].Address.ScriptAddress(), contractPKH.Bytes()) {
		full = true
	}

	if freeze.AssetCode.IsZero() {
		if !full {
			return fmt.Errorf("Zero asset code in non-full freeze : %s", contractPKH.String())
		} else {
			// Contract wide freeze
			var zeroTimestamp protocol.Timestamp
			uc := contract.UpdateContract{FreezePeriod: &zeroTimestamp}
			if err := contract.Update(ctx, e.MasterDB, contractPKH, &uc, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to clear contract freeze period")
			}
		}
	} else {
		if full {
			// Asset wide freeze
			var zeroTimestamp protocol.Timestamp
			ua := asset.UpdateAsset{FreezePeriod: &zeroTimestamp}
			if err := asset.Update(ctx, e.MasterDB, contractPKH, &freeze.AssetCode, &ua, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to clear asset freeze period")
			}
		} else {
			ua := asset.UpdateAsset{ClearHoldingStatuses: make(map[protocol.PublicKeyHash]protocol.TxId)}
			freezeTxId := protocol.TxIdFromBytes(freezeTx.Hash[:])

			// Validate target addresses
			for _, quantity := range freeze.Quantities {
				if int(quantity.Index) >= len(freezeTx.Outputs) {
					return fmt.Errorf("Freeze quantity index out of range : %d/%d", quantity.Index, len(freezeTx.Outputs))
				}

				userPKH := protocol.PublicKeyHashFromBytes(freezeTx.Outputs[quantity.Index].Address.ScriptAddress())
				ua.ClearHoldingStatuses[*userPKH] = *freezeTxId
			}

			if err := asset.Update(ctx, e.MasterDB, contractPKH, &freeze.AssetCode, &ua, msg.Timestamp); err != nil {
				return errors.Wrap(err, "Failed to clear asset holding freezes")
			}
		}
	}

	node.Log(ctx, "Processed Thaw : %s", freeze.AssetCode.String())
	return nil
}

// ConfiscationResponse handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) ConfiscationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Confiscation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Confiscation)
	if !ok {
		return errors.New("Could not assert as *protocol.Confiscation")
	}

	if !itx.DataIsValid {
		return errors.New("Confiscation response invalid")
	}

	// Locate Asset
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Confiscation not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	// Apply confiscations
	ua := asset.UpdateAsset{NewBalances: make(map[protocol.PublicKeyHash]uint64)}

	highestIndex := uint16(0)
	for _, quantity := range msg.Quantities {
		userPKH := protocol.PublicKeyHashFromBytes(itx.Outputs[quantity.Index].Address.ScriptAddress())
		ua.NewBalances[*userPKH] = quantity.Quantity
		if quantity.Index > highestIndex {
			highestIndex = quantity.Index
		}
	}

	// Update deposit balance
	depositPKH := protocol.PublicKeyHashFromBytes(itx.Outputs[highestIndex+1].Address.ScriptAddress())
	ua.NewBalances[*depositPKH] = msg.DepositQty

	if err := asset.Update(ctx, e.MasterDB, contractPKH, &msg.AssetCode, &ua, msg.Timestamp); err != nil {
		return errors.Wrap(err, "Failed to udpate asset holdings for confiscation")
	}

	node.Log(ctx, "Processed Confiscation : %s", msg.AssetCode.String())
	return nil
}

// ReconciliationResponse handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) ReconciliationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Reconciliation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Reconciliation)
	if !ok {
		return errors.New("Could not assert as *protocol.Reconciliation")
	}

	if !itx.DataIsValid {
		return errors.New("Reconciliation response invalid")
	}

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Reconciliation not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	// Apply reconciliations
	ua := asset.UpdateAsset{NewBalances: make(map[protocol.PublicKeyHash]uint64)}

	highestIndex := uint16(0)
	for _, quantity := range msg.Quantities {
		userPKH := protocol.PublicKeyHashFromBytes(itx.Outputs[quantity.Index].Address.ScriptAddress())
		ua.NewBalances[*userPKH] = quantity.Quantity
		if quantity.Index > highestIndex {
			highestIndex = quantity.Index
		}
	}

	if err := asset.Update(ctx, e.MasterDB, contractPKH, &msg.AssetCode, &ua, msg.Timestamp); err != nil {
		return errors.Wrap(err, "Failed to udpate asset holdings for confiscation")
	}

	node.Log(ctx, "Processed Confiscation : %s", msg.AssetCode.String())
	return nil
}
