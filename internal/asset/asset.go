package asset

import (
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"go.opencensus.io/trace"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Asset not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified asset from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH, assetID string) (*state.Asset, error) {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Retrieve")
	defer span.End()

	// Find asset in storage
	c, err := Fetch(ctx, dbConn, contractPKH, assetID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return c, nil
}

// Create the asset
func Create(ctx context.Context, dbConn *db.DB, contractPKH, assetID string, nu *NewAsset, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Update")
	defer span.End()

	// Find asset
	a, err := Fetch(ctx, dbConn, contractPKH, assetID)
	if err != nil {
		return ErrNotFound
	}

	// Set up holdings
	holding := state.Holding{
		Address:   nu.IssuerAddress,
		Balance:   nu.Qty,
		CreatedAt: now.UnixNano(),
	}

	holdings := map[string]state.Holding{
		holding.Address: holding,
	}

	// Set up asset
	a.ID = nu.ID
	a.Type = nu.Type
	a.VotingSystem = nu.VotingSystem
	a.VoteMultiplier = nu.VoteMultiplier
	a.Qty = nu.Qty
	a.Holdings = holdings
	a.CreatedAt = now.UnixNano()

	if a.AuthorizationFlags == nil {
		a.AuthorizationFlags = []byte{}
	}

	if err := Save(ctx, dbConn, contractPKH, *a); err != nil {
		return err
	}

	return nil
}

// Update the asset
func Update(ctx context.Context, dbConn *db.DB, contractPKH, assetID string, upd *UpdateAsset, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Update")
	defer span.End()

	// Find asset
	a, err := Fetch(ctx, dbConn, contractPKH, assetID)
	if err != nil {
		return ErrNotFound
	}

	// TODO(srg) New protocol spec - This double up in logic is reserved for using
	// conditional pointers where only some fields are updated on the object.
	a.Type = upd.Type
	a.Revision = upd.Revision
	a.VotingSystem = upd.VotingSystem
	a.VoteMultiplier = upd.VoteMultiplier
	a.Qty = upd.Qty

	if a.AuthorizationFlags == nil {
		a.AuthorizationFlags = []byte{}
	}

	if err := Save(ctx, dbConn, contractPKH, *a); err != nil {
		return err
	}

	return nil
}
