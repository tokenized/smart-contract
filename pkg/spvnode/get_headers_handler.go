package spvnode

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/spvnode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// GetHeadersHandler exists to handle the Ping command.
type GetHeadersHandler struct {
	Config       Config
	BlockService *BlockService
}

// NewGetHeadersHandler returns a new GetHeadersHandler with the given Config.
func NewGetHeadersHandler(config Config, bs *BlockService) GetHeadersHandler {
	return GetHeadersHandler{
		Config:       config,
		BlockService: bs,
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h GetHeadersHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	in, ok := m.(*wire.MsgGetHeaders)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	return h.handle(ctx, in)
}

// handle processes the MsgGetHeaders.
//
// There are no responses for this, but new messages to send may be queued.
func (h GetHeadersHandler) handle(ctx context.Context,
	m *wire.MsgGetHeaders) ([]wire.Message, error) {

	// loop over the hashes, which are ordered most recent to oldest. For
	// the first one that is found, send a getheaders message. If none are
	// found, send the last hash (genesis block).
	var mostRecent *chainhash.Hash

	for _, hash := range m.BlockLocatorHashes {
		if h.BlockService.HasBlock(ctx, *hash) {
			mostRecent = hash
			break
		}
	}

	if mostRecent == nil {
		// we have no blocks with matching hashes, so use the oldest, which
		// for an empty set will be the genesis block.
		mostRecent = m.BlockLocatorHashes[len(m.BlockLocatorHashes)-1]
	}

	block, ok := h.BlockService.Blocks[*mostRecent]
	if !ok {
		block = Block{
			Hash:   mostRecent.String(),
			Height: 0,
		}
	}

	log := logger.NewLoggerFromContext(ctx).Sugar()
	log.Infof("Starting from block hash=%v height=%v",
		block.Hash,
		block.Height)

	if err := h.BlockService.Write(ctx, block); err != nil {
		return nil, err
	}

	return nil, nil

	// out := wire.NewMsgGetHeaders()
	// out.BlockLocatorHashes = []*chainhash.Hash{
	// 	mostRecent,
	// }

	// return []wire.Message{out}, nil
}
