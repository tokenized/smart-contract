package asset

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const (
	FieldCount = 12 // The number of fields that can be changed with amendments.
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Asset not found")
)

// Retrieve gets the specified asset from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	assetCode *protocol.AssetCode) (*state.Asset, error) {

	ctx, span := trace.StartSpan(ctx, "internal.asset.Retrieve")
	defer span.End()

	// Find asset in storage
	a, err := Fetch(ctx, dbConn, contractAddress, assetCode)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Create the asset
func Create(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	assetCode *protocol.AssetCode, nu *NewAsset, now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.asset.Create")
	defer span.End()

	// Set up asset
	var a state.Asset

	// Get current state
	err := node.Convert(ctx, &nu, &a)
	if err != nil {
		return err
	}

	a.Code = assetCode
	a.Revision = 0
	a.CreatedAt = now
	a.UpdatedAt = now

	// a.Holdings = make(map[protocol.PublicKeyHash]*state.Holding)
	// a.Holdings[nu.AdministrationPKH] = &state.Holding{
	// 	PKH:              nu.AdministrationPKH,
	// 	PendingBalance:   nu.TokenQty,
	// 	FinalizedBalance: nu.TokenQty,
	// 	CreatedAt:        a.CreatedAt,
	// 	UpdatedAt:        a.UpdatedAt,
	// }

	if a.AssetPayload == nil {
		a.AssetPayload = []byte{}
	}

	return Save(ctx, dbConn, contractAddress, &a)
}

// Update the asset
func Update(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	assetCode *protocol.AssetCode, upd *UpdateAsset, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Update")
	defer span.End()

	// Find asset
	a, err := Fetch(ctx, dbConn, contractAddress, assetCode)
	if err != nil {
		return ErrNotFound
	}

	// Update fields
	if upd.Revision != nil {
		a.Revision = *upd.Revision
	}
	if upd.Timestamp != nil {
		a.Timestamp = *upd.Timestamp
	}

	if upd.AssetType != nil {
		a.AssetType = *upd.AssetType
	}
	if upd.TransfersPermitted != nil {
		a.TransfersPermitted = *upd.TransfersPermitted
	}
	if upd.TradeRestrictions != nil {
		a.TradeRestrictions = *upd.TradeRestrictions
	}
	if upd.EnforcementOrdersPermitted != nil {
		a.EnforcementOrdersPermitted = *upd.EnforcementOrdersPermitted
	}
	if upd.VoteMultiplier != nil {
		a.VoteMultiplier = *upd.VoteMultiplier
	}
	if upd.AdministrationProposal != nil {
		a.AdministrationProposal = *upd.AdministrationProposal
	}
	if upd.HolderProposal != nil {
		a.HolderProposal = *upd.HolderProposal
	}
	if upd.AssetModificationGovernance != nil {
		a.AssetModificationGovernance = *upd.AssetModificationGovernance
	}
	if upd.TokenQty != nil {
		a.TokenQty = *upd.TokenQty
	}
	if upd.AssetPayload != nil {
		a.AssetPayload = *upd.AssetPayload
	}
	if upd.FreezePeriod != nil {
		a.FreezePeriod = *upd.FreezePeriod
	}

	a.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, a)
}

// ValidateVoting returns an error if voting is not allowed.
func ValidateVoting(ctx context.Context, as *state.Asset, initiatorType uint32,
	votingSystem *actions.VotingSystemField) error {

	switch initiatorType {
	case 0: // Administration
		if !as.AdministrationProposal {
			return errors.New("Administration proposals not allowed")
		}
	case 1: // Holder
		if !as.HolderProposal {
			return errors.New("Holder proposals not allowed")
		}
	}

	return nil
}
