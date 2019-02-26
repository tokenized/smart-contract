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
