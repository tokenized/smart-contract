package handlers

import (
	"context"

	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
)

// API returns a handler for a set of routes for protocol actions.
func API(
	ctx context.Context,
	masterWallet wallet.WalletInterface,
	config *node.Config,
	masterDB *db.DB,
	tracer *filters.Tracer,
	sch *scheduler.Scheduler,
	headers node.BitcoinHeaders,
	utxos *utxos.UTXOs,
	holdingsChannel *holdings.CacheChannel,
) (protomux.Handler, error) {

	app := node.New(config, masterDB, masterWallet)

	// Register contract based events.
	c := Contract{
		MasterDB: masterDB,
		Config:   config,
		Headers:  headers,
	}

	app.Handle("SEE", actions.CodeContractOffer, c.OfferRequest)
	app.Handle("SEE", actions.CodeContractAmendment, c.AmendmentRequest)
	app.Handle("SEE", actions.CodeContractFormation, c.FormationResponse)
	app.Handle("SEE", actions.CodeContractAddressChange, c.AddressChange)

	// Register agreement based events.
	agreement := Agreement{
		MasterDB: masterDB,
		Config:   config,
	}

	app.Handle("SEE", actions.CodeBodyOfAgreementOffer, agreement.OfferRequest)
	app.Handle("SEE", actions.CodeBodyOfAgreementAmendment, agreement.AmendmentRequest)
	app.Handle("SEE", actions.CodeBodyOfAgreementFormation, agreement.FormationResponse)

	// Register asset based events.
	a := Asset{
		MasterDB:        masterDB,
		Config:          config,
		HoldingsChannel: holdingsChannel,
	}

	app.Handle("SEE", actions.CodeAssetDefinition, a.DefinitionRequest)
	app.Handle("SEE", actions.CodeAssetModification, a.ModificationRequest)
	app.Handle("SEE", actions.CodeAssetCreation, a.CreationResponse)

	// Register transfer based operations.
	t := Transfer{
		handler:         app,
		MasterDB:        masterDB,
		Config:          config,
		Headers:         headers,
		Tracer:          tracer,
		Scheduler:       sch,
		HoldingsChannel: holdingsChannel,
	}

	app.Handle("SEE", actions.CodeTransfer, t.TransferRequest)
	app.Handle("SEE", actions.CodeSettlement, t.SettlementResponse)
	app.Handle("END", actions.CodeTransfer, t.TransferTimeout)

	// Register enforcement based events.
	e := Enforcement{
		MasterDB:        masterDB,
		Config:          config,
		HoldingsChannel: holdingsChannel,
	}

	app.Handle("SEE", actions.CodeOrder, e.OrderRequest)
	app.Handle("SEE", actions.CodeFreeze, e.FreezeResponse)
	app.Handle("SEE", actions.CodeThaw, e.ThawResponse)
	app.Handle("SEE", actions.CodeConfiscation, e.ConfiscationResponse)
	app.Handle("SEE", actions.CodeReconciliation, e.ReconciliationResponse)

	// Register enforcement based events.
	g := Governance{
		handler:   app,
		MasterDB:  masterDB,
		Config:    config,
		Scheduler: sch,
	}

	app.Handle("SEE", actions.CodeProposal, g.ProposalRequest)
	app.Handle("SEE", actions.CodeVote, g.VoteResponse)
	app.Handle("SEE", actions.CodeBallotCast, g.BallotCastRequest)
	app.Handle("SEE", actions.CodeBallotCounted, g.BallotCountedResponse)
	app.Handle("SEE", actions.CodeResult, g.ResultResponse)
	app.Handle("END", actions.CodeVote, g.FinalizeVote)

	// Register message based operations.
	m := Message{
		MasterDB:        masterDB,
		Config:          config,
		Headers:         headers,
		Tracer:          tracer,
		Scheduler:       sch,
		UTXOs:           utxos,
		HoldingsChannel: holdingsChannel,
	}

	app.Handle("SEE", actions.CodeMessage, m.ProcessMessage)
	app.Handle("SEE", actions.CodeRejection, m.ProcessRejection)

	app.Handle("LOST", protomux.ANY_EVENT, m.ProcessRevert)
	app.Handle("STOLE", protomux.ANY_EVENT, m.ProcessRevert)

	return app, nil
}
