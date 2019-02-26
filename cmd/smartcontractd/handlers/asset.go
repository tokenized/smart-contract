package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB *db.DB
	Config   *node.Config
}

// Definition handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) Definition(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetDefinition)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetDefinition")
	}

	dbConn := a.MasterDB
	defer dbConn.Close()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		return err
	}

	// Contract could not be found
	if ct == nil {
		log.Printf("%s : Contract not found: %+v\n", v.TraceID, contractAddr)
		return node.ErrNoResponse
	}

	// Locate Asset
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// The asset should not exist already
	if as != nil {
		log.Printf("%s : Asset already exists: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeDuplicateAssetID)
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		log.Printf("%s : Number of assets exceeds contract Qty: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeFixedQuantity)
	}

	// Asset Creation <- Asset Definition
	ac := protocol.NewAssetCreation()
	ac.AssetType = msg.AssetType
	ac.AssetID = msg.AssetID
	ac.AssetRevision = 0
	ac.AuthorizationFlags = msg.AuthorizationFlags
	ac.VotingSystem = msg.VotingSystem
	ac.VoteMultiplier = msg.VoteMultiplier
	ac.Qty = msg.Qty
	ac.ContractFeeCurrency = msg.ContractFeeCurrency
	ac.ContractFeeVar = msg.ContractFeeVar
	ac.ContractFeeFixed = msg.ContractFeeFixed
	ac.Payload = msg.Payload

	// Build outputs
	// 1 - Contract Address
	// 2 - Issuer (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: contractAddr,
		Value:   a.Config.DustLimit,
	}, {
		Address: itx.Inputs[0].Address,
		Value:   a.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, log, a.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, log, mux, itx, rk, &ac, outs)
}

// Modification handles an incoming Asset Modification and prepares a Creation response
func (a *Asset) Modification(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetModification)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetModification")
	}

	dbConn := a.MasterDB
	defer dbConn.Close()

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
		log.Printf("%s : Asset ID not found: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		log.Printf("%s : Asset Revision does not match current: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeAssetRevision)
	}

	// @TODO: When reducing an assets available supply, the amount must
	// be deducted from the issuers balance, otherwise the action cannot
	// be performed. i.e: Reduction amount must not be in circulation.

	// @TODO: Likewise when the asset quantity is increased, the amount
	// must be added to the issuers holding balance.

	// Bump the revision
	newRevision := as.Revision + 1

	// Asset Creation <- Asset Modification
	ac := protocol.NewAssetCreation()
	ac.AssetType = msg.AssetType
	ac.AssetID = msg.AssetID
	ac.AssetRevision = newRevision
	ac.AuthorizationFlags = msg.AuthorizationFlags
	ac.VotingSystem = msg.VotingSystem
	ac.VoteMultiplier = msg.VoteMultiplier
	ac.Qty = msg.Qty
	ac.ContractFeeCurrency = msg.ContractFeeCurrency
	ac.ContractFeeVar = msg.ContractFeeVar
	ac.ContractFeeFixed = msg.ContractFeeFixed
	ac.Payload = msg.Payload

	// Build outputs
	// 1 - Contract Address
	// 2 - Issuer (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: contractAddr,
		Value:   a.Config.DustLimit,
	}, {
		Address: itx.Inputs[0].Address,
		Value:   a.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, log, a.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, log, mux, itx, rk, &ac, outs)
}

// Creation handles an outgoing Asset Creation and writes it to the state
func (a *Asset) Creation(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetCreation")
	}

	dbConn := a.MasterDB
	defer dbConn.Close()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Create or update Asset
	if as == nil {
		// Prepare creation object
		na := asset.NewAsset{
			ID:                 string(msg.AssetID),
			Type:               string(msg.AssetType),
			VotingSystem:       string(msg.VotingSystem),
			VoteMultiplier:     msg.VoteMultiplier,
			Qty:                msg.Qty,
			AuthorizationFlags: msg.AuthorizationFlags,
		}
		if err := asset.Create(ctx, dbConn, contractAddr.String(), assetID, &na, v.Now); err != nil {
			return err
		}
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		ua := asset.UpdateAsset{
			Revision:           &msg.AssetRevision,
			Type:               stringPointer(string(msg.AssetType)),
			VotingSystem:       stringPointer(string(msg.VotingSystem)),
			VoteMultiplier:     &msg.VoteMultiplier,
			Qty:                &msg.Qty,
			AuthorizationFlags: msg.AuthorizationFlags,
		}

		if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
			return err
		}
	}

	return nil
}
