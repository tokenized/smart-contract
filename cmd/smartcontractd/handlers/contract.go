package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
}

func (c *Contract) Offer(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.IssuerCreate")
	defer span.End()

	dbConn := c.MasterDB
	defer dbConn.Close()

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Contract Offer
	msg, ok := itx.MsgProto.(*protocol.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractOffer")
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, dbConn, rk.Address.String())
	if err != nil {
		return err
	}

	// The contract should not exist already
	if ct != nil {
		log.Printf("%s : Contract already exists: %+v %+v\n", v.TraceID, rk.Address, msg)
		node.RespondReject(ctx, log, mux, itx, rk, protocol.RejectionCodeContractExists)
		return node.ErrRejected
	}

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
		Address: rk.Address,
		Value:   c.Config.DustLimit,
	}, {
		Address: itx.Inputs[0].Address,
		Value:   c.Config.DustLimit,
		Change:  true,
	}}

	if fee := node.OutputFee(ctx, log, c.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a formation
	node.RespondSuccess(ctx, log, mux, itx, rk, &cf, outs)
	return nil
}

func (c *Contract) Formation(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.ContractUpdate")
	defer span.End()
	return nil
}

func (c *Contract) Amendment(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.IssuerUpdate")
	defer span.End()
	return nil
}
