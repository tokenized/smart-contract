package contract

import (
	"context"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const (
	FieldCount       = 20 // The count of fields that can be changed with amendments.
	EntityFieldCount = 15 // The count of fields in an entity that can be changed with amendments.
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Contract not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified contract from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress) (*state.Contract, error) {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Retrieve")
	defer span.End()

	// Find contract in storage
	contract, err := Fetch(ctx, dbConn, contractAddress)
	if err != nil {
		return nil, err
	}

	return contract, nil
}

// Create the contract
func Create(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress, nu *NewContract,
	now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.contract.Create")
	defer span.End()

	// Find contract
	var contract state.Contract

	// Get current state
	err := node.Convert(ctx, &nu, &contract)
	if err != nil {
		return errors.Wrap(err, "Failed to convert new contract to contract")
	}

	contract.Address = bitcoin.NewJSONRawAddress(contractAddress)
	contract.Revision = 0
	contract.CreatedAt = now
	contract.UpdatedAt = now

	if contract.VotingSystems == nil {
		contract.VotingSystems = []*actions.VotingSystemField{}
	}
	if contract.Oracles == nil {
		contract.Oracles = []*actions.OracleField{}

		if err := ExpandOracles(ctx, &contract); err != nil {
			return err
		}
	}

	return Save(ctx, dbConn, &contract)
}

// Update the contract
func Update(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	upd *UpdateContract, now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.contract.Update")
	defer span.End()

	// Find contract
	c, err := Fetch(ctx, dbConn, contractAddress)
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

	if upd.AdministrationAddress != nil {
		c.AdministrationAddress = upd.AdministrationAddress
	}
	if upd.OperatorAddress != nil {
		c.OperatorAddress = upd.OperatorAddress
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
		c.Issuer = upd.Issuer
	}
	if upd.IssuerLogoURL != nil {
		c.IssuerLogoURL = *upd.IssuerLogoURL
	}
	if upd.ContractOperator != nil {
		c.ContractOperator = upd.ContractOperator
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
			c.VotingSystems = []*actions.VotingSystemField{}
		}
	}
	if upd.RestrictedQtyAssets != nil {
		c.RestrictedQtyAssets = *upd.RestrictedQtyAssets
	}
	if upd.AdministrationProposal != nil {
		c.AdministrationProposal = *upd.AdministrationProposal
	}
	if upd.HolderProposal != nil {
		c.HolderProposal = *upd.HolderProposal
	}
	if upd.Oracles != nil {
		c.Oracles = *upd.Oracles
		if c.Oracles == nil {
			c.Oracles = []*actions.OracleField{}
		}

		if err := ExpandOracles(ctx, c); err != nil {
			return err
		}
	}
	if upd.FreezePeriod != nil {
		c.FreezePeriod = *upd.FreezePeriod
	}

	c.UpdatedAt = now

	return Save(ctx, dbConn, c)
}

// Move marks the contract as moved and copies the data to the new address.
func Move(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	newContractAddress bitcoin.RawAddress, now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.contract.Move")
	defer span.End()

	// Find contract
	c, err := Fetch(ctx, dbConn, contractAddress)
	if err != nil {
		return ErrNotFound
	}

	// Get assets
	assets := make([]*state.Asset, 0, len(c.AssetCodes))
	for _, assetCode := range c.AssetCodes {
		as, err := asset.Retrieve(ctx, dbConn, contractAddress, assetCode)
		if err != nil {
			return err
		}
		assets = append(assets, as)
	}

	// Get votes
	vts, err := vote.List(ctx, dbConn, contractAddress)
	if err != nil {
		return err
	}

	newContract := *c
	newContract.Address = bitcoin.NewJSONRawAddress(newContractAddress)

	c.MovedTo = bitcoin.NewJSONRawAddress(newContractAddress)

	if err = Save(ctx, dbConn, c); err != nil {
		return err
	}
	if err = Save(ctx, dbConn, &newContract); err != nil {
		return err
	}

	// Copy assets
	for _, as := range assets {
		err = asset.Save(ctx, dbConn, newContractAddress, as)
		if err != nil {
			return err
		}
	}

	// Copy votes
	for _, vt := range vts {
		err = vote.Save(ctx, dbConn, newContractAddress, vt)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddAssetCode adds an asset code to a contract
func AddAssetCode(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress, assetCode *protocol.AssetCode, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.contract.Update")
	defer span.End()

	// Find contract
	ct, err := Fetch(ctx, dbConn, contractAddress)
	if err != nil {
		return ErrNotFound
	}

	ct.AssetCodes = append(ct.AssetCodes, assetCode)
	ct.UpdatedAt = now

	if err := Save(ctx, dbConn, ct); err != nil {
		return errors.Wrap(err, "Failed to add asset to contract")
	}

	return nil
}

// CanHaveMoreAssets returns true if an Asset can be added to the Contract,
// false otherwise.
//
// A "dynamic" contract is permitted to have unlimited assets if the
// contract.Qty == 0.
func CanHaveMoreAssets(ctx context.Context, ct *state.Contract) bool {
	if ct.RestrictedQtyAssets == 0 {
		return true
	}

	// number of current assets
	total := uint64(len(ct.AssetCodes))

	// more assets can be added if the current total is less than the limit
	// imposed by the contract.
	return total < ct.RestrictedQtyAssets
}

// HasAnyBalance checks if the user has any balance of any token across the contract.
func HasAnyBalance(ctx context.Context, dbConn *db.DB, ct *state.Contract, userAddress bitcoin.RawAddress) bool {
	for _, a := range ct.AssetCodes {
		h, err := holdings.Fetch(ctx, dbConn, ct.Address, a, userAddress)
		if err != nil {
			continue
		}

		if h.FinalizedBalance > 0 {
			return true
		}
	}

	return false
}

// GetTokenQty returns the token quantities for all assets.
func GetTokenQty(ctx context.Context, dbConn *db.DB, ct *state.Contract, applyMultiplier bool) uint64 {
	result := uint64(0)
	for _, a := range ct.AssetCodes {
		as, err := asset.Retrieve(ctx, dbConn, ct.Address, a)
		if err != nil {
			continue
		}

		if !as.VotingRights {
			continue
		}

		if applyMultiplier {
			result += as.TokenQty * uint64(as.VoteMultiplier)
		} else {
			result += as.TokenQty
		}
	}

	return result
}

// GetVotingBalance returns the tokens held across all of the contract's assets.
func GetVotingBalance(ctx context.Context, dbConn *db.DB, ct *state.Contract,
	userAddress bitcoin.RawAddress, applyMultiplier bool, now protocol.Timestamp) uint64 {

	result := uint64(0)
	for _, a := range ct.AssetCodes {
		as, err := asset.Retrieve(ctx, dbConn, ct.Address, a)
		if err != nil {
			continue
		}

		h, err := holdings.Fetch(ctx, dbConn, ct.Address, a, userAddress)
		if err != nil {
			continue
		}

		result += holdings.VotingBalance(as, h, applyMultiplier, now)
	}

	return result
}

// IsOperator will check if the supplied pkh has operator permission (Administration or operator)
func IsOperator(ctx context.Context, ct *state.Contract, address bitcoin.RawAddress) bool {
	return ct.AdministrationAddress.Equal(address) || ct.OperatorAddress.Equal(address)
}

// ValidateVoting returns an error if voting is not allowed.
func ValidateVoting(ctx context.Context, ct *state.Contract, initiatorType uint32) error {
	if len(ct.VotingSystems) == 0 {
		return errors.New("No voting systems")
	}

	switch initiatorType {
	case 0: // Administration
		if !ct.AdministrationProposal {
			return errors.New("Administration proposals not allowed")
		}
	case 1: // Holder
		if !ct.HolderProposal {
			return errors.New("Holder proposals not allowed")
		}
	}

	return nil
}
