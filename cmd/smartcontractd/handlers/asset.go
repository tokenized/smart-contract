package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB *db.DB
	Config   *node.Config
}

// DefinitionRequest handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) DefinitionRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetDefinition)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetDefinition")
	}

	dbConn := a.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve contract : %s\n", v.TraceID, contractAddr.String())
		return err
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s\n", v.TraceID, contractAddr.String())
		return node.ErrNoResponse
	}

	// Verify issuer is sender of tx.
	if itx.Inputs[0].Address.String() != ct.IssuerAddress {
		logger.Warn(ctx, "%s : Only issuer can create assets: %s %s\n", v.TraceID, contractAddr.String(), string(msg.AssetID))
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeIssuerAddress)
	}

	// Locate Asset
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// The asset should not exist already
	if as != nil {
		logger.Warn(ctx, "%s : Asset already exists: %s %s\n", v.TraceID, contractAddr.String(), assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeDuplicateAssetID)
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		logger.Verbose(ctx, "%s : Number of assets exceeds contract Qty: %s %s\n", v.TraceID, contractAddr.String(), assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFixedQuantity)
	}

	logger.Verbose(ctx, "%s : Accepting asset creation request : %s %s\n", v.TraceID, contractAddr.String(), assetID)

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
	if fee := node.OutputFee(ctx, a.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, mux, itx, rk, &ac, outs)
}

// ModificationRequest handles an incoming Asset Modification and prepares a Creation response
func (a *Asset) ModificationRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetModification)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetModification")
	}

	dbConn := a.MasterDB

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
		logger.Verbose(ctx, "%s : Asset ID not found: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		logger.Verbose(ctx, "%s : Asset Revision does not match current: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetRevision)
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
	if fee := node.OutputFee(ctx, a.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, mux, itx, rk, &ac, outs)
}

// CreationResponse handles an outgoing Asset Creation and writes it to the state
func (a *Asset) CreationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetCreation")
	}

	dbConn := a.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve asset : %s %s\n", v.TraceID, contractAddr.String(), assetID)
		return err
	}

	// Create or update Asset
	if as == nil {
		// Prepare creation object
		na := asset.NewAsset{
			IssuerAddress:      itx.Outputs[1].Address.String(), // Second output of formation tx
			ID:                 string(msg.AssetID),
			Type:               string(msg.AssetType),
			VotingSystem:       string(msg.VotingSystem),
			VoteMultiplier:     msg.VoteMultiplier,
			Qty:                msg.Qty,
			AuthorizationFlags: msg.AuthorizationFlags,
		}
		if err := asset.Create(ctx, dbConn, contractAddr.String(), assetID, &na, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to create asset : %s %s\n", v.TraceID, contractAddr.String(), assetID)
			return err
		}
		logger.Verbose(ctx, "%s : Created asset : %s %s\n", v.TraceID, contractAddr.String(), assetID)
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
			logger.Warn(ctx, "%s : Failed to update asset : %s %s\n", v.TraceID, contractAddr.String(), assetID)
			return err
		}
		logger.Verbose(ctx, "%s : Updated asset : %s %s\n", v.TraceID, contractAddr.String(), assetID)
	}

	return nil
}
