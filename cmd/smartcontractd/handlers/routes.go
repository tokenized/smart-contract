package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/specification/dist/golang/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// BitcoinHeaders provides functions for retrieving information about headers on the currently
//   longest chain.
type BitcoinHeaders interface {
	LastHeight(ctx context.Context) int
	Hash(ctx context.Context, height int) (*chainhash.Hash, error)
	Time(ctx context.Context, height int) (uint32, error)
}

// API returns a handler for a set of routes for protocol actions.
func API(ctx context.Context, masterWallet wallet.WalletInterface, config *node.Config, masterDB *db.DB,
	tracer *listeners.Tracer, sch *scheduler.Scheduler, headers BitcoinHeaders, utxos *utxos.UTXOs) (protomux.Handler, error) {

	app := node.New(config, masterWallet)

	// Register contract based events.
	c := Contract{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle(protomux.SEE, protocol.CodeContractOffer, c.OfferRequest)
	app.Handle(protomux.SEE, protocol.CodeContractAmendment, c.AmendmentRequest)
	app.Handle(protomux.SEE, protocol.CodeContractFormation, c.FormationResponse)
	// app.Handle(protomux.LOST, protocol.CodeContractAmendment, c.AmendmentReorg)
	// app.Handle(protomux.STOLE, protocol.CodeContractAmendment, c.AmendmentDoubleSpend)

	// Register asset based events.
	a := Asset{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle(protomux.SEE, protocol.CodeAssetDefinition, a.DefinitionRequest)
	app.Handle(protomux.SEE, protocol.CodeAssetModification, a.ModificationRequest)
	app.Handle(protomux.SEE, protocol.CodeAssetCreation, a.CreationResponse)

	// Register transfer based operations.
	t := Transfer{
		handler:   app,
		MasterDB:  masterDB,
		Config:    config,
		Headers:   headers,
		Tracer:    tracer,
		Scheduler: sch,
	}

	app.Handle(protomux.SEE, protocol.CodeTransfer, t.TransferRequest)
	app.Handle(protomux.SEE, protocol.CodeSettlement, t.SettlementResponse)
	app.Handle(protomux.REPROCESS, protocol.CodeTransfer, t.TransferTimeout)

	// Register enforcement based events.
	e := Enforcement{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle(protomux.SEE, protocol.CodeOrder, e.OrderRequest)
	app.Handle(protomux.SEE, protocol.CodeFreeze, e.FreezeResponse)
	app.Handle(protomux.SEE, protocol.CodeThaw, e.ThawResponse)
	app.Handle(protomux.SEE, protocol.CodeConfiscation, e.ConfiscationResponse)
	app.Handle(protomux.SEE, protocol.CodeReconciliation, e.ReconciliationResponse)

	// Register enforcement based events.
	g := Governance{
		handler:   app,
		MasterDB:  masterDB,
		Config:    config,
		Scheduler: sch,
	}

	app.Handle(protomux.SEE, protocol.CodeProposal, g.ProposalRequest)
	app.Handle(protomux.SEE, protocol.CodeVote, g.VoteResponse)
	app.Handle(protomux.SEE, protocol.CodeBallotCast, g.BallotCastRequest)
	app.Handle(protomux.SEE, protocol.CodeBallotCounted, g.BallotCountedResponse)
	app.Handle(protomux.SEE, protocol.CodeResult, g.ResultResponse)
	app.Handle(protomux.REPROCESS, protocol.CodeVote, g.FinalizeVote)

	// Register message based operations.
	m := Message{
		MasterDB:  masterDB,
		Config:    config,
		Headers:   headers,
		Tracer:    tracer,
		Scheduler: sch,
		UTXOs:     utxos,
	}

	app.Handle(protomux.SEE, protocol.CodeMessage, m.ProcessMessage)
	app.Handle(protomux.SEE, protocol.CodeRejection, m.ProcessRejection)

	app.HandleDefault(protomux.LOST, m.ProcessRevert)
	app.HandleDefault(protomux.STOLE, m.ProcessRevert)

	return app, nil
}
