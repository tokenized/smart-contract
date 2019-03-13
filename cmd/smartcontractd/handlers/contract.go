package handlers

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
}

// OfferRequest handles an incoming Contract Offer and prepares a Formation response
func (c *Contract) OfferRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Offer")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractOffer")
	}

	dbConn := c.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		return err
	}

	// The contract should not exist already
	if ct != nil {
		logger.Warn(ctx, "%s : Contract already exists: %s", v.TraceID, contractAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeContractExists)
	}

	logger.Info(ctx, "%s : Accepting contract offer (%s) : %s", v.TraceID, msg.ContractName, contractAddr)

	// Contract Formation <- Contract Offer
	cf := protocol.NewContractFormation()

	err = platform.Convert(msg, cf)
	if err != nil {
		return err
	}

	cf.ContractRevision = 0
	cf.Timestamp = uint64(time.Now().UnixNano())

	// Build outputs
	// 1 - Contract Address
	// 2 - Issuer (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: contractAddr,
		Value:   c.Config.DustLimit,
	}, {
		Address: itx.Inputs[0].Address,
		Value:   c.Config.DustLimit,
		Change:  true,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, c.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, mux, itx, rk, &cf, outs)
}

// AmendmentRequest handles an incoming Contract Amendment and prepares a Formation response
func (c *Contract) AmendmentRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Amendment")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractAmendment)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractAmendment")
	}

	dbConn := c.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		return err
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr)
		return node.ErrNoResponse
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if ct.RestrictedQtyAssets > 0 && ct.RestrictedQtyAssets < uint64(len(ct.Assets)) {
		logger.Warn(ctx, "%s : Cannot reduce allowable assets below existing number: %s", v.TraceID, contractAddr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeContractQtyReduction)
	}

	if ct.Revision != uint64(msg.ContractRevision) {
		logger.Warn(ctx, "%s : Incorrect contract revision (%s) : specified %d != current %d", v.TraceID, ct.ContractName, msg.ContractRevision, ct.Revision)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeContractRevision)
	}

	// TODO Validate that changes are allowed. Check votes, ...
	// TODO Verify RefTxID data
	//msg.RefTxID               []byte

	logger.Info(ctx, "%s : Accepting contract amendment (%s) : %s", v.TraceID, ct.ContractName, contractAddr)

	// Contract Formation <- Contract Amendment
	cf := protocol.NewContractFormation()

	// Get current state
	err = platform.Convert(ct, cf)
	if err != nil {
		return err
	}

	// Apply modifications
	cf.ContractRevision = ct.Revision + 1 // Bump the revision
	cf.Timestamp = uint64(time.Now().UnixNano())

	// TODO Implement contract amendments
	// type Amendment struct {
	// FieldIndex    uint8
	// Element       uint16
	// SubfieldIndex uint8
	// DeleteElement bool
	// Data          []byte
	// }
	// for _, amendment := range msg.Amendments {
	// switch(amendment.FieldIndex) {
	// case 0: // ContractName               Nvarchar8
	// case 1: // ContractFileType           uint8
	// case 2: // LenContractFile            uint32
	// case 3: // ContractFile               []byte
	// case 4: // GoverningLaw               []byte
	// case 5: // Jurisdiction               []byte
	// case 6: // ContractExpiration         uint64
	// case 7: // ContractURI                Nvarchar8
	// case 8: // IssuerName                 Nvarchar8
	// case 9: // IssuerType                 byte
	// case 10: // IssuerLogoURL              Nvarchar8
	// case 11: // ContractOperatorID         Nvarchar8
	// case 12: // ContractAuthFlags          []byte
	// case 13: // VotingSystemCount          uint8
	// case 14: // VotingSystems              []VotingSystem
	// case 15: // RestrictedQtyAssets        uint64
	// case 16: // ReferendumProposal         bool
	// case 17: // InitiativeProposal         bool
	// case 18: // RegistryCount              uint8
	// case 19: // Registries                 []Registry
	// case 20: // IssuerAddress              bool
	// case 21: // UnitNumber                 Nvarchar8
	// case 22: // BuildingNumber             Nvarchar8
	// case 23: // Street                     Nvarchar16
	// case 24: // SuburbCity                 Nvarchar8
	// case 25: // TerritoryStateProvinceCode []byte
	// case 26: // CountryCode                []byte
	// case 27: // PostalZIPCode              Nvarchar8
	// case 28: // EmailAddress               Nvarchar8
	// case 29: // PhoneNumber                Nvarchar8
	// case 30: // KeyRolesCount              uint8
	// case 31: // KeyRoles                   []KeyRole
	// case 32: // NotableRolesCount          uint8
	// case 33: // NotableRoles               []NotableRole
	// default:
	// logger.Warn(ctx, "%s : Incorrect contract amendment field offset (%s) : %d", v.TraceID, ct.ContractName, amendment.FieldIndex)
	// return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeContractMalformedAmendment)
	// }
	// }

	// Update counts
	cf.LenContractFile = uint32(len(ct.ContractFile))
	cf.VotingSystemCount = uint8(len(ct.VotingSystems))
	cf.RegistryCount = uint8(len(ct.Registries))
	cf.KeyRolesCount = uint8(len(ct.KeyRoles))
	cf.NotableRolesCount = uint8(len(ct.NotableRoles))

	// Build outputs
	// 1 - Contract Address
	// 2 - Issuer (Change)
	// 3 - Fee
	outs := []node.Output{{
		Address: contractAddr,
		Value:   c.Config.DustLimit,
	}}

	// Issuer change. New issuer in second input
	if msg.ChangeIssuerAddress {
		if len(itx.Inputs) < 2 {
			logger.Warn(ctx, "%s : New issuer specified but not included in inputs (%s)", v.TraceID, ct.ContractName)
			return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeContractMissingNewIssuer)
		}

		outs = append(outs, node.Output{
			Address: itx.Inputs[1].Address,
			Value:   c.Config.DustLimit,
			Change:  true,
		})
	} else {
		outs = append(outs, node.Output{
			Address: itx.Inputs[0].Address,
			Value:   c.Config.DustLimit,
			Change:  true,
		})
	}

	// TODO Operator changes
	// if msg.ChangeOperatorAddress {

	// }

	// Add fee output
	if fee := node.OutputFee(ctx, c.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, mux, itx, rk, &cf, outs)
}

// FormationResponse handles an outgoing Contract Formation and writes it to the state
func (c *Contract) FormationResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Formation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractFormation)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractFormation")
	}

	dbConn := c.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract. Sender is verified to be contract before this response function is called.
	contractAddr := rk.Address
	contractName := msg.ContractName.String()
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve contract (%s) : %s", v.TraceID, contractName, err.Error())
		return err
	}

	// Create or update Contract
	if ct == nil {
		// Prepare creation object
		var nc contract.NewContract
		err := platform.Convert(msg, &nc)
		if err != nil {
			logger.Warn(ctx, "%s : Failed to convert formation to new contract (%s) : %s", v.TraceID, contractName, err.Error())
			return err
		}

		nc.Issuer = itx.Outputs[1].Address.String() // Second output of formation tx
		// nc.Operator =  // TODO How do we determine if an operator is specified?

		if err := contract.Create(ctx, dbConn, contractAddr.String(), &nc, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to create contract (%s) : %s", v.TraceID, contractName, err.Error())
			return err
		}
		logger.Info(ctx, "%s : Created contract (%s) : %s", v.TraceID, contractName, contractAddr)
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		uc := contract.UpdateContract{}

		if ct.Issuer != itx.Outputs[1].Address.String() { // Second output of formation tx
			uc.Issuer = stringPointer(itx.Outputs[1].Address.String())
			logger.Info(ctx, "%s : Updating contract issuer address (%s) : %s", v.TraceID, ct.ContractName, itx.Outputs[1].Address.String())
		}

		// TODO Update operator address - OperatorAddress *string

		if ct.ContractName != msg.ContractName.String() {
			uc.ContractName = stringPointer(msg.ContractName.String())
			logger.Info(ctx, "%s : Updating contract name (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractName)
		}

		if ct.ContractFileType != msg.ContractFileType {
			uc.ContractFileType = &msg.ContractFileType
			logger.Info(ctx, "%s : Updating contract file type (%s) : %02x", v.TraceID, ct.ContractName, msg.ContractFileType)
		}

		if !bytes.Equal(ct.ContractFile, msg.ContractFile) {
			uc.ContractFile = &msg.ContractFile
			logger.Info(ctx, "%s : Updating contract file (%s)", v.TraceID, ct.ContractName)
		}

		if ct.GoverningLaw != string(msg.GoverningLaw) {
			uc.GoverningLaw = stringPointer(string(msg.GoverningLaw))
			logger.Info(ctx, "%s : Updating contract governing law (%s) : %s", v.TraceID, ct.ContractName, *uc.GoverningLaw)
		}

		if ct.Jurisdiction != string(msg.Jurisdiction) {
			uc.Jurisdiction = stringPointer(string(msg.Jurisdiction))
			logger.Info(ctx, "%s : Updating contract jurisdiction (%s) : %s", v.TraceID, ct.ContractName, *uc.Jurisdiction)
		}

		if ct.ContractExpiration != msg.ContractExpiration {
			uc.ContractExpiration = &msg.ContractExpiration
			newExpiration := time.Unix(int64(msg.ContractExpiration), 0)
			logger.Info(ctx, "%s : Updating contract expiration (%s) : %s", v.TraceID, ct.ContractName, newExpiration.Format(time.UnixDate))
		}

		if ct.ContractURI != msg.ContractURI.String() {
			uc.ContractURI = stringPointer(msg.ContractURI.String())
			logger.Info(ctx, "%s : Updating contract URI (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractURI)
		}

		if ct.IssuerName != msg.IssuerName.String() {
			uc.IssuerName = stringPointer(msg.IssuerName.String())
			logger.Info(ctx, "%s : Updating contract issuer name (%s) : %s", v.TraceID, ct.ContractName, *uc.IssuerName)
		}

		if ct.IssuerType != msg.IssuerType {
			uc.IssuerType = &msg.IssuerType
			logger.Info(ctx, "%s : Updating contract issuer type (%s) : %02x", v.TraceID, ct.ContractName, *uc.IssuerType)
		}

		if ct.IssuerLogoURL != msg.IssuerLogoURL.String() {
			uc.IssuerLogoURL = stringPointer(msg.IssuerLogoURL.String())
			logger.Info(ctx, "%s : Updating contract issuer logo URL (%s) : %s", v.TraceID, ct.ContractName, *uc.IssuerLogoURL)
		}

		if ct.ContractOperatorID != msg.ContractOperatorID.String() {
			uc.ContractOperatorID = stringPointer(msg.ContractOperatorID.String())
			logger.Info(ctx, "%s : Updating contract operator ID (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractOperatorID)
		}

		if !bytes.Equal(ct.ContractAuthFlags, msg.ContractAuthFlags) {
			uc.ContractAuthFlags = &msg.ContractAuthFlags
			logger.Info(ctx, "%s : Updating contract auth flags (%s) : %v", v.TraceID, ct.ContractName, *uc.ContractAuthFlags)
		}

		if ct.RestrictedQtyAssets != msg.RestrictedQtyAssets {
			uc.RestrictedQtyAssets = &msg.RestrictedQtyAssets
			logger.Info(ctx, "%s : Updating contract restricted quantity assets (%s) : %d", v.TraceID, ct.ContractName, *uc.RestrictedQtyAssets)
		}

		if ct.ReferendumProposal != msg.ReferendumProposal {
			uc.ReferendumProposal = &msg.ReferendumProposal
			logger.Info(ctx, "%s : Updating contract referendum proposal (%s) : %t", v.TraceID, ct.ContractName, *uc.ReferendumProposal)
		}

		if ct.InitiativeProposal != msg.InitiativeProposal {
			uc.InitiativeProposal = &msg.InitiativeProposal
			logger.Info(ctx, "%s : Updating contract initiative proposal (%s) : %t", v.TraceID, ct.ContractName, *uc.InitiativeProposal)
		}

		if ct.UnitNumber != msg.UnitNumber.String() {
			uc.UnitNumber = stringPointer(msg.UnitNumber.String())
			logger.Info(ctx, "%s : Updating contract unit number (%s) : %s", v.TraceID, ct.ContractName, *uc.UnitNumber)
		}

		if ct.BuildingNumber != msg.BuildingNumber.String() {
			uc.BuildingNumber = stringPointer(msg.BuildingNumber.String())
			logger.Info(ctx, "%s : Updating contract building number (%s) : %s", v.TraceID, ct.ContractName, *uc.BuildingNumber)
		}

		if ct.Street != msg.Street.String() {
			uc.Street = stringPointer(msg.Street.String())
			logger.Info(ctx, "%s : Updating contract street (%s) : %s", v.TraceID, ct.ContractName, *uc.Street)
		}

		if ct.SuburbCity != msg.SuburbCity.String() {
			uc.SuburbCity = stringPointer(msg.SuburbCity.String())
			logger.Info(ctx, "%s : Updating contract city (%s) : %s", v.TraceID, ct.ContractName, *uc.SuburbCity)
		}

		if ct.TerritoryStateProvinceCode != string(msg.TerritoryStateProvinceCode) {
			uc.TerritoryStateProvinceCode = stringPointer(string(msg.TerritoryStateProvinceCode))
			logger.Info(ctx, "%s : Updating contract state (%s) : %s", v.TraceID, ct.ContractName, *uc.TerritoryStateProvinceCode)
		}

		if ct.CountryCode != string(msg.CountryCode) {
			uc.CountryCode = stringPointer(string(msg.CountryCode))
			logger.Info(ctx, "%s : Updating contract country (%s) : %s", v.TraceID, ct.ContractName, *uc.CountryCode)
		}

		if ct.PostalZIPCode != msg.PostalZIPCode.String() {
			uc.PostalZIPCode = stringPointer(msg.PostalZIPCode.String())
			logger.Info(ctx, "%s : Updating contract postal code (%s) : %s", v.TraceID, ct.ContractName, *uc.PostalZIPCode)
		}

		if ct.EmailAddress != msg.EmailAddress.String() {
			uc.EmailAddress = stringPointer(msg.EmailAddress.String())
			logger.Info(ctx, "%s : Updating contract email (%s) : %s", v.TraceID, ct.ContractName, *uc.EmailAddress)
		}

		if ct.PhoneNumber != msg.PhoneNumber.String() {
			uc.PhoneNumber = stringPointer(msg.PhoneNumber.String())
			logger.Info(ctx, "%s : Updating contract phone (%s) : %s", v.TraceID, ct.ContractName, *uc.PhoneNumber)
		}

		// Check if key roles are different
		different := len(ct.KeyRoles) != len(msg.KeyRoles)
		if !different {
			for i, keyRole := range ct.KeyRoles {
				if keyRole.Type != msg.KeyRoles[i].Type || keyRole.Name != msg.KeyRoles[i].Name.String() {
					different = true
					break
				}
			}
		}

		if different {
			newKeyRoles := make([]state.KeyRole, 0, len(msg.KeyRoles))
			for _, keyRole := range msg.KeyRoles {
				var newKeyRole state.KeyRole
				err := platform.Convert(keyRole, &newKeyRole)
				if err != nil {
					return err
				}
				newKeyRoles = append(newKeyRoles, newKeyRole)
			}
			uc.KeyRoles = &newKeyRoles
		}

		// Check if notable roles are different
		different = len(ct.NotableRoles) != len(msg.NotableRoles)
		if !different {
			for i, notableRole := range ct.NotableRoles {
				if notableRole.Type != msg.NotableRoles[i].Type || notableRole.Name != msg.NotableRoles[i].Name.String() {
					different = true
					break
				}
			}
		}

		if different {
			newNotableRoles := make([]state.NotableRole, 0, len(msg.NotableRoles))
			for _, notableRole := range msg.NotableRoles {
				var newNotableRole state.NotableRole
				err := platform.Convert(notableRole, &newNotableRole)
				if err != nil {
					return err
				}
				newNotableRoles = append(newNotableRoles, newNotableRole)
			}
			uc.NotableRoles = &newNotableRoles
		}

		// Check if registries are different
		different = len(ct.Registries) != len(msg.Registries)
		if !different {
			for i, registry := range ct.Registries {
				if registry.Name != msg.Registries[i].Name.String() ||
					registry.URL != msg.Registries[i].URL.String() ||
					registry.PublicKey != msg.Registries[i].PublicKey.String() {
					different = true
					break
				}
			}
		}

		if different {
			newRegistries := make([]state.Registry, 0, len(msg.Registries))
			for _, registry := range msg.Registries {
				var newRegistry state.Registry
				err := platform.Convert(registry, &newRegistry)
				if err != nil {
					return err
				}
				newRegistries = append(newRegistries, newRegistry)
			}
			uc.Registries = &newRegistries
		}

		// Check if voting systems are different
		different = len(ct.VotingSystems) != len(msg.VotingSystems)
		if !different {
			for i, votingSystem := range ct.VotingSystems {
				if votingSystem.Name != msg.VotingSystems[i].Name.String() {
					different = true
					break
				}

				if !bytes.Equal(votingSystem.System, msg.VotingSystems[i].System) {
					different = true
					break
				}

				if votingSystem.Method != msg.VotingSystems[i].Method {
					different = true
					break
				}

				if votingSystem.Logic != msg.VotingSystems[i].Logic {
					different = true
					break
				}

				if votingSystem.ThresholdPercentage != msg.VotingSystems[i].ThresholdPercentage {
					different = true
					break
				}

				if votingSystem.VoteMultiplierPermitted != msg.VotingSystems[i].VoteMultiplierPermitted {
					different = true
					break
				}

				if votingSystem.InitiativeThreshold != msg.VotingSystems[i].InitiativeThreshold {
					different = true
					break
				}

				if !bytes.Equal(votingSystem.InitiativeThresholdCurrency, msg.VotingSystems[i].InitiativeThresholdCurrency) {
					different = true
					break
				}
			}
		}

		if different {
			newVotingSystems := make([]state.VotingSystem, 0, len(msg.VotingSystems))
			for _, votingSystem := range msg.VotingSystems {
				var newVotingSystem state.VotingSystem
				err := platform.Convert(votingSystem, &newVotingSystem)
				if err != nil {
					return err
				}
				newVotingSystems = append(newVotingSystems, newVotingSystem)
			}
			uc.VotingSystems = &newVotingSystems
		}

		if err := contract.Update(ctx, dbConn, contractAddr.String(), &uc, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed contract update (%s) : %s", v.TraceID, msg.ContractName, err.Error())
			return err
		}
		logger.Info(ctx, "%s : Updated contract (%s) : %s", v.TraceID, msg.ContractName, contractAddr)
	}

	return nil
}
