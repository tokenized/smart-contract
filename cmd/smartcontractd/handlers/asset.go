package handlers

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform"
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

	// TODO Check action funding here
	// Variable depending on the size of the payload.
	// Fee rate * (response payload size + size of response inputs(average P2PKH input) + size of response outputs(average P2PKH output))

	dbConn := a.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve contract : %s", v.TraceID, contractAddr)
		return err
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr)
		return node.ErrNoResponse
	}

	// Verify issuer is sender of tx.
	if itx.Inputs[0].Address.String() != ct.Issuer {
		logger.Warn(ctx, "%s : Only issuer can create assets: %s %s", v.TraceID, contractAddr, string(msg.AssetID))
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
		logger.Warn(ctx, "%s : Asset already exists: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeDuplicateAssetID)
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		logger.Verbose(ctx, "%s : Number of assets exceeds contract Qty: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFixedQuantity)
	}

	logger.Info(ctx, "%s : Accepting asset creation request : %s %s", v.TraceID, contractAddr, assetID)

	// Asset Creation <- Asset Definition
	ac := protocol.AssetCreation{}

	err = platform.Convert(msg, ac)
	if err != nil {
		return err
	}

	ac.Timestamp = uint64(time.Now().UnixNano())

	// Build outputs
	// 1 - Contract Address
	// 2 - Issuer (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: contractAddr,
		Value:   a.Config.DustLimit,
	}, {
		Address: itx.Inputs[0].Address, // Request must come from issuer
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
		logger.Verbose(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		logger.Verbose(ctx, "%s : Asset Revision does not match current: %s %s", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetRevision)
	}

	// @TODO: When reducing an assets available supply, the amount must
	// be deducted from the issuers balance, otherwise the action cannot
	// be performed. i.e: Reduction amount must not be in circulation.

	// @TODO: Likewise when the asset quantity is increased, the amount
	// must be added to the issuers holding balance.

	// Asset Creation <- Asset Modification
	ac := protocol.AssetCreation{}

	err = platform.Convert(as, ac)
	if err != nil {
		return err
	}

	ac.AssetRevision = as.Revision + 1
	ac.Timestamp = uint64(time.Now().UnixNano())

	// TODO Implement asset amendments
	// type Amendment struct {
	// FieldIndex    uint8
	// Element       uint16
	// SubfieldIndex uint8
	// Operation     uint8
	// Data          []byte
	// }
	// for _, amendment := range msg.Amendments {
	// switch(amendment.FieldIndex) {
	// default:
	// logger.Warn(ctx, "%s : Incorrect asset amendment field offset (%s) : %d", v.TraceID, assetID, amendment.FieldIndex)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetMalformedAmendment)
	// }
	// }

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
		logger.Warn(ctx, "%s : Failed to retrieve asset : %s %s", v.TraceID, contractAddr, assetID)
		return err
	}

	// Create or update Asset
	if as == nil {
		// Prepare creation object
		na := asset.NewAsset{}

		err = platform.Convert(msg, na)
		if err != nil {
			return err
		}

		if err := asset.Create(ctx, dbConn, contractAddr.String(), assetID, &na, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to create asset : %s %s", v.TraceID, contractAddr, assetID)
			return err
		}
		logger.Info(ctx, "%s : Created asset : %s %s", v.TraceID, contractAddr, assetID)
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		ua := asset.UpdateAsset{
			Revision:  &msg.AssetRevision,
			Timestamp: &msg.Timestamp,
		}

		if as.AssetType != string(msg.AssetType) {
			ua.AssetType = stringPointer(string(msg.AssetType))
			logger.Info(ctx, "%s : Updating asset type (%s) : %s", v.TraceID, assetID, *ua.AssetType)
		}
		if !bytes.Equal(as.AssetAuthFlags[:], msg.AssetAuthFlags[:]) {
			ua.AssetAuthFlags = &msg.AssetAuthFlags
			logger.Info(ctx, "%s : Updating asset auth flags (%s) : %s", v.TraceID, assetID, *ua.AssetAuthFlags)
		}
		if as.TransfersPermitted != msg.TransfersPermitted {
			ua.TransfersPermitted = &msg.TransfersPermitted
			logger.Info(ctx, "%s : Updating asset transfers permitted (%s) : %t", v.TraceID, assetID, *ua.TransfersPermitted)
		}
		if as.TradeRestrictions != msg.TradeRestrictions {
			ua.TradeRestrictions = &msg.TradeRestrictions
			logger.Info(ctx, "%s : Updating asset trade restrictions (%s) : %s", v.TraceID, assetID, *ua.TradeRestrictions)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			logger.Info(ctx, "%s : Updating asset enforcement orders permitted (%s) : %t", v.TraceID, assetID, *ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			logger.Info(ctx, "%s : Updating asset vote multiplier (%s) : %02x", v.TraceID, assetID, *ua.VoteMultiplier)
		}
		if as.ReferendumProposal != msg.ReferendumProposal {
			ua.ReferendumProposal = &msg.ReferendumProposal
			logger.Info(ctx, "%s : Updating asset referendum proposal (%s) : %t", v.TraceID, assetID, *ua.ReferendumProposal)
		}
		if as.InitiativeProposal != msg.InitiativeProposal {
			ua.InitiativeProposal = &msg.InitiativeProposal
			logger.Info(ctx, "%s : Updating asset initiative proposal (%s) : %t", v.TraceID, assetID, *ua.InitiativeProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			logger.Info(ctx, "%s : Updating asset modification governance (%s) : %t", v.TraceID, assetID, *ua.AssetModificationGovernance)
		}
		if as.TokenQty != msg.TokenQty {
			ua.TokenQty = &msg.TokenQty
			logger.Info(ctx, "%s : Updating asset token quantity (%s) : %d", v.TraceID, assetID, *ua.TokenQty)
		}
		if as.ContractFeeCurrency != msg.ContractFeeCurrency {
			ua.ContractFeeCurrency = &msg.ContractFeeCurrency
			logger.Info(ctx, "%s : Updating asset contract fee currency (%s) : %s", v.TraceID, assetID, *ua.ContractFeeCurrency)
		}
		if as.ContractFeeVar != msg.ContractFeeVar {
			ua.ContractFeeVar = &msg.ContractFeeVar
			logger.Info(ctx, "%s : Updating asset contract fee variable (%s) : %f", v.TraceID, assetID, *ua.ContractFeeVar)
		}
		if as.ContractFeeFixed != msg.ContractFeeFixed {
			ua.ContractFeeFixed = &msg.ContractFeeFixed
			logger.Info(ctx, "%s : Updating asset contract fee fixed (%s) : %f", v.TraceID, assetID, *ua.ContractFeeFixed)
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			logger.Info(ctx, "%s : Updating asset payload (%s) : %s", v.TraceID, assetID, *ua.AssetPayload)
		}

		if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to update asset : %s %s", v.TraceID, contractAddr, assetID)
			return err
		}
		logger.Info(ctx, "%s : Updated asset : %s %s", v.TraceID, contractAddr, assetID)
	}

	return nil
}
