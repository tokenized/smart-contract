package listeners

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// TXListener exists to handle the TX command.
type TXListener struct {
	Node    inspector.NodeInterface
	Handler protomux.Handler
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (h TXListener) Handle(ctx context.Context, m wire.Message) error {
	tx, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	// Check if transaction relates to protocol
	itx, err := inspector.NewTransactionFromWire(ctx, tx)
	if err != nil {
		return err
	}

	// Prefilter out non-protocol messages
	if !itx.IsTokenized() {
		return nil
	}

	// Promote TX
	if err := itx.Promote(ctx, h.Node); err != nil {
		return err
	}

	return h.Handler.Trigger(ctx, protomux.SEE, itx)
}
