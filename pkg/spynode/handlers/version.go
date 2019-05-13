package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// VersionHandler exists to handle the Version command.
type VersionHandler struct {
	state   *data.State
	address string
}

// NewVersionHandler returns a new VersionHandler with the given Config.
func NewVersionHandler(state *data.State, address string) *VersionHandler {
	result := VersionHandler{state: state, address: address}
	return &result
}

// Verifies version message and sends acknowledge
func (handler *VersionHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgVersion)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgVersion")
	}

	logger.Verbose(ctx, "(%s) Version : %s protocol %d, blocks %d", handler.address, msg.UserAgent,
		msg.ProtocolVersion, msg.LastBlock)
	handler.state.SetVersionReceived()

	// Return a version acknowledge
	// TODO Verify the version is compatible
	return []wire.Message{wire.NewMsgVerAck()}, nil
}
