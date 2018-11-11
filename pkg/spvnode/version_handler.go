package spvnode

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// VersionHandler exists to handle the Version command.
type VersionHandler struct {
	Config Config
}

// NewVersionHandler returns a new VersionHandler with the given Config.
func NewVersionHandler(config Config) VersionHandler {
	return VersionHandler{
		Config: config,
	}
}

// Handle implments the Handler interface
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h VersionHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	msg, ok := m.(*wire.MsgVersion)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgVersion")
	}

	return h.handle(ctx, msg)
}

// handle processes the MsgVersion, and responds with a MsgVersion.
//
// For now this just echos the version back in the response.
func (h VersionHandler) handle(ctx context.Context,
	m *wire.MsgVersion) ([]wire.Message, error) {

	out := wire.NewMsgVerAck()

	return []wire.Message{out}, nil
}
