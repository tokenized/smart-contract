package spvnode

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// PingHandler exists to handle the Ping command.
type PingHandler struct {
	Config Config
}

// NewPingHandler returns a new PingHandler with the given Config.
func NewPingHandler(config Config) PingHandler {
	return PingHandler{
		Config: config,
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h PingHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	msg, ok := m.(*wire.MsgPing)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgPing")
	}

	return h.handle(ctx, msg)
}

// handle processes the MsgPing, and responds with a MsgPong.
func (h PingHandler) handle(ctx context.Context,
	m *wire.MsgPing) ([]wire.Message, error) {

	pong := wire.MsgPong{
		Nonce: m.Nonce,
	}

	return []wire.Message{&pong}, nil
}
