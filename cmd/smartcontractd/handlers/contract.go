package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
}

// OfferRequest handles an incoming Contract Offer and prepares a Formation response
func (c *Contract) OfferRequest(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
		logger.Log(ctx, logger.Warn, "%s : Contract already exists: %s\n", v.TraceID, contractAddr.String())
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeContractExists)
	}

	logger.Log(ctx, logger.Verbose, "%s : Accepting contract offer (%s) : %s\n", v.TraceID, msg.ContractName, contractAddr.String())

	// Contract Formation <- Contract Offer
	cf := protocol.NewContractFormation()
	cf.Version = msg.Version
	cf.ContractName = msg.ContractName
	cf.ContractFileHash = msg.ContractFileHash
	cf.GoverningLaw = msg.GoverningLaw
	cf.Jurisdiction = msg.Jurisdiction
	cf.ContractExpiration = msg.ContractExpiration
	cf.URI = msg.URI
	cf.ContractRevision = 0
	cf.IssuerID = msg.IssuerID
	cf.IssuerType = msg.IssuerType
	cf.ContractOperatorID = msg.ContractOperatorID
	cf.AuthorizationFlags = msg.AuthorizationFlags
	cf.VotingSystem = msg.VotingSystem
	cf.InitiativeThreshold = msg.InitiativeThreshold
	cf.InitiativeThresholdCurrency = msg.InitiativeThresholdCurrency
	cf.RestrictedQty = msg.RestrictedQty

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
	if fee := node.OutputFee(ctx, log, c.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, log, mux, itx, rk, &cf, outs)
}

// AmendmentRequest handles an incoming Contract Amendment and prepares a Formation response
func (c *Contract) AmendmentRequest(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
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
		logger.Log(ctx, logger.Warn, "%s : Contract not found: %s\n", v.TraceID, contractAddr.String())
		return node.ErrNoResponse
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if ct.Qty > 0 && int(msg.RestrictedQty) < len(ct.Assets) {
		logger.Log(ctx, logger.Warn, "%s : Cannot reduce allowable assets below existing number: %s\n", v.TraceID, contractAddr.String())
		return node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeContractQtyReduction)
	}

	// Bump the revision
	newRevision := ct.Revision + 1

	// Contract Formation <- Contract Amendment
	cf := protocol.NewContractFormation()
	cf.Version = msg.Version
	cf.ContractName = msg.ContractName
	cf.ContractFileHash = msg.ContractFileHash
	cf.GoverningLaw = msg.GoverningLaw
	cf.Jurisdiction = msg.Jurisdiction
	cf.ContractExpiration = msg.ContractExpiration
	cf.URI = msg.URI
	cf.ContractRevision = newRevision
	cf.IssuerID = msg.IssuerID
	cf.IssuerType = msg.IssuerType
	cf.ContractOperatorID = msg.ContractOperatorID
	cf.AuthorizationFlags = msg.AuthorizationFlags
	cf.VotingSystem = msg.VotingSystem
	cf.InitiativeThreshold = msg.InitiativeThreshold
	cf.InitiativeThresholdCurrency = msg.InitiativeThresholdCurrency
	cf.RestrictedQty = msg.RestrictedQty

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
	if fee := node.OutputFee(ctx, log, c.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, log, mux, itx, rk, &cf, outs)
}

// FormationResponse handles an outgoing Contract Formation and writes it to the state
func (c *Contract) FormationResponse(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Formation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractFormation)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractFormation")
	}

	dbConn := c.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := rk.Address
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr.String())
	if err != nil {
		logger.Log(ctx, logger.Warn, "%s : Failed to retrieve contract (%s) : %s\n", v.TraceID, msg.ContractName, err.Error())
		return err
	}

	// Create or update Contract
	if ct == nil {
		// Prepare creation object
		nc := contract.NewContract{
			IssuerAddress:               itx.Outputs[1].Address.String(), // Second output of formation tx
			ContractName:                string(msg.ContractName),
			ContractFileHash:            fmt.Sprintf("%x", msg.ContractFileHash),
			GoverningLaw:                string(msg.GoverningLaw),
			Jurisdiction:                string(msg.Jurisdiction),
			ContractExpiration:          msg.ContractExpiration,
			URI:                         string(msg.URI),
			IssuerID:                    string(msg.IssuerID),
			IssuerType:                  string(msg.IssuerType),
			ContractOperatorID:          string(msg.ContractOperatorID),
			AuthorizationFlags:          msg.AuthorizationFlags,
			VotingSystem:                string(msg.VotingSystem),
			InitiativeThreshold:         msg.InitiativeThreshold,
			InitiativeThresholdCurrency: string(msg.InitiativeThresholdCurrency),
			Qty:                         msg.RestrictedQty,
		}

		if err := contract.Create(ctx, dbConn, contractAddr.String(), &nc, v.Now); err != nil {
			logger.Log(ctx, logger.Warn, "%s : Failed to create contract (%s) : %s\n", v.TraceID, msg.ContractName, err.Error())
			return err
		}
		logger.Log(ctx, logger.Verbose, "%s : Created contract (%s) : %s\n", v.TraceID, msg.ContractName, contractAddr.String())
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		uc := contract.UpdateContract{
			ContractName:                stringPointer(string(msg.ContractName)),
			ContractFileHash:            stringPointer(fmt.Sprintf("%x", msg.ContractFileHash)),
			GoverningLaw:                stringPointer(string(msg.GoverningLaw)),
			Jurisdiction:                stringPointer(string(msg.Jurisdiction)),
			ContractExpiration:          &msg.ContractExpiration,
			URI:                         stringPointer(string(msg.URI)),
			Revision:                    &msg.ContractRevision,
			IssuerID:                    stringPointer(string(msg.IssuerID)),
			IssuerType:                  stringPointer(string(msg.IssuerType)),
			ContractOperatorID:          stringPointer(string(msg.ContractOperatorID)),
			AuthorizationFlags:          msg.AuthorizationFlags,
			VotingSystem:                stringPointer(string(msg.VotingSystem)),
			InitiativeThreshold:         &msg.InitiativeThreshold,
			InitiativeThresholdCurrency: stringPointer(string(msg.InitiativeThresholdCurrency)),
			Qty:                         &msg.RestrictedQty,
		}

		if err := contract.Update(ctx, dbConn, contractAddr.String(), &uc, v.Now); err != nil {
			logger.Log(ctx, logger.Warn, "%s : Failed contract update (%s) : %s\n", v.TraceID, msg.ContractName, err.Error())
			return err
		}
		logger.Log(ctx, logger.Verbose, "%s : Updated contract (%s) : %s\n", v.TraceID, msg.ContractName, contractAddr.String())
	}

	return nil
}
