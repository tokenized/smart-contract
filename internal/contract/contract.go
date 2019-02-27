package contract

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
	ErrNotFound = errors.New("Contract not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified contract from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, address string) (*state.Contract, error) {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Retrieve")
	defer span.End()

	// Find contract in storage
	c, err := Fetch(ctx, dbConn, address)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return c, nil
}

// Create the contract
func Create(ctx context.Context, dbConn *db.DB, address string, nu *NewContract, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Create")
	defer span.End()

	// Find contract
	c, err := Fetch(ctx, dbConn, address)
	if err != nil {
		return ErrNotFound
	}

	c.ContractName = nu.ContractName
	c.ContractFileHash = nu.ContractFileHash
	c.GoverningLaw = nu.GoverningLaw
	c.Jurisdiction = nu.Jurisdiction
	c.ContractExpiration = nu.ContractExpiration
	c.URI = nu.URI
	c.Revision = 0
	c.IssuerID = nu.IssuerID
	c.ContractOperatorID = nu.ContractOperatorID
	c.AuthorizationFlags = nu.AuthorizationFlags
	c.InitiativeThreshold = nu.InitiativeThreshold
	c.InitiativeThresholdCurrency = nu.InitiativeThresholdCurrency
	c.Qty = nu.Qty
	c.IssuerType = nu.IssuerType
	c.VotingSystem = nu.VotingSystem

	if c.AuthorizationFlags == nil {
		c.AuthorizationFlags = []byte{}
	}

	if nu.VotingSystem == string(0x0) {
		c.VotingSystem = ""
	}

	if nu.IssuerType == string(0x0) {
		c.IssuerType = ""
	}

	if err := Save(ctx, dbConn, *c); err != nil {
		return err
	}

	return nil
}

// Update the contract
func Update(ctx context.Context, dbConn *db.DB, address string, upd *UpdateContract, now time.Time) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Update")
	defer span.End()

	// Find contract
	c, err := Fetch(ctx, dbConn, address)
	if err != nil {
		return ErrNotFound
	}

	// Update fields
	if upd.ContractName != nil {
		c.ContractName = *upd.ContractName
	}
	if upd.ContractFileHash != nil {
		c.ContractFileHash = *upd.ContractFileHash
	}
	if upd.GoverningLaw != nil {
		c.GoverningLaw = *upd.GoverningLaw
	}
	if upd.Jurisdiction != nil {
		c.Jurisdiction = *upd.Jurisdiction
	}
	if upd.ContractExpiration != nil {
		c.ContractExpiration = *upd.ContractExpiration
	}
	if upd.URI != nil {
		c.URI = *upd.URI
	}
	if upd.Revision != nil {
		c.Revision = *upd.Revision
	}
	if upd.IssuerID != nil {
		c.IssuerID = *upd.IssuerID
	}
	if upd.ContractOperatorID != nil {
		c.ContractOperatorID = *upd.ContractOperatorID
	}
	if upd.AuthorizationFlags != nil {
		c.AuthorizationFlags = upd.AuthorizationFlags
	}
	if upd.InitiativeThreshold != nil {
		c.InitiativeThreshold = *upd.InitiativeThreshold
	}
	if upd.InitiativeThresholdCurrency != nil {
		c.InitiativeThresholdCurrency = *upd.InitiativeThresholdCurrency
	}
	if upd.Qty != nil {
		c.Qty = *upd.Qty
	}

	if upd.IssuerType != nil {
		c.IssuerType = *upd.IssuerType
		if c.IssuerType == string(0x0) {
			c.IssuerType = ""
		}
	}
	if upd.VotingSystem != nil {
		c.VotingSystem = *upd.VotingSystem
		if c.IssuerType == string(0x0) {
			c.VotingSystem = ""
		}
	}

	if err := Save(ctx, dbConn, *c); err != nil {
		return err
	}

	return nil
}

// CanHaveMoreAssets returns true if an Asset can be added to the Contract,
// false otherwise.
//
// A "dynamic" contract is permitted to have unlimted assets if the
// contract.Qty == 0.
func CanHaveMoreAssets(ctx context.Context, contract *state.Contract) bool {
	if contract.Qty == 0 {
		return true
	}

	// number of current assets
	total := uint64(len(contract.Assets))

	// more assets can be added if the current total is less than the limit
	// imposed by the contract.
	return total < contract.Qty
}

// HasAnyBalance checks if the user has any balance of any token across the contract
func HasAnyBalance(ctx context.Context, contract *state.Contract, userPKH string) bool {
	for _, a := range contract.Assets {
		if h, ok := a.Holdings[userPKH]; ok && h.Balance > 0 {
			return true
		}
	}

	return false
}

// IsOperator will check if the supplied pkh has operator permission (issuer or operator)
func IsOperator(ctx context.Context, contract *state.Contract, pkh string) bool {
	return contract.IssuerAddress == pkh || contract.OperatorAddress == pkh
}

// IsVotingPermitted returns true if contract allows voting
func IsVotingPermitted(ctx context.Context, contract *state.Contract) bool {
	return contract.VotingSystem != "N"
}
