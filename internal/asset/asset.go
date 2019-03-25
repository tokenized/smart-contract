package asset

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"

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
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode) (*state.Asset, error) {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Retrieve")
	defer span.End()

	// Find asset in storage
	a, err := Fetch(ctx, dbConn, contractPKH, assetCode)
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
func Create(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode, nu *NewAsset, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Update")
	defer span.End()

	// Set up asset
	var a state.Asset

	// Get current state
	err := node.Convert(ctx, &nu, &a)
	if err != nil {
		return err
	}

	a.ID = *assetCode
	a.Revision = 0
	a.CreatedAt = now
	a.UpdatedAt = now

	a.Holdings = make(map[protocol.PublicKeyHash]*state.Holding, 1)
	a.Holdings[nu.IssuerAddress] = &state.Holding{
		Address:   nu.IssuerAddress,
		Balance:   nu.TokenQty,
		CreatedAt: a.CreatedAt,
	}

	if a.AssetPayload == nil {
		a.AssetPayload = []byte{}
	}

	if err := Save(ctx, dbConn, contractPKH, &a); err != nil {
		return err
	}
	return nil
}

// Update the asset
func Update(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode, upd *UpdateAsset, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.asset.Update")
	defer span.End()

	// Find asset
	a, err := Fetch(ctx, dbConn, contractPKH, assetCode)
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
	if upd.ReferendumProposal != nil {
		a.ReferendumProposal = *upd.ReferendumProposal
	}
	if upd.InitiativeProposal != nil {
		a.InitiativeProposal = *upd.InitiativeProposal
	}
	if upd.AssetModificationGovernance != nil {
		a.AssetModificationGovernance = *upd.AssetModificationGovernance
	}
	if upd.TokenQty != nil {
		a.TokenQty = *upd.TokenQty
	}
	if upd.ContractFeeCurrency != nil {
		a.ContractFeeCurrency = *upd.ContractFeeCurrency
	}
	if upd.ContractFeeVar != nil {
		a.ContractFeeVar = *upd.ContractFeeVar
	}
	if upd.ContractFeeFixed != nil {
		a.ContractFeeFixed = *upd.ContractFeeFixed
	}
	if upd.AssetPayload != nil {
		a.AssetPayload = *upd.AssetPayload
	}

	// Update balances
	if upd.NewBalances != nil {
		for pkh, balance := range upd.NewBalances {
			UpdateBalance(ctx, a, &pkh, balance, now)
		}
	}

	// Update holding statuses
	if upd.NewHoldingStatuses != nil {
		for pkh, status := range upd.NewHoldingStatuses {
			holding := MakeHolding(ctx, a, &pkh, now)
			holding.HoldingStatuses = append(holding.HoldingStatuses, status)
			a.Holdings[pkh] = holding
		}
	}

	// Clear holding statuses
	if upd.ClearHoldingStatuses != nil {
		for pkh, txid := range upd.ClearHoldingStatuses {
			holding := MakeHolding(ctx, a, &pkh, now)
			found := false
			for i, status := range holding.HoldingStatuses {
				if bytes.Equal(status.TxId.Bytes(), txid.Bytes()) {
					// Remove holding status
					holding.HoldingStatuses = append(holding.HoldingStatuses[:i], holding.HoldingStatuses[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				return errors.New(fmt.Sprintf("Matching hold not found : address %s txid %s", pkh, txid))
			}
			a.Holdings[pkh] = holding
		}
	}

	a.UpdatedAt = protocol.CurrentTimestamp()

	if err := Save(ctx, dbConn, contractPKH, a); err != nil {
		return err
	}
	return nil
}

// MakeHolding will return a users holding or make one for them.
func MakeHolding(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, now protocol.Timestamp) *state.Holding {
	holding, ok := asset.Holdings[*userPKH]

	// New holding
	if !ok {
		*holding = state.Holding{
			Address:   *userPKH,
			Balance:   0,
			CreatedAt: now,
		}
	}

	return holding
}

// UpdateBalance will set the balance of a users holdings against the supplied asset.
// New holdings are created for new users and expired holding statuses are cleared.
func UpdateBalance(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, balance uint64, now protocol.Timestamp) error {
	// TODO Check timestamp against latest timestamp on each holding and only update if the timestamp is newer.
	// Set balance
	holding := MakeHolding(ctx, asset, userPKH, now)
	holding.Balance = balance

	// Clear expired holding status
	for {
		removed := false
		for i, status := range holding.HoldingStatuses {
			if HoldingStatusExpired(ctx, status, now) {
				// Remove expired status
				holding.HoldingStatuses = append(holding.HoldingStatuses[:i], holding.HoldingStatuses[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}

	// Put the holding back on the asset
	asset.Holdings[*userPKH] = holding
	return nil
}

// GetBalance returns the balance for a PKH holder
func GetBalance(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash) uint64 {
	holding, ok := asset.Holdings[*userPKH]
	if !ok {
		return 0
	}
	return holding.Balance
}

// CheckBalance checks to see if the user has the specified balance available
func CheckBalance(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, amount uint64) bool {
	holding, ok := asset.Holdings[*userPKH]
	if !ok {
		return false
	}
	return holding.Balance >= amount
}

// CheckBalanceFrozen checks to see if the user has the specified unfrozen balance available
// NB(srg): Amount remains as an argument because it will be a consideration in future
func CheckBalanceFrozen(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, amount uint64, now protocol.Timestamp) bool {
	holding, ok := asset.Holdings[*userPKH]
	if !ok {
		return false
	}

	result := holding.Balance
	for _, status := range holding.HoldingStatuses {
		if HoldingStatusExpired(ctx, status, now) {
			continue
		}
		if status.Balance > result {
			return false // Unfrozen balance is negative
		}
		result -= status.Balance
	}
	return result >= amount
}

// HoldingStatusExpired checks to see if a holding status has expired
func HoldingStatusExpired(ctx context.Context, hs *state.HoldingStatus, now protocol.Timestamp) bool {
	if hs.Expires.Nano() == 0 {
		return false
	}

	// Current time is after expiry, so this order has expired.
	if now.Nano() > hs.Expires.Nano() {
		return true
	}
	return false
}
