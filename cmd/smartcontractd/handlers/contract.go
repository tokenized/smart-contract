package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

type Contract struct{}

func (c *Contract) Offer(ctx context.Context, log *log.Logger, itx *inspector.Transaction, rk *wallet.RootKey, m protocol.OpReturnMessage) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.IssuerCreate")
	defer span.End()

	// Validate and cast message to type
	msg, ok := m.(*protocol.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractOffer")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	log.Printf("%s : Received transaction: %+v %+v\n", v.TraceID, itx.Hash, msg)
	return nil
}

func (c *Contract) Formation(ctx context.Context, log *log.Logger, itx *inspector.Transaction, rk *wallet.RootKey, m protocol.OpReturnMessage) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.ContractUpdate")
	defer span.End()
	return nil
}

func (c *Contract) Amendment(ctx context.Context, log *log.Logger, itx *inspector.Transaction, rk *wallet.RootKey, m protocol.OpReturnMessage) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.IssuerUpdate")
	defer span.End()
	return nil
}
