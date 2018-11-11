package spvnode

import (
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// InvHandler exists to handle the Ping command.
type InvHandler struct {
	Config Config
}

// NewInvHandler returns a new InvHandler with the given Config.
func NewInvHandler(config Config) InvHandler {
	return InvHandler{
		Config: config,
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h InvHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	in, ok := m.(*wire.MsgInv)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	return h.handle(ctx, in)
}

// handle processes the MsgInv.
//
// There are no responses for this, but new messages to send may be queued.
func (h InvHandler) handle(ctx context.Context,
	m *wire.MsgInv) ([]wire.Message, error) {

	messages := []wire.Message{}

	for _, v := range m.InvList {
		switch v.Type {
		case wire.InvTypeTx:
			out := wire.NewMsgGetData()
			out.AddInvVect(v)
			messages = append(messages, out)

		case wire.InvTypeBlock:
			out := wire.NewMsgGetData()
			out.AddInvVect(v)

			messages = append(messages, out)

		default:
			fmt.Printf("unhandled inv vector type = %v\n", v.Type)
		}
	}

	return messages, nil
}
