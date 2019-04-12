package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/protocol"
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

	app.Handle("SEE", protocol.CodeContractOffer, c.OfferRequest)
	app.Handle("SEE", protocol.CodeContractAmendment, c.AmendmentRequest)
	app.Handle("SEE", protocol.CodeContractFormation, c.FormationResponse)
	// app.Handle("LOST", protocol.CodeContractAmendment, c.AmendmentReorg)
	// app.Handle("STOLE", protocol.CodeContractAmendment, c.AmendmentDoubleSpend)

	// Register asset based events.
	a := Asset{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeAssetDefinition, a.DefinitionRequest)
	app.Handle("SEE", protocol.CodeAssetModification, a.ModificationRequest)
	app.Handle("SEE", protocol.CodeAssetCreation, a.CreationResponse)

	// Register transfer based operations.
	t := Transfer{
		handler:   app,
		MasterDB:  masterDB,
		Config:    config,
		Headers:   headers,
		Tracer:    tracer,
		Scheduler: sch,
	}

	app.Handle("SEE", protocol.CodeTransfer, t.TransferRequest)
	app.Handle("SEE", protocol.CodeSettlement, t.SettlementResponse)
	app.Handle("REPROCESS", protocol.CodeTransfer, t.TransferTimeout)

	// Register enforcement based events.
	e := Enforcement{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeOrder, e.OrderRequest)
	app.Handle("SEE", protocol.CodeFreeze, e.FreezeResponse)
	app.Handle("SEE", protocol.CodeThaw, e.ThawResponse)
	app.Handle("SEE", protocol.CodeConfiscation, e.ConfiscationResponse)
	app.Handle("SEE", protocol.CodeReconciliation, e.ReconciliationResponse)

	// Register enforcement based events.
	g := Governance{
		handler:   app,
		MasterDB:  masterDB,
		Config:    config,
		Scheduler: sch,
	}

	app.Handle("SEE", protocol.CodeProposal, g.ProposalRequest)
	app.Handle("SEE", protocol.CodeVote, g.VoteResponse)
	app.Handle("SEE", protocol.CodeBallotCast, g.BallotCastRequest)
	app.Handle("SEE", protocol.CodeBallotCounted, g.BallotCountedResponse)
	app.Handle("SEE", protocol.CodeResult, g.ResultResponse)
	app.Handle("REPROCESS", protocol.CodeVote, g.FinalizeVote)

	// Register message based operations.
	m := Message{
		MasterDB:  masterDB,
		Config:    config,
		Headers:   headers,
		Tracer:    tracer,
		Scheduler: sch,
		UTXOs:     utxos,
	}

	app.Handle("SEE", protocol.CodeMessage, m.ProcessMessage)
	app.Handle("SEE", protocol.CodeRejection, m.ProcessRejection)

	app.HandleDefault("LOST", m.ProcessRevert)
	app.HandleDefault("STOLE", m.ProcessRevert)

	return app, nil
}
