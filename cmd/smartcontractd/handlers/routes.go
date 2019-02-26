package handlers

import (
	"log"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// API returns a handler for a set of routes for protocol actions.
func API(log *log.Logger, masterWallet wallet.WalletInterface, config *node.Config, masterDB *db.DB) protomux.Handler {

	app := node.New(log, masterWallet)

	// Register contract based events.
	c := Contract{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeContractOffer, c.Offer)
	app.Handle("SEE", protocol.CodeContractFormation, c.Formation)
	app.Handle("SEE", protocol.CodeContractAmendment, c.Amendment)
	// app.Handle("LOST", protocol.CodeContractAmendment, c.AmendmentReorg)
	// app.Handle("STOLE", protocol.CodeContractAmendment, c.AmendmentDoubleSpend)

	// Register asset based events.
	a := Asset{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeAssetDefinition, a.Definition)
	app.Handle("SEE", protocol.CodeAssetCreation, a.Creation)
	app.Handle("SEE", protocol.CodeAssetModification, a.Modification)

	// Register transfer based operations.
	t := Transfer{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeSend, t.Send)
	app.Handle("SEE", protocol.CodeExchange, t.Exchange)
	app.Handle("SEE", protocol.CodeSwap, t.Swap)
	app.Handle("SEE", protocol.CodeSettlement, t.Settlement)

	// Register enforcement based events.
	e := Enforcement{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeOrder, e.Order)
	app.Handle("SEE", protocol.CodeFreeze, e.Freeze)
	app.Handle("SEE", protocol.CodeThaw, e.Thaw)
	app.Handle("SEE", protocol.CodeConfiscation, e.Confiscation)
	app.Handle("SEE", protocol.CodeReconciliation, e.Reconciliation)

	// Register enforcement based events.
	g := Governance{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeInitiative, g.Initiative)
	app.Handle("SEE", protocol.CodeReferendum, g.Referendum)
	app.Handle("SEE", protocol.CodeVote, g.Vote)
	app.Handle("SEE", protocol.CodeBallotCast, g.BallotCast)
	app.Handle("SEE", protocol.CodeBallotCounted, g.BallotCounted)
	app.Handle("SEE", protocol.CodeResult, g.Result)

	return app
}
