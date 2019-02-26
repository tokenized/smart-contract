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

	return app
}
