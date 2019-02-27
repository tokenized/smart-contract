package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// PingHandler exists to handle the ping command.
type PingHandler struct{}

// NewPingHandler returns a new PingHandler with the given Config.
func NewPingHandler() *PingHandler {
	result := PingHandler{}
	return &result
}

// Handle implments the Handler interface.
func (h *PingHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgPing)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgPing")
	}

	pong := wire.MsgPong{Nonce: msg.Nonce}
	return []wire.Message{&pong}, nil
}
