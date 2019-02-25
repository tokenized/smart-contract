package handlers

import (
	"log"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// API returns a handler for a set of routes for protocol actions.
func API(log *log.Logger, config *node.Config, masterDB *db.DB) protomux.Handler {

	app := node.New(log)

	// Register block based events.
	c := Contract{}

	app.Handle("SEE", protocol.CodeContractOffer, c.IssuerCreate)
	app.Handle("SEE", protocol.CodeContractFormation, c.ContractUpdate)
	app.Handle("SEE", protocol.CodeContractAmendment, c.IssuerUpdate)
	// app.Handle("LOST", protocol.CodeContractAmendment, c.IssuerReorg)
	// app.Handle("STOLE", protocol.CodeContractAmendment, c.IssuerDoubleSpend)

	return app
}
