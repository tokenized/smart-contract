package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/btcsuite/btcutil"
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

	v := ctx.Value(node.KeyValues).(*node.Values)

	// TODO Validate enforcement authority public key and signature

	// Apply logic based on Compliance Action type
	var err error
	switch msg.ComplianceAction {
	case protocol.ComplianceActionFreeze:
		err = e.OrderFreezeRequest(ctx, w, itx, rk)
	case protocol.ComplianceActionThaw:
		err = e.OrderThawRequest(ctx, w, itx, rk)
	case protocol.ComplianceActionConfiscation:
		err = e.OrderConfiscateRequest(ctx, w, itx, rk)
	default:
		logger.Warn(ctx, "%s : Unknown enforcement: %s", v.TraceID, string(msg.ComplianceAction))
	}

	logger.Info(ctx, "%s : Order request %s", v.TraceID, string(msg.ComplianceAction))
	return err
}

// OrderFreezeRequest is a helper of Order
func (e *Enforcement) OrderFreezeRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderFreezeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Freeze <- Order
	freeze := protocol.Freeze{
		Timestamp: v.Now,
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address (Change)
	// n+2  - Fee

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		// Holdings check
		_, ok := as.Holdings[target.Address]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		logger.Info(ctx, "%s : Freeze order request : %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
		freeze.Addresses = append(freeze.Addresses, target.Address)

		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Notify target address
		w.AddOutput(ctx, targetAddr, 0)
	}

	// Change from/back to contract
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractAddr.Bytes(), &e.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid contract address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	}
	w.AddChangeOutput(ctx, contractAddress)

	// Add fee output
	w.AddFee(ctx)

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

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Thaw <- Order
	thaw := protocol.Thaw{
		Timestamp: v.Now,
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address (Change)
	// n+2  - Fee

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		// Holdings check
		_, ok = as.Holdings[target.Address]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		logger.Info(ctx, "%s : Thaw order request : %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
		thaw.Addresses = append(thaw.Addresses, target.Address)

		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Notify target address
		w.AddOutput(ctx, targetAddr, 0)
	}

	// Change from/back to contract
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractAddr.Bytes(), &e.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid contract address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	}
	w.AddChangeOutput(ctx, contractAddress)

	// Add fee output
	w.AddFee(ctx)

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

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Confiscation <- Order
	confiscation := protocol.Confiscation{
		Timestamp:  v.Now,
		DepositQty: 0,
	}

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Deposit Address
	// n+2  - Contract Address (Change)
	// n+3  - Fee

	// Validate deposit address, and increase balance by confiscation.DepositQty and increase DepositQty by previous balance
	depositAddr, err := btcutil.NewAddressPubKeyHash(msg.DepositAddress.Bytes(), &e.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid deposit address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), msg.DepositAddress.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Holdings check
	depositHolding, depositOK := as.Holdings[msg.DepositAddress]
	if depositOK {
		confiscation.DepositQty = depositHolding.Balance
	} else {
		confiscation.DepositQty = 0 // No intial balance in "custodian"
	}

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		// Holdings check
		holding, ok := as.Holdings[target.Address]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		confiscation.DepositQty += holding.Balance

		logger.Info(ctx, "%s : Confiscation order request : %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
		confiscation.Addresses = append(confiscation.Addresses, target.Address)

		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address.Bytes(), &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String(), target.Address.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Notify target address
		w.AddOutput(ctx, targetAddr, 0)
	}

	// Notify deposit address
	w.AddOutput(ctx, depositAddr, 0)

	// Change from/back to contract
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractAddr.Bytes(), &e.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid contract address: %s %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	}
	w.AddChangeOutput(ctx, contractAddress)

	// Add fee output
	w.AddFee(ctx)

	// Respond with a confiscation action
	return node.RespondSuccess(ctx, w, itx, rk, &confiscation)
}

// FreezeResponse handles an outgoing Freeze action and writes it to the state
func (e *Enforcement) FreezeResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Freeze")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	_, ok := itx.MsgProto.(*protocol.Freeze)
	if !ok {
		return errors.New("Could not assert as *protocol.Freeze")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for freeze : %s", v.TraceID, err)
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert freeze input as *protocol.Order")
	}

	// dbConn := e.MasterDB

	// Common vars
	contractAddr := rk.Address

	logger.Warn(ctx, "%s : Freeze response not implemented : %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
	return errors.New("Freeze response not implemented")

	// TODO Implement after format is updated
	// Update asset
	// ua := asset.UpdateAsset{NewHoldingStatuses: make(map[string]*state.HoldingStatus)}

	// for _, address := range freeze.Addresses {
	// status := state.HoldingStatus{
	// Code:    "F", // Freeze
	// Expires: order.FreezePeriod,
	// Balance: address.Quantity,
	// TxId:    itx.Hash,
	// }
	// ua.NewHoldingStatuses[address.Address] = &status
	// }

	// if err := asset.Update(ctx, dbConn, contractAddr.String(), order.AssetCode, &ua, v.Now); err != nil {
	// logger.Warn(ctx, "%s : Failed to update for freeze : %s %s %s", v.TraceID, contractAddr, order.AssetCode, party1PKH)
	// return err
	// }

	// logger.Info(ctx, "%s : Processed Freeze : %s %s %s", v.TraceID, contractAddr, order.AssetCode, party1PKH)
	// return nil
}

// ThawResponse handles an outgoing Thaw action and writes it to the state
func (e *Enforcement) ThawResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Thaw")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	_, ok := itx.MsgProto.(*protocol.Thaw)
	if !ok {
		return errors.New("Could not assert as *protocol.Thaw")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for thaw : %s", v.TraceID, err)
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert thaw input as *protocol.Order")
	}

	dbConn := e.MasterDB

	// Common vars
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	// Get reference tx hash
	// refTxID, err := chainhash.NewHash(thaw.RefTxnID)
	// if err != nil {
	// logger.Warn(ctx, "%s : Failed read ref tx id for thaw : %s %s : %s", v.TraceID, contractAddr, order.AssetCode, err.Error)
	// return err
	// }

	// Remove freezes
	ua := asset.UpdateAsset{NewHoldingStatuses: make(map[protocol.PublicKeyHash]*state.HoldingStatus)}

	// TODO Implement after format is updated
	// for _, address := range thaw.Addresses {
	// ua.ClearHoldingStatuses[address.Address] = refTxID
	// }

	if err := asset.Update(ctx, dbConn, contractAddr, &order.AssetCode, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update for thaw : %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
		return err
	}

	logger.Info(ctx, "%s : Processed Thaw : %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
	return nil
}

// ConfiscationResponse handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) ConfiscationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Confiscation")
	defer span.End()

	v := ctx.Value(node.KeyValues).(*node.Values)

	_, ok := itx.MsgProto.(*protocol.Confiscation)
	if !ok {
		return errors.New("Could not assert as *protocol.Confiscation")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for thaw : %s", v.TraceID, err)
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert confiscate input as *protocol.Order")
	}

	dbConn := e.MasterDB

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &order.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
		return node.ErrNoResponse
	}

	// Apply confiscations
	ua := asset.UpdateAsset{NewBalances: make(map[protocol.PublicKeyHash]uint64)}

	// TODO Finish implementing ConfiscationResponse
	// TODO Implement after format is updated
	// for _, address := range confiscation.Addresses {
	// ua.NewBalances[address.Address] -=
	// }

	// Validate transaction
	if len(itx.Outputs) < 2 {
		logger.Warn(ctx, "%s : Not enough outputs: %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
		return node.ErrNoResponse
	}

	if err := asset.Update(ctx, dbConn, contractAddr, &order.AssetCode, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update confiscation : %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
		return err
	}

	logger.Info(ctx, "%s : Processed Confiscation : %s %s", v.TraceID, contractAddr.String(), order.AssetCode.String())
	return nil
}

// ReconciliationResponse handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) ReconciliationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Reconciliation")
	defer span.End()

	// TODO(srg) - This feature is incomplete

	return nil
}
