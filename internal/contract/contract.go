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
	var c state.Contract

	c.ID = address
	c.Revision = 0
	c.CreatedAt = uint64(time.Now().UnixNano())
	c.UpdatedAt = c.CreatedAt

	c.ContractName = nu.ContractName
	c.ContractFileType = nu.ContractFileType
	c.ContractFile = nu.ContractFile
	c.GoverningLaw = nu.GoverningLaw
	c.Jurisdiction = nu.Jurisdiction
	c.ContractExpiration = nu.ContractExpiration
	c.ContractURI = nu.ContractURI
	c.IssuerName = nu.IssuerName
	c.IssuerType = nu.IssuerType
	c.IssuerLogoURL = nu.IssuerLogoURL
	c.ContractOperatorID = nu.ContractOperatorID
	c.ContractAuthFlags = nu.ContractAuthFlags
	c.VotingSystems = nu.VotingSystems
	c.RestrictedQtyAssets = nu.RestrictedQtyAssets
	c.ReferendumProposal = nu.ReferendumProposal
	c.InitiativeProposal = nu.InitiativeProposal
	c.Registries = nu.Registries
	c.UnitNumber = nu.UnitNumber
	c.BuildingNumber = nu.BuildingNumber
	c.Street = nu.Street
	c.SuburbCity = nu.SuburbCity
	c.TerritoryStateProvinceCode = nu.TerritoryStateProvinceCode
	c.CountryCode = nu.CountryCode
	c.PostalZIPCode = nu.PostalZIPCode
	c.EmailAddress = nu.EmailAddress
	c.PhoneNumber = nu.PhoneNumber
	c.KeyRoles = nu.KeyRoles
	c.NotableRoles = nu.NotableRoles

	if c.ContractAuthFlags == nil {
		c.ContractAuthFlags = []byte{}
	}
	if c.VotingSystems == nil {
		c.VotingSystems = []state.VotingSystem{}
	}
	if c.Registries == nil {
		c.Registries = []state.Registry{}
	}
	if c.KeyRoles == nil {
		c.KeyRoles = []state.KeyRole{}
	}
	if c.NotableRoles == nil {
		c.NotableRoles = []state.NotableRole{}
	}

	if err := Save(ctx, dbConn, c); err != nil {
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
	if upd.IssuerAddress != nil {
		c.IssuerAddress = *upd.IssuerAddress
	}
	if upd.OperatorAddress != nil {
		c.OperatorAddress = *upd.OperatorAddress
	}

	if upd.ContractName != nil {
		c.ContractName = *upd.ContractName
	}
	if upd.ContractFileType != nil {
		c.ContractFileType = *upd.ContractFileType
	}
	if upd.ContractFile != nil {
		c.ContractFile = *upd.ContractFile
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
	if upd.ContractURI != nil {
		c.ContractURI = *upd.ContractURI
	}
	if upd.IssuerName != nil {
		c.IssuerName = *upd.IssuerName
	}
	if upd.IssuerType != nil {
		c.IssuerType = *upd.IssuerType
	}
	if upd.IssuerLogoURL != nil {
		c.IssuerLogoURL = *upd.IssuerLogoURL
	}
	if upd.ContractOperatorID != nil {
		c.ContractOperatorID = *upd.ContractOperatorID
	}
	if upd.ContractAuthFlags != nil {
		c.ContractAuthFlags = *upd.ContractAuthFlags
		if c.ContractAuthFlags == nil {
			c.ContractAuthFlags = []byte{}
		}
	}
	if upd.VotingSystems != nil {
		c.VotingSystems = *upd.VotingSystems
		if c.VotingSystems == nil {
			c.VotingSystems = []state.VotingSystem{}
		}
	}
	if upd.RestrictedQtyAssets != nil {
		c.RestrictedQtyAssets = *upd.RestrictedQtyAssets
	}
	if upd.ReferendumProposal != nil {
		c.ReferendumProposal = *upd.ReferendumProposal
	}
	if upd.InitiativeProposal != nil {
		c.InitiativeProposal = *upd.InitiativeProposal
	}
	if upd.Registries != nil {
		c.Registries = *upd.Registries
		if c.Registries == nil {
			c.Registries = []state.Registry{}
		}
	}
	if upd.UnitNumber != nil {
		c.UnitNumber = *upd.UnitNumber
	}
	if upd.BuildingNumber != nil {
		c.BuildingNumber = *upd.BuildingNumber
	}
	if upd.Street != nil {
		c.Street = *upd.Street
	}
	if upd.SuburbCity != nil {
		c.SuburbCity = *upd.SuburbCity
	}
	if upd.TerritoryStateProvinceCode != nil {
		c.TerritoryStateProvinceCode = *upd.TerritoryStateProvinceCode
	}
	if upd.CountryCode != nil {
		c.CountryCode = *upd.CountryCode
	}
	if upd.PostalZIPCode != nil {
		c.PostalZIPCode = *upd.PostalZIPCode
	}
	if upd.EmailAddress != nil {
		c.EmailAddress = *upd.EmailAddress
	}
	if upd.PhoneNumber != nil {
		c.PhoneNumber = *upd.PhoneNumber
	}
	if upd.KeyRoles != nil {
		c.KeyRoles = *upd.KeyRoles
		if c.KeyRoles == nil {
			c.KeyRoles = []state.KeyRole{}
		}
	}
	if upd.NotableRoles != nil {
		c.NotableRoles = *upd.NotableRoles
		if c.NotableRoles == nil {
			c.NotableRoles = []state.NotableRole{}
		}
	}

	c.UpdatedAt = uint64(time.Now().UnixNano())

	if err := Save(ctx, dbConn, *c); err != nil {
		return err
	}

	return nil
}

// CanHaveMoreAssets returns true if an Asset can be added to the Contract,
// false otherwise.
//
// A "dynamic" contract is permitted to have unlimited assets if the
// contract.Qty == 0.
func CanHaveMoreAssets(ctx context.Context, contract *state.Contract) bool {
	if contract.RestrictedQtyAssets == 0 {
		return true
	}

	// number of current assets
	total := uint64(len(contract.Assets))

	// more assets can be added if the current total is less than the limit
	// imposed by the contract.
	return total < contract.RestrictedQtyAssets
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
	return len(contract.VotingSystems) != 0
}
