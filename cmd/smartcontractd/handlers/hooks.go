package handlers

import (
	"log"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/pkg/spvnode"
)

// API returns a handler for a set of hooks.
func Register(log *log.Logger, sn *spvnode.Node, config *node.Config, masterDB *db.DB) {

	app := node.New(log, sn)

	// Register block based events.
	b := Block{}

	app.Handle(spvnode.ListenerBlock, b.VerifyBlockTransactions)

	// Register transaction based events.
	t := Transaction{}

	app.Handle(spvnode.ListenerTX, t.ProcessTransaction)
}
