package handlers

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB *db.DB
	Config   *node.Config
}

// DefinitionRequest handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) DefinitionRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr.String())
		return node.ErrNoResponse
	}

	// Verify issuer is sender of tx.
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.IssuerPKH.Bytes()) {
		logger.Warn(ctx, "%s : Only issuer can create assets: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeIssuerAddress)
	}

	// Locate Asset
	_, err = asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != asset.ErrNotFound {
		if err != nil {
			logger.Warn(ctx, "%s : Error retrieving asset : %s", v.TraceID, msg.AssetCode.String())
		} else {
			logger.Warn(ctx, "%s : Asset already exists : %s", v.TraceID, msg.AssetCode.String())
		}
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeDuplicateAssetCode)
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		logger.Verbose(ctx, "%s : Number of assets exceeds contract Qty: %s %x", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeFixedQuantity)
	}

	// TODO Validate all fields have valid values.

	logger.Info(ctx, "%s : Accepting asset creation request : %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())

	// Asset Creation <- Asset Definition
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, &msg, &ac)
	if err != nil {
		return err
	}

	ac.Timestamp = v.Now

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractAddr.Bytes(), &a.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &ac)
}

// ModificationRequest handles an incoming Asset Modification and prepares a Creation response
func (a *Asset) ModificationRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetModification)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetModification")
	}

	dbConn := a.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}
	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Verbose(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		logger.Verbose(ctx, "%s : Asset Revision does not match current: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetRevision)
	}

	// @TODO: When reducing an assets available supply, the amount must
	// be deducted from the issuers balance, otherwise the action cannot
	// be performed. i.e: Reduction amount must not be in circulation.

	// @TODO: Likewise when the asset quantity is increased, the amount
	// must be added to the issuers holding balance.

	// Asset Creation <- Asset Modification
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, &as, &ac)
	if err != nil {
		return err
	}

	ac.AssetRevision = as.Revision + 1
	ac.Timestamp = v.Now

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
	// logger.Warn(ctx, "%s : Incorrect asset amendment field offset (%s) : %d", v.TraceID, assetCode, amendment.FieldIndex)
	// return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetMalformedAmendment)
	// }
	// }

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractAddr.Bytes(), &a.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &ac)
}

// CreationResponse handles an outgoing Asset Creation and writes it to the state
func (a *Asset) CreationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetCreation")
	}

	dbConn := a.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractAddr.Bytes()) {
		return errors.New("Asset Creation not from contract")
	}

	as, err := asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
	if err != nil && err != asset.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Create or update Asset
	if as == nil {
		// Prepare creation object
		na := asset.NewAsset{}

		if err = node.Convert(ctx, &msg, &na); err != nil {
			return err
		}

		if err := contract.AddAssetCode(ctx, dbConn, contractAddr, &msg.AssetCode, v.Now); err != nil {
			return err
		}

		if err := asset.Create(ctx, dbConn, contractAddr, &msg.AssetCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create asset")
		}
		logger.Info(ctx, "%s : Created asset : %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
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
			logger.Info(ctx, "%s : Updating asset type (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetType)
		}
		if !bytes.Equal(as.AssetAuthFlags[:], msg.AssetAuthFlags[:]) {
			ua.AssetAuthFlags = &msg.AssetAuthFlags
			logger.Info(ctx, "%s : Updating asset auth flags (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetAuthFlags)
		}
		if as.TransfersPermitted != msg.TransfersPermitted {
			ua.TransfersPermitted = &msg.TransfersPermitted
			logger.Info(ctx, "%s : Updating asset transfers permitted (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.TransfersPermitted)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			logger.Info(ctx, "%s : Updating asset enforcement orders permitted (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			logger.Info(ctx, "%s : Updating asset vote multiplier (%s) : %02x", v.TraceID, msg.AssetCode.String(), *ua.VoteMultiplier)
		}
		if as.IssuerProposal != msg.IssuerProposal {
			ua.IssuerProposal = &msg.IssuerProposal
			logger.Info(ctx, "%s : Updating asset issuer proposal (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.IssuerProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			logger.Info(ctx, "%s : Updating asset holder proposal (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.HolderProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			logger.Info(ctx, "%s : Updating asset modification governance (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.AssetModificationGovernance)
		}
		if as.TokenQty != msg.TokenQty {
			ua.TokenQty = &msg.TokenQty
			logger.Info(ctx, "%s : Updating asset token quantity (%s) : %d", v.TraceID, msg.AssetCode.String(), *ua.TokenQty)
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			logger.Info(ctx, "%s : Updating asset payload (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetPayload)
		}

		// Check if trade restrictions are different
		different := len(as.TradeRestrictions.Items) != len(msg.TradeRestrictions.Items)
		if !different {
			for i, tradeRestrictions := range as.TradeRestrictions.Items {
				if !bytes.Equal(tradeRestrictions[:], msg.TradeRestrictions.Items[i][:]) {
					different = true
					break
				}
			}
		}

		if different {
			var newTradeRestrictions protocol.Polity
			for _, tradeRestriction := range msg.TradeRestrictions.Items {
				newTradeRestrictions.Items = append(newTradeRestrictions.Items, tradeRestriction)
			}
			ua.TradeRestrictions = &newTradeRestrictions
		}

		if err := asset.Update(ctx, dbConn, contractAddr, &msg.AssetCode, &ua, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to update asset : %s %s", v.TraceID, contractAddr, msg.AssetCode.String())
			return err
		}
		logger.Info(ctx, "%s : Updated asset : %s %s", v.TraceID, contractAddr, msg.AssetCode.String())
	}

	return nil
}
