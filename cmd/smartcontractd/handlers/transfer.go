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

type Transfer struct {
	MasterDB *db.DB
	Config   *node.Config
}

// Send handles an incoming Send request and prepares a Settlement response
func (t *Transfer) Send(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Exchange handles an incoming Exchange request and prepares a Settlement response
func (t *Transfer) Exchange(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Swap handles an incoming Swap request and prepares a Settlement response
func (t *Transfer) Swap(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// Settlement handles an outgoing Settlement action and writes it to the state
func (t *Transfer) Settlement(ctx context.Context, log *log.Logger, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}
