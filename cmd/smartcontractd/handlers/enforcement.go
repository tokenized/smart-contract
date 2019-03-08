package handlers

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Enforcement struct {
	MasterDB *db.DB
	Config   *node.Config
}

// OrderRequest handles an incoming Order request and prepares a Confiscation response
func (e *Enforcement) OrderRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Order")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Apply logic based on Compliance Action type
	var err error
	switch msg.ComplianceAction {
	case protocol.ComplianceActionFreeze:
		err = e.OrderFreezeRequest(ctx, mux, itx, rk)
	case protocol.ComplianceActionThaw:
		err = e.OrderThawRequest(ctx, mux, itx, rk)
	case protocol.ComplianceActionConfiscation:
		err = e.OrderConfiscateRequest(ctx, mux, itx, rk)
	default:
		logger.Warn(ctx, "%s : Unknown enforcement: %s", v.TraceID, string(msg.ComplianceAction))
	}

	logger.Info(ctx, "%s : Order request %s", v.TraceID, string(msg.ComplianceAction))
	return err
}

// OrderFreezeRequest is a helper of Order
func (e *Enforcement) OrderFreezeRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderFreezeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Validate target address
	targetAddr, err := btcutil.DecodeAddress(string(msg.TargetAddress), &chaincfg.MainNetParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, msg.TargetAddress)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Holdings check
	_, ok = as.Holdings[targetAddr.String()]
	if !ok {
		logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	logger.Info(ctx, "%s : Freeze order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)

	// Freeze <- Order
	freeze := protocol.NewFreeze()
	freeze.AssetID = msg.AssetID
	freeze.AssetType = msg.AssetType
	freeze.Timestamp = uint64(v.Now.Unix())
	freeze.Qty = msg.Qty
	freeze.Message = msg.Message
	freeze.Expiration = msg.Expiration

	// Build outputs
	// 1 - Target Address
	// 2 - Contract Address (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: targetAddr,
		Value:   e.Config.DustLimit,
	}, {
		Address: contractAddr,
		Value:   e.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, e.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a freeze action
	return node.RespondSuccess(ctx, mux, itx, rk, &freeze, outs)
}

// OrderThawRequest is a helper of Order
func (e *Enforcement) OrderThawRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderThawRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Validate target address
	targetAddr, err := btcutil.DecodeAddress(string(msg.TargetAddress), &chaincfg.MainNetParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, msg.TargetAddress)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Holdings check
	_, ok = as.Holdings[targetAddr.String()]
	if !ok {
		logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	logger.Info(ctx, "%s : Thaw order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)

	// Thaw <- Order
	thaw := protocol.NewThaw()
	thaw.AssetID = msg.AssetID
	thaw.AssetType = msg.AssetType
	thaw.Timestamp = uint64(v.Now.Unix())
	thaw.Qty = msg.Qty
	thaw.Message = msg.Message

	// Build outputs
	// 1 - Target Address
	// 2 - Contract Address (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: targetAddr,
		Value:   e.Config.DustLimit,
	}, {
		Address: contractAddr,
		Value:   e.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, e.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a thaw action
	return node.RespondSuccess(ctx, mux, itx, rk, &thaw, outs)
}

// OrderConfiscateRequest is a helper of Order
func (e *Enforcement) OrderConfiscateRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderConfiscateRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Validate target address
	targetAddr, err := btcutil.DecodeAddress(string(msg.TargetAddress), &chaincfg.MainNetParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, msg.TargetAddress)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Validate deposit address
	depositAddr, err := btcutil.DecodeAddress(string(msg.DepositAddress), &chaincfg.MainNetParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid deposit address: %s %s %s", v.TraceID, contractAddr, assetID, msg.TargetAddress)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Holdings check
	_, ok = as.Holdings[targetAddr.String()]
	if !ok {
		logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	logger.Info(ctx, "%s : Confiscation order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)

	// Find balances
	targetBalance := asset.GetBalance(ctx, as, targetAddr.String())
	depositBalance := asset.GetBalance(ctx, as, depositAddr.String())

	// Transfer the qty from the target to the deposit
	qty := msg.Qty

	// Trying to take more than is held by the target, limit
	// to the amount they are holding.
	if targetBalance < qty {
		qty = targetBalance
	}

	// Modify balances
	targetBalance -= qty
	depositBalance += qty

	// Confiscation <- Order
	confiscation := protocol.NewConfiscation()
	confiscation.AssetID = msg.AssetID
	confiscation.AssetType = msg.AssetType
	confiscation.Timestamp = uint64(v.Now.Unix())
	confiscation.Message = msg.Message
	confiscation.TargetsQty = targetBalance
	confiscation.DepositsQty = depositBalance

	// Build outputs
	// 1 - Target Address
	// 2 - Deposit Address
	// 3 - Contract Address (Change)
	// 4 - Fee
	outs := []node.Output{{
		Address: targetAddr,
		Value:   e.Config.DustLimit,
	}, {
		Address: depositAddr,
		Value:   e.Config.DustLimit,
	}, {
		Address: contractAddr,
		Value:   e.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, e.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a confiscation action
	return node.RespondSuccess(ctx, mux, itx, rk, &confiscation, outs)
}

// FreezeResponse handles an incoming Freeze request and prepares a Confiscation response
func (e *Enforcement) FreezeResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Freeze")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Freeze)
	if !ok {
		return errors.New("Could not assert as *protocol.Freeze")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Common vars
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	party1PKH := itx.Outputs[0].Address.String()

	// Prepare hold
	hold := &state.HoldingStatus{
		Code:    "F",
		Expires: msg.Expiration,
	}

	newStatuses := map[string]*state.HoldingStatus{
		party1PKH: hold,
	}

	// Update asset
	ua := asset.UpdateAsset{
		NewHoldingStatus: newStatuses,
	}

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update freeze : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
		return err
	}

	logger.Info(ctx, "%s : Froze : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
	return nil
}

// ThawResponse handles an incoming Thaw request and prepares a Confiscation response
func (e *Enforcement) ThawResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Thaw")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Thaw)
	if !ok {
		return errors.New("Could not assert as *protocol.Thaw")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Common vars
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	party1PKH := itx.Outputs[0].Address.String()

	// Remove hold
	newStatuses := map[string]*state.HoldingStatus{
		party1PKH: nil,
	}

	// Update asset
	ua := asset.UpdateAsset{
		NewHoldingStatus: newStatuses,
	}

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update thaw : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
		return err
	}

	logger.Info(ctx, "%s : Thawed : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
	return nil
}

// ConfiscationResponse handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) ConfiscationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Confiscation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Confiscation)
	if !ok {
		return errors.New("Could not assert as *protocol.Confiscation")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	// Validate transaction
	if len(itx.Outputs) < 2 {
		logger.Warn(ctx, "%s : Not enough outputs: %s %s", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	// Party 1 (Target), Party 2 (Deposit)
	party1PKH := itx.Outputs[0].Address.String()
	party2PKH := itx.Outputs[1].Address.String()

	newBalances := map[string]uint64{
		party1PKH: msg.TargetsQty,
		party2PKH: msg.DepositsQty,
	}

	// Update asset
	ua := asset.UpdateAsset{
		NewBalances: newBalances,
	}

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update confiscation : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
		return err
	}

	logger.Info(ctx, "%s : Confiscated : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
	return nil
}

// ReconciliationResponse handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) ReconciliationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Reconciliation")
	defer span.End()

	// TODO(srg) - This feature is incomplete

	return nil
}
