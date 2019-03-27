package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type InspectorTxCache interface {
	GetTx(context.Context, *chainhash.Hash) *inspector.Transaction
	SaveTx(context.Context, *inspector.Transaction) error
}

// API returns a handler for a set of routes for protocol actions.
func API(masterWallet wallet.WalletInterface, config *node.Config, masterDB *db.DB, txCache InspectorTxCache) protomux.Handler {

	app := node.New(config, masterWallet)

	// Register contract based events.
	c := Contract{
		MasterDB: masterDB,
		Config:   config,
		TxCache:  txCache,
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
		MasterDB: masterDB,
		Config:   config,
		TxCache:  txCache,
	}

	app.Handle("SEE", protocol.CodeTransfer, t.TransferRequest)
	app.Handle("SEE", protocol.CodeSettlement, t.SettlementResponse)

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
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", protocol.CodeProposal, g.ProposalRequest)
	app.Handle("SEE", protocol.CodeVote, g.VoteResponse)
	app.Handle("SEE", protocol.CodeBallotCast, g.BallotCastRequest)
	app.Handle("SEE", protocol.CodeBallotCounted, g.BallotCountedResponse)
	app.Handle("SEE", protocol.CodeResult, g.ResultResponse)

	// Register message based operations.
	m := Message{
		MasterDB: masterDB,
		Config:   config,
		TxCache:  txCache,
	}

	app.Handle("SEE", protocol.CodeMessage, m.ProcessMessage)

	return app
}
