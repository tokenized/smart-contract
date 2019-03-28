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

const (
	FieldCount = 10 // The number of fields that can be changed with amendments.
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
	ctx, span := trace.StartSpan(ctx, "internal.asset.Create")
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

	a.Holdings = make([]state.Holding, 1)
	a.Holdings = append(a.Holdings, state.Holding{
		PKH:       nu.IssuerPKH,
		Balance:   nu.TokenQty,
		CreatedAt: a.CreatedAt,
	})

	if a.AssetPayload == nil {
		a.AssetPayload = []byte{}
	}

	return Save(ctx, dbConn, contractPKH, &a)
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
	if upd.IssuerProposal != nil {
		a.IssuerProposal = *upd.IssuerProposal
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

	// Update balances
	if upd.NewBalances != nil {
		for pkh, balance := range upd.NewBalances {
			UpdateBalance(ctx, a, &pkh, balance, now)
		}
	}

	// Update holding statuses
	if upd.NewHoldingStatuses != nil {
		for pkh, status := range upd.NewHoldingStatuses {
			holding := GetHolding(ctx, a, &pkh, now)
			holding.HoldingStatuses = append(holding.HoldingStatuses, status)
			SetHolding(ctx, a, &pkh, &holding)
		}
	}

	// Clear holding statuses
	if upd.ClearHoldingStatuses != nil {
		for pkh, txid := range upd.ClearHoldingStatuses {
			holding := GetHolding(ctx, a, &pkh, now)
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
			SetHolding(ctx, a, &pkh, &holding)
		}
	}

	a.UpdatedAt = now

	return Save(ctx, dbConn, contractPKH, a)
}

// GetHolding will return a users holding or make one for them.
func GetHolding(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, now protocol.Timestamp) state.Holding {
	// Find userPKH
	for _, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			return holding
		}
	}

	// New holding
	return state.Holding{
		PKH:       *userPKH,
		Balance:   0,
		CreatedAt: now,
	}
}

// SetHolding will return a users holding or make one for them.
func SetHolding(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, newHolding *state.Holding) {
	// Find userPKH
	for i, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			as.Holdings[i] = *newHolding
			return
		}
	}

	// Add new holding
	as.Holdings = append(as.Holdings, *newHolding)
}

// UpdateBalance will set the balance of a users holdings against the supplied asset.
// New holdings are created for new users and expired holding statuses are cleared.
func UpdateBalance(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, balance uint64, now protocol.Timestamp) error {
	// TODO Check timestamp against latest timestamp on each holding and only update if the timestamp is newer.
	// Set balance
	holding := GetHolding(ctx, as, userPKH, now)
	holding.Balance = balance

	// Clear expired holding status
	for {
		removed := false
		for i, status := range holding.HoldingStatuses {
			if HoldingStatusExpired(ctx, &status, now) {
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
	SetHolding(ctx, as, userPKH, &holding)
	return nil
}

// GetBalance returns the balance for a PKH holder
func GetBalance(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash) uint64 {
	// Find userPKH
	for _, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			return holding.Balance
		}
	}

	return 0
}

// GetBalance returns the balance for a PKH holder
func GetVotingBalance(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, applyMultiplier bool, now protocol.Timestamp) uint64 {
	if !as.VotingRights {
		return 0
	}

	// Find userPKH
	for _, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			unfrozenBalance := holding.Balance
			for _, status := range holding.HoldingStatuses {
				if HoldingStatusExpired(ctx, &status, now) {
					continue
				}
				if status.Balance > unfrozenBalance {
					unfrozenBalance = 0
				} else {
					unfrozenBalance -= status.Balance
				}
			}

			if applyMultiplier {
				return unfrozenBalance * uint64(as.VoteMultiplier)
			}
			return unfrozenBalance
		}
	}

	return 0
}

// CheckHolding checks to see if the user has a holding.
func CheckHolding(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash) bool {
	// Find userPKH
	for _, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			return true
		}
	}

	return false
}

// CheckBalance checks to see if the user has the specified balance available
func CheckBalance(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, amount uint64) bool {
	// Find userPKH
	for _, holding := range asset.Holdings {
		if holding.PKH.Equal(*userPKH) {
			return holding.Balance >= amount
		}
	}

	return false
}

// CheckBalanceFrozen checks to see if the user has the specified unfrozen balance available
// NB(srg): Amount remains as an argument because it will be a consideration in future
func CheckBalanceFrozen(ctx context.Context, asset *state.Asset, userPKH *protocol.PublicKeyHash, amount uint64, now protocol.Timestamp) bool {
	// Find userPKH
	for _, holding := range asset.Holdings {
		if holding.PKH.Equal(*userPKH) {
			result := holding.Balance
			for _, status := range holding.HoldingStatuses {
				if HoldingStatusExpired(ctx, &status, now) {
					continue
				}
				if status.Balance > result {
					return false // Unfrozen balance is negative
				}
				result -= status.Balance
			}
			return result >= amount
		}
	}

	return false
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

// ValidateVoting returns an error if voting is not allowed.
func ValidateVoting(ctx context.Context, as *state.Asset, initiatorType uint8, votingSystem *protocol.VotingSystem) error {
	switch initiatorType {
	case 0: // Issuer
		if !as.IssuerProposal {
			return errors.New("Issuer proposals not allowed")
		}
	case 1: // Holder
		if !as.HolderProposal {
			return errors.New("Holder proposals not allowed")
		}
	}

	return nil
}

func ValidatePermissions(ctx context.Context, as *state.Asset, votingSystemCount int,
	proposedAmendments []protocol.Amendment, viaProposal bool, initiatorType, votingSystem uint8) error {
	permissions, err := protocol.ReadAuthFlags(as.AssetAuthFlags, FieldCount, votingSystemCount)
	if err != nil {
		return err
	}

	for _, amendment := range proposedAmendments {
		if amendment.FieldIndex >= FieldCount {
			return errors.New("Field index out of range")
		}

		if viaProposal {
			switch initiatorType {
			case 0: // Issuer
				if !permissions[amendment.FieldIndex].IssuerProposal {
					return fmt.Errorf("Issuer proposals not permitted to amend field %d", amendment.FieldIndex)
				}
			case 1: // Holder
				if !permissions[amendment.FieldIndex].HolderProposal {
					return fmt.Errorf("Holder proposals not permitted to amend field %d", amendment.FieldIndex)
				}
			}
			if !permissions[amendment.FieldIndex].VotingSystemsAllowed[votingSystem] {
				return fmt.Errorf("Voting system %d not permitted to amend field %d", votingSystem, amendment.FieldIndex)
			}
		} else {
			if !permissions[amendment.FieldIndex].Permitted {
				return fmt.Errorf("Direct amendments not permitted on field %d", amendment.FieldIndex)
			}
		}
	}

	return nil
}
