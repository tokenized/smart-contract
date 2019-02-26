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

	// ErrNoHoldings occurs when an asset is corrupt.
	ErrNoHoldings = errors.New("Asset has no holdings")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified asset from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH, assetID string) (*state.Asset, error) {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Retrieve")
	defer span.End()

	// Find asset in storage
	a, err := Fetch(ctx, dbConn, contractPKH, assetID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	// A valid asset should contain at least one holding
	if a.Holdings == nil {
		return nil, ErrNoHoldings
	}

	return a, nil
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

	// Update fields
	if upd.Type != nil {
		a.Type = *upd.Type
	}
	if upd.Revision != nil {
		a.Revision = *upd.Revision
	}
	if upd.VotingSystem != nil {
		a.VotingSystem = *upd.VotingSystem
	}
	if upd.VoteMultiplier != nil {
		a.VoteMultiplier = *upd.VoteMultiplier
	}
	if upd.Qty != nil {
		a.Qty = *upd.Qty
	}
	if upd.AuthorizationFlags != nil {
		a.AuthorizationFlags = upd.AuthorizationFlags
	}

	// Update balances
	if upd.NewBalances != nil {
		for pkh, balance := range upd.NewBalances {
			UpdateBalance(ctx, a, pkh, balance, now)
		}
	}

	// Update holding statuses
	if upd.NewHoldingStatus != nil {
		for pkh, status := range upd.NewHoldingStatus {
			holding := MakeHolding(ctx, a, pkh, now)
			holding.HoldingStatus = status
			a.Holdings[pkh] = *holding
		}
	}

	if err := Save(ctx, dbConn, contractPKH, *a); err != nil {
		return err
	}

	return nil
}

// MakeHolding will return a users holding or make one for them.
func MakeHolding(ctx context.Context, asset *state.Asset, userPKH string, now time.Time) *state.Holding {

	holding, ok := asset.Holdings[userPKH]

	// New holding
	if !ok {
		holding = state.Holding{
			Address:   userPKH,
			Balance:   0,
			CreatedAt: now.UnixNano(),
		}
	}

	return &holding
}

// UpdateBalance will set the balance of a users holdings against the supplied asset.
// New holdings are created for new users and expired holding statuses are cleared.
func UpdateBalance(ctx context.Context, asset *state.Asset, userPKH string, balance uint64, now time.Time) error {

	// Set balance
	holding := MakeHolding(ctx, asset, userPKH, now)
	holding.Balance = balance

	// Clear expired holding status
	if holding.HoldingStatus != nil && HoldingStatusExpired(ctx, holding.HoldingStatus, now) {
		holding.HoldingStatus = nil
	}

	// Put the holding back on the asset
	asset.Holdings[userPKH] = *holding

	return nil
}

// GetBalance returns the balance for a PKH holder
func GetBalance(ctx context.Context, asset *state.Asset, userPKH string) uint64 {
	holding, ok := asset.Holdings[userPKH]
	if !ok {
		return 0
	}

	return holding.Balance
}

// CheckBalance checks to see if the user has the specified balance available
func CheckBalance(ctx context.Context, asset *state.Asset, userPKH string, amount uint64) bool {
	holding, ok := asset.Holdings[userPKH]
	if !ok {
		return false
	}

	return holding.Balance >= amount
}

// CheckBalanceFrozen checks to see if the user has the specified unfrozen balance available
// NB(srg): Amount remains as an argument because it will be a consideration in future
func CheckBalanceFrozen(ctx context.Context, asset *state.Asset, userPKH string, amount uint64, now time.Time) bool {
	holding, ok := asset.Holdings[userPKH]
	if !ok {
		return false
	}

	if holding.HoldingStatus != nil {
		return HoldingStatusExpired(ctx, holding.HoldingStatus, now)
	}

	return true
}

// HoldingStatusExpired checks to see if a holding status has expired
func HoldingStatusExpired(ctx context.Context, hs *state.HoldingStatus, now time.Time) bool {
	if hs.Expires == 0 {
		return false
	}

	// Current time is after expiry, so this order has expired.
	if now.Unix() > int64(hs.Expires) {
		return true
	}

	return false
}
