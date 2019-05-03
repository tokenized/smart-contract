package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// RejectHandler exists to handle the inv command.
type RejectHandler struct {
}

// NewRejectHandler returns a new RejectHandler
func NewRejectHandler() *RejectHandler {
	result := RejectHandler{}
	return &result
}

// Handle implements the Handler interface.
func (handler *RejectHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	_, ok := m.(*wire.MsgReject)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgReject")
	}

	return nil, nil
}
