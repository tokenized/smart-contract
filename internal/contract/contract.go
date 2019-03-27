package contract

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const (
	FieldCount = 21 // The number of fields that can be changed with amendments.
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Contract not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified contract from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash) (*state.Contract, error) {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Retrieve")
	defer span.End()

	// Find contract in storage
	contract, err := Fetch(ctx, dbConn, contractPKH)
	if err != nil {
		return nil, err
	}

	return contract, nil
}

// Create the contract
func Create(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, nu *NewContract, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Create")
	defer span.End()

	// Find contract
	var contract state.Contract

	// Get current state
	err := node.Convert(ctx, &nu, &contract)
	if err != nil {
		return errors.Wrap(err, "Failed to convert new contract to contract")
	}

	contract.ID = *contractPKH
	contract.Revision = 0
	contract.CreatedAt = now
	contract.UpdatedAt = now

	if contract.VotingSystems == nil {
		contract.VotingSystems = []protocol.VotingSystem{}
	}
	if contract.Registries == nil {
		contract.Registries = []protocol.Registry{}
	}

	logger.Verbose(ctx, "Creating contract :\n%+v", &contract)

	if err := Save(ctx, dbConn, &contract); err != nil {
		return errors.Wrap(err, "Failed to create contract")
	}

	return nil
}

// Update the contract
func Update(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, upd *UpdateContract, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Update")
	defer span.End()

	// Find contract
	c, err := Fetch(ctx, dbConn, contractPKH)
	if err != nil {
		return ErrNotFound
	}

	// Update fields
	if upd.Revision != nil {
		c.Revision = *upd.Revision
	}
	if upd.Timestamp != nil {
		c.Timestamp = *upd.Timestamp
	}

	if upd.IssuerPKH != nil {
		c.IssuerPKH = *upd.IssuerPKH
	}
	if upd.OperatorPKH != nil {
		c.OperatorPKH = *upd.OperatorPKH
	}

	if upd.ContractName != nil {
		c.ContractName = *upd.ContractName
	}
	if upd.ContractType != nil {
		c.ContractType = *upd.ContractType
	}
	if upd.BodyOfAgreementType != nil {
		c.BodyOfAgreementType = *upd.BodyOfAgreementType
	}
	if upd.BodyOfAgreement != nil {
		c.BodyOfAgreement = *upd.BodyOfAgreement
	}
	if upd.SupportingDocsFileType != nil {
		c.SupportingDocsFileType = *upd.SupportingDocsFileType
	}
	if upd.SupportingDocs != nil {
		c.SupportingDocs = *upd.SupportingDocs
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
	if upd.Issuer != nil {
		c.Issuer = *upd.Issuer
	}
	if upd.IssuerLogoURL != nil {
		c.IssuerLogoURL = *upd.IssuerLogoURL
	}
	if upd.ContractOperator != nil {
		c.ContractOperator = *upd.ContractOperator
	}
	if upd.ContractAuthFlags != nil {
		c.ContractAuthFlags = *upd.ContractAuthFlags
	}
	if upd.ContractFee != nil {
		c.ContractFee = *upd.ContractFee
	}
	if upd.VotingSystems != nil {
		c.VotingSystems = *upd.VotingSystems
		if c.VotingSystems == nil {
			c.VotingSystems = []protocol.VotingSystem{}
		}
	}
	if upd.RestrictedQtyAssets != nil {
		c.RestrictedQtyAssets = *upd.RestrictedQtyAssets
	}
	if upd.IssuerProposal != nil {
		c.IssuerProposal = *upd.IssuerProposal
	}
	if upd.HolderProposal != nil {
		c.HolderProposal = *upd.HolderProposal
	}
	if upd.Registries != nil {
		c.Registries = *upd.Registries
		if c.Registries == nil {
			c.Registries = []protocol.Registry{}
		}
	}

	c.UpdatedAt = now

	if err := Save(ctx, dbConn, c); err != nil {
		return errors.Wrap(err, "Failed to update contract")
	}

	return nil
}

// AddAssetCode adds an asset code to a contract
func AddAssetCode(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Update")
	defer span.End()

	// Find contract
	contract, err := Fetch(ctx, dbConn, contractPKH)
	if err != nil {
		return ErrNotFound
	}

	contract.AssetCodes = append(contract.AssetCodes, *assetCode)
	contract.UpdatedAt = now

	if err := Save(ctx, dbConn, contract); err != nil {
		return errors.Wrap(err, "Failed to add asset to contract")
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
	total := uint64(len(contract.AssetCodes))

	// more assets can be added if the current total is less than the limit
	// imposed by the contract.
	return total < contract.RestrictedQtyAssets
}

// HasAnyBalance checks if the user has any balance of any token across the contract
func HasAnyBalance(ctx context.Context, dbConn *db.DB, contract *state.Contract, userPKH *protocol.PublicKeyHash) bool {
	for _, a := range contract.AssetCodes {
		as, err := asset.Retrieve(ctx, dbConn, &contract.ID, &a)
		if err != nil {
			continue
		}

		if asset.GetBalance(ctx, as, userPKH) > 0 {
			return true
		}
	}

	return false
}

// IsOperator will check if the supplied pkh has operator permission (issuer or operator)
func IsOperator(ctx context.Context, contract *state.Contract, pkh *protocol.PublicKeyHash) bool {
	return bytes.Equal(contract.IssuerPKH.Bytes(), pkh.Bytes()) || bytes.Equal(contract.OperatorPKH.Bytes(), pkh.Bytes())
}

// IsVotingPermitted returns true if contract allows voting
func IsVotingPermitted(ctx context.Context, contract *state.Contract) bool {
	return len(contract.VotingSystems) != 0
}
