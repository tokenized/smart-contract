package listeners

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// TXListener exists to handle the TX command.
type TXListener struct {
	Handler protomux.Handler
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (h TXListener) Handle(ctx context.Context, m wire.Message) error {
	msg, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	return h.Handler.Trigger(ctx, protomux.SEE, msg)
}
