package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// BlockHandler exists to handle the block command.
type BlockHandler struct {
	state *data.State
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(state *data.State) *BlockHandler {

	result := BlockHandler{
		state: state,
	}
	return &result
}

// Handle implements the Handler interface for a block handler.
func (handler *BlockHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	message, ok := m.(*wire.MsgBlock)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgBlock")
	}

	receivedHash := message.BlockHash()
	logger.Debug(ctx, "Received block : %s", receivedHash.String())
	if !handler.state.AddBlock(receivedHash, message) {
		logger.Warn(ctx, "Block not requested : %s", receivedHash.String())
	}

	return nil, nil
}
