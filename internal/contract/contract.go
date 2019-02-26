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

	// TODO(srg) New protocol spec - This double up in logic is reserved for using
	// conditional pointers where only some fields are updated on the object.
	c.ContractName = upd.ContractName
	c.ContractFileHash = upd.ContractFileHash
	c.GoverningLaw = upd.GoverningLaw
	c.Jurisdiction = upd.Jurisdiction
	c.ContractExpiration = upd.ContractExpiration
	c.URI = upd.URI
	c.Revision = upd.Revision
	c.IssuerID = upd.IssuerID
	c.ContractOperatorID = upd.ContractOperatorID
	c.AuthorizationFlags = upd.AuthorizationFlags
	c.InitiativeThreshold = upd.InitiativeThreshold
	c.InitiativeThresholdCurrency = upd.InitiativeThresholdCurrency
	c.Qty = upd.Qty
	c.IssuerType = upd.IssuerType
	c.VotingSystem = upd.VotingSystem

	if upd.VotingSystem == string(0x0) {
		c.VotingSystem = ""
	}

	if upd.IssuerType == string(0x0) {
		c.IssuerType = ""
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
