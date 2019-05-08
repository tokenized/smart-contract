package asset

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/logger"
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

	a.Holdings = make([]state.Holding, 0, 1)
	a.Holdings = append(a.Holdings, state.Holding{
		PKH:       nu.AdministrationPKH,
		Balance:   nu.TokenQty,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
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

	// Update balances
	if upd.NewBalances != nil {
		for pkh, balance := range upd.NewBalances {
			if err := updateBalance(ctx, a, &pkh, balance, now); err != nil {
				return errors.Wrap(err, "Failed to update balance")
			}
		}
	}

	// Update holding statuses
	if upd.NewHoldingStatuses != nil {
		for pkh, status := range upd.NewHoldingStatuses {
			if err := addHoldingStatus(ctx, a, &pkh, &status); err != nil {
				return errors.Wrap(err, "Failed to add holding status")
			}
		}
	}

	// Clear holding statuses
	if upd.ClearHoldingStatuses != nil {
		for pkh, txid := range upd.ClearHoldingStatuses {
			if err := removeHoldingStatus(ctx, a, &pkh, &txid); err != nil {
				return errors.Wrap(err, "Failed to remove holding status")
			}
		}
	}

	a.UpdatedAt = now

	return Save(ctx, dbConn, contractPKH, a)
}

// UpdateBalance will set the balance of a users holdings against the supplied asset.
// New holdings are created for new users and expired holding statuses are cleared.
func updateBalance(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, balance uint64, now protocol.Timestamp) error {
	// Find userPKH
	for i, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			updated := false
			// Clear expired holding statuses.
			for {
				removed := false
				for j, status := range holding.HoldingStatuses {
					if HoldingStatusExpired(ctx, &status, now) {
						// Remove expired status
						holding.HoldingStatuses = append(holding.HoldingStatuses[:j], holding.HoldingStatuses[j+1:]...)
						removed = true
						updated = true
						break
					}
				}
				if !removed {
					break
				}
			}

			// Only update if this is the latest update.
			if now.Nano() > holding.UpdatedAt.Nano() {
				logger.Verbose(ctx, "Updating balance to %d for %s", balance, userPKH.String())
				holding.Balance = balance
				updated = true
			} else {
				logger.Warn(ctx, "Balance update old %d : %s (%s > %s)", balance, userPKH.String(), now.String(), holding.UpdatedAt.String())
			}

			if updated {
				as.Holdings[i] = holding
			}

			return nil
		}
	}

	// New holding
	logger.Verbose(ctx, "Creating new balance to %d for %s", balance, userPKH.String())
	as.Holdings = append(as.Holdings, state.Holding{
		PKH:       *userPKH,
		Balance:   balance,
		CreatedAt: now,
		UpdatedAt: now,
	})

	return nil
}

func addHoldingStatus(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, st *state.HoldingStatus) error {
	// Find userPKH
	for i, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			for _, status := range holding.HoldingStatuses {
				if status.Code == st.Code && bytes.Equal(status.TxId.Bytes(), st.TxId.Bytes()) {
					return errors.New("already exists")
				}
			}

			holding.HoldingStatuses = append(holding.HoldingStatuses, *st)
			as.Holdings[i] = holding
			return nil
		}
	}

	return errors.New("holding not found")
}

func removeHoldingStatus(ctx context.Context, as *state.Asset, userPKH *protocol.PublicKeyHash, txid *protocol.TxId) error {
	// Find userPKH
	for i, holding := range as.Holdings {
		if holding.PKH.Equal(*userPKH) {
			// Clear expired holding statuses.
			for j, status := range holding.HoldingStatuses {
				if bytes.Equal(status.TxId.Bytes(), txid.Bytes()) {
					holding.HoldingStatuses = append(holding.HoldingStatuses[:j], holding.HoldingStatuses[j+1:]...)
					as.Holdings[i] = holding
					return nil
				}
			}

			return errors.New("status not found")
		}
	}

	return errors.New("holding not found")
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
