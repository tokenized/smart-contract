package spvnode

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// BlockHandler exists to handle the Ping command.
type BlockHandler struct {
	Config       Config
	BlockService *BlockService
	Listeners    []Listener
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(config Config, blockService *BlockService, listeners []Listener) BlockHandler {

	return BlockHandler{
		Config:       config,
		BlockService: blockService,
		Listeners:    listeners,
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h BlockHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	msg, ok := m.(*wire.MsgBlock)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgBlock")
	}

	return h.handle(ctx, msg)
}

// handle processes the MsgBlock.
//
// The node has new block available via the inventory.
func (h BlockHandler) handle(ctx context.Context,
	b *wire.MsgBlock) ([]wire.Message, error) {

	// if we already have this block, we don't need to ask for more
	if h.BlockService.HasBlock(ctx, b.BlockHash()) {
		return nil, nil
	}

	prevBlock, err := h.BlockService.Read(ctx, b.Header.PrevBlock)
	if err != nil {
		// TODO we don't have the block, so we should fetch it if this was
		// a "not found" error. Probably worth a full sync again.
		return nil, err
	}

	block := Block{
		Hash:      b.BlockHash().String(),
		PrevBlock: prevBlock.Hash,
		Height:    prevBlock.Height + 1,
	}

	// we haven't seen this block, store it
	if err = h.BlockService.Write(ctx, block); err != nil {
		return nil, err
	}

	// do we need to send the block to the listeners?
	if h.shouldNotify(block) && h.Listeners != nil {
		for _, listener := range h.Listeners {
			listener.Handle(ctx, b)
		}
	}

	// potenitally update te "last seen" block.
	if _, err := h.BlockService.LastSeen(ctx, block); err != nil {
		return nil, err
	}

	return nil, nil
}

func (h BlockHandler) shouldNotify(block Block) bool {
	if !h.BlockService.synced || h.BlockService.State == nil {
		return false
	}

	if block.Height > h.BlockService.State.LastSeen.Height {
		return true
	}

	return false
}
