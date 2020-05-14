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
	state         *data.State
	blockRefeeder *BlockRefeeder
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(state *data.State, blockRefeeder *BlockRefeeder) *BlockHandler {
	result := BlockHandler{
		state:         state,
		blockRefeeder: blockRefeeder,
	}
	return &result
}

// Handle implements the Handler interface for a block handler.
func (handler *BlockHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	block, ok := m.(*wire.MsgParseBlock)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgParseBlock")
	}

	receivedHash := block.Header.BlockHash()
	logger.Debug(ctx, "Received block : %s", receivedHash.String())

	if handler.blockRefeeder != nil && handler.blockRefeeder.SetBlock(*receivedHash, block) {
		return nil, nil
	}

	if !handler.state.AddBlock(receivedHash, block) {
		logger.Warn(ctx, "Block not requested : %s", receivedHash.String())
	}

	return nil, nil
}
