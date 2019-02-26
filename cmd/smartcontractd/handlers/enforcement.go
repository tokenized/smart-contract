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

type Enforcement struct {
	MasterDB *db.DB
	Config   *node.Config
}

// Order handles an incoming Order request and prepares a Confiscation response
func (e *Enforcement) Order(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Freeze handles an incoming Freeze request and prepares a Confiscation response
func (e *Enforcement) Freeze(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Thaw handles an incoming Thaw request and prepares a Confiscation response
func (e *Enforcement) Thaw(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Confiscation handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) Confiscation(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Reconciliation handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) Reconciliation(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}
