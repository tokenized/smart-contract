package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// VersionHandler exists to handle the Version command.
type UntrustedVersionHandler struct {
	state *data.UntrustedState
}

// NewVersionHandler returns a new VersionHandler with the given Config.
func NewUntrustedVersionHandler(state *data.UntrustedState) *UntrustedVersionHandler {
	result := UntrustedVersionHandler{state: state}
	return &result
}

// Verifies version message and sends acknowledge
func (handler *UntrustedVersionHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	_, ok := m.(*wire.MsgVersion)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgVersion")
	}

	handler.state.SetVersionReceived()

	// Return a version acknowledge
	// TODO Verify the version is compatible
	return []wire.Message{wire.NewMsgVerAck()}, nil
}
