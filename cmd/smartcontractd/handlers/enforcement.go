package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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

	// TODO Validate enforcement authority public key and signature

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

	// Freeze <- Order
	freeze := protocol.NewFreeze()
	freeze.Timestamp = uint64(time.Now().UnixNano())

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address (Change)
	// n+2  - Fee
	outs := make([]node.Output, 0, len(msg.TargetAddresses)+2)

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address, &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, target.Address)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Holdings check
		_, ok := as.Holdings[targetAddr.String()]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		logger.Info(ctx, "%s : Freeze order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)
		address := protocol.NewAddress()
		address.Address = targetAddr.ScriptAddress()
		freeze.Addresses = append(freeze.Addresses, address)

		// Notify target address
		outs = append(outs, node.Output{Address: targetAddr, Value: e.Config.DustLimit})
	}

	freeze.AddressCount = uint16(len(freeze.Addresses))

	// Change from/back to contract
	outs = append(outs, node.Output{Address: contractAddr, Value: e.Config.DustLimit, Change: true})

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

	// Thaw <- Order
	thaw := protocol.NewThaw()
	thaw.Timestamp = uint64(time.Now().UnixNano())

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address (Change)
	// n+2  - Fee
	outs := make([]node.Output, 0, len(msg.TargetAddresses)+2)

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address, &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, target.Address)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Holdings check
		_, ok = as.Holdings[targetAddr.String()]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		logger.Info(ctx, "%s : Thaw order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)
		address := protocol.NewAddress()
		address.Address = targetAddr.ScriptAddress()
		thaw.Addresses = append(thaw.Addresses, address)

		// Notify target address
		outs = append(outs, node.Output{Address: targetAddr, Value: e.Config.DustLimit})
	}

	thaw.AddressCount = uint16(len(thaw.Addresses))

	// Change from/back to contract
	outs = append(outs, node.Output{Address: contractAddr, Value: e.Config.DustLimit, Change: true})

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

	// Confiscation <- Order
	confiscation := protocol.NewConfiscation()
	confiscation.Timestamp = uint64(time.Now().UnixNano())
	confiscation.DepositQty = 0

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Deposit Address
	// n+2  - Contract Address (Change)
	// n+3  - Fee
	outs := make([]node.Output, 0, len(msg.TargetAddresses)+3)

	// Validate deposit address, and increase balance by confiscation.DepositQty and increase DepositQty by previous balance
	depositAddr, err := btcutil.NewAddressPubKeyHash(msg.DepositAddress, &e.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Invalid deposit address: %s %s %s", v.TraceID, contractAddr, assetID, depositAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Holdings check
	depositHolding, depositOK := as.Holdings[depositAddr.String()]
	if depositOK {
		confiscation.DepositQty = depositHolding.Balance
	} else {
		confiscation.DepositQty = 0 // No intial balance in "custodian"
	}

	// Validate target addresses
	for _, target := range msg.TargetAddresses {
		targetAddr, err := btcutil.NewAddressPubKeyHash(target.Address, &e.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Invalid target address: %s %s %s", v.TraceID, contractAddr, assetID, target.Address)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeUnknownAddress)
		}

		// Holdings check
		holding, ok := as.Holdings[targetAddr.String()]
		if !ok {
			logger.Warn(ctx, "%s : Holding not found: contract=%s asset=%s party=%s", v.TraceID, contractAddr, assetID, targetAddr)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		confiscation.DepositQty += holding.Balance

		logger.Info(ctx, "%s : Confiscation order request : %s %s %s", v.TraceID, contractAddr, assetID, targetAddr)
		address := protocol.NewAddress()
		address.Address = targetAddr.ScriptAddress()
		confiscation.Addresses = append(confiscation.Addresses, address)

		// Notify target address
		outs = append(outs, node.Output{Address: targetAddr, Value: e.Config.DustLimit})
	}

	confiscation.AddressCount = uint16(len(confiscation.Addresses))

	// Notify deposit address
	outs = append(outs, node.Output{Address: depositAddr, Value: e.Config.DustLimit})

	// Change from/back to contract
	outs = append(outs, node.Output{Address: contractAddr, Value: e.Config.DustLimit, Change: true})

	// Add fee output
	if fee := node.OutputFee(ctx, e.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a confiscation action
	return node.RespondSuccess(ctx, mux, itx, rk, &confiscation, outs)
}

// FreezeResponse handles an outgoing Freeze action and writes it to the state
func (e *Enforcement) FreezeResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Freeze")
	defer span.End()

	freeze, ok := itx.MsgProto.(*protocol.Freeze)
	if !ok {
		return errors.New("Could not assert as *protocol.Freeze")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for freeze : %s", err.Error())
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert freeze input as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Common vars
	contractAddr := rk.Address
	assetID := string(order.AssetID)

	logger.Warn(ctx, "%s : Freeze response not implemented : %s %s", v.TraceID, contractAddr, assetID)

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

	// if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
	// logger.Warn(ctx, "%s : Failed to update for freeze : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
	// return err
	// }

	// logger.Info(ctx, "%s : Processed Freeze : %s %s %s", v.TraceID, contractAddr, assetID, party1PKH)
	return nil
}

// ThawResponse handles an outgoing Thaw action and writes it to the state
func (e *Enforcement) ThawResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Thaw")
	defer span.End()

	thaw, ok := itx.MsgProto.(*protocol.Thaw)
	if !ok {
		return errors.New("Could not assert as *protocol.Thaw")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for thaw : %s", err.Error())
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert thaw input as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Common vars
	contractAddr := rk.Address
	assetID := string(order.AssetID)

	// Get reference tx hash
	refTxID, err := chainhash.NewHash(thaw.RefTxnID)
	if err != nil {
		logger.Warn(ctx, "%s : Failed read ref tx id for thaw : %s %s : %s", v.TraceID, contractAddr, assetID, err.Error)
		return err
	}

	// Remove freezes
	ua := asset.UpdateAsset{NewHoldingStatuses: make(map[string]*state.HoldingStatus)}

	// TODO Implement after format is updated
	// for _, address := range thaw.Addresses {
	// ua.ClearHoldingStatuses[address.Address] = refTxID
	// }

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update for thaw : %s %s", v.TraceID, contractAddr, assetID)
		return err
	}

	logger.Info(ctx, "%s : Processed Thaw : %s %s", v.TraceID, contractAddr, assetID)
	return nil
}

// ConfiscationResponse handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) ConfiscationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Confiscation")
	defer span.End()

	confiscation, ok := itx.MsgProto.(*protocol.Confiscation)
	if !ok {
		return errors.New("Could not assert as *protocol.Confiscation")
	}

	// Get order that triggered freeze
	orderItx, err := inspector.NewTransactionFromWire(ctx, itx.Inputs[0].FullTx)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to get order for thaw : %s", err.Error())
		return err
	}

	order, ok := orderItx.MsgProto.(*protocol.Order)
	if !ok {
		return errors.New("Could not assert confiscate input as *protocol.Order")
	}

	dbConn := e.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(order.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	// Apply confiscations
	ua := asset.UpdateAsset{NewBalances: make(map[string]uint64)}

	// TODO Finish implementing ConfiscationResponse
	// TODO Implement after format is updated
	// for _, address := range confiscation.Addresses {
	// ua.NewBalances[address.Address] -=
	// }

	// Validate transaction
	if len(itx.Outputs) < 2 {
		logger.Warn(ctx, "%s : Not enough outputs: %s %s", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		logger.Warn(ctx, "%s : Failed to update confiscation : %s %s", v.TraceID, contractAddr, assetID)
		return err
	}

	logger.Info(ctx, "%s : Processed Confiscation : %s %s", v.TraceID, contractAddr, assetID)
	return nil
}

// ReconciliationResponse handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) ReconciliationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Reconciliation")
	defer span.End()

	// TODO(srg) - This feature is incomplete

	return nil
}
