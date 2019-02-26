package handlers

import (
	"context"
	"log"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

type Governance struct {
	MasterDB *db.DB
	Config   *node.Config
}

// Initiative handles an incoming Initiative request and prepares a BallotCounted response
func (g *Governance) Initiative(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Referendum handles an incoming Referendum request and prepares a BallotCounted response
func (g *Governance) Referendum(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Vote handles an incoming Vote request and prepares a BallotCounted response
func (g *Governance) Vote(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// BallotCast handles an incoming BallotCast request and prepares a BallotCounted response
func (g *Governance) BallotCast(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// BallotCounted handles an outgoing BallotCounted action and writes it to the state
func (g *Governance) BallotCounted(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Result handles an outgoing Result action and writes it to the state
func (g *Governance) Result(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}
