package spvnode

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/spvnode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// HeadersHandler exists to handle the Ping command.
type HeadersHandler struct {
	Config       Config
	BlockService *BlockService
}

// NewHeadersHandler returns a new HeadersHandler with the given Config.
func NewHeadersHandler(config Config,
	blockService *BlockService) HeadersHandler {

	return HeadersHandler{
		Config:       config,
		BlockService: blockService,
	}
}

// handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (h HeadersHandler) Handle(ctx context.Context,
	m wire.Message) ([]wire.Message, error) {

	in, ok := m.(*wire.MsgHeaders)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	return h.handle(ctx, in)
}

// handle processes the MsgHeaders.
//
// There are no responses for this, but new messages to send may be queued.
func (h HeadersHandler) handle(ctx context.Context,
	m *wire.MsgHeaders) ([]wire.Message, error) {

	if len(m.Headers) == 0 {
		return nil, nil
	}

	// headers are in order from lowest block height, to highest
	// fmt.Printf("headers m : len=%v : first=%v : last=%v\n",
	// 	len(m.Headers),
	// 	m.Headers[0].BlockHash(),
	// 	m.Headers[len(m.Headers)-1].BlockHash())

	var max Block

	outs := []wire.Message{}

	// loop over the headers in reverse order, so we start at the newest
	// block hash, and work towards the oldest.
	for _, header := range m.Headers {
		if len(header.PrevBlock) == 0 {
			continue
		}

		hash := header.BlockHash()

		previous, err := h.BlockService.Read(ctx, header.PrevBlock)
		if err != nil {
			continue
		}

		b := Block{
			Hash:      hash.String(),
			PrevBlock: header.PrevBlock.String(),
			Height:    previous.Height + 1,
		}

		if getdata := h.buildGetDataForBlock(ctx, hash); getdata != nil {
			outs = append(outs, getdata)
		}

		if err := h.BlockService.Write(ctx, b); err != nil {
			return nil, err
		}

		max = b
	}

	if max.Height == 0 {
		return nil, nil
	}

	if _, err := h.BlockService.LastSeen(ctx, max); err != nil {
		return nil, err
	}

	log := logger.NewLoggerFromContext(ctx).Sugar()
	log.Infof("Latest block hash=%v height=%v", max.Hash, max.Height)

	// prune the blocks map, we only need a few recent one
	if err := h.BlockService.prune(ctx, max.Height); err != nil {
		log := logger.NewLoggerFromContext(ctx).Sugar()
		log.Errorf("Failed to prune : %v", err)
	}

	if len(m.Headers) > 1 {
		// get more headers if we need them
		last := m.Headers[len(m.Headers)-1]
		lastHash := last.BlockHash()

		out := wire.NewMsgGetHeaders()
		out.BlockLocatorHashes = []*chainhash.Hash{
			&lastHash,
		}

		outs = append(outs, out)
	}

	return outs, nil
}

func (h HeadersHandler) buildGetDataForBlock(ctx context.Context,
	blockHash chainhash.Hash) *wire.MsgGetData {

	if !h.BlockService.synced || h.BlockService.HasBlock(ctx, blockHash) {
		return nil
	}

	getblock := wire.NewMsgGetData()
	iv := wire.NewInvVect(wire.InvTypeBlock, &blockHash)
	getblock.AddInvVect(iv)

	return getblock
}
