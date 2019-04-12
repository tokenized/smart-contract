package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// HeadersHandler exists to handle the headers command.
type HeadersHandler struct {
	config    data.Config
	state     *data.State
	blocks    *storage.BlockRepository
	txs       *storage.TxRepository
	reorgs    *storage.ReorgRepository
	listeners []Listener
}

// NewHeadersHandler returns a new HeadersHandler with the given Config.
func NewHeadersHandler(config data.Config, state *data.State, blockRepo *storage.BlockRepository,
	txRepo *storage.TxRepository, reorgs *storage.ReorgRepository, listeners []Listener) *HeadersHandler {

	result := HeadersHandler{
		config:    config,
		state:     state,
		blocks:    blockRepo,
		txs:       txRepo,
		reorgs:    reorgs,
		listeners: listeners,
	}
	return &result
}

// Implements the Handler interface.
// Headers are in order from lowest block height, to highest
func (handler *HeadersHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	message, ok := m.(*wire.MsgHeaders)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	response := []wire.Message{}
	logger.Debug(ctx, "Received %d headers", len(message.Headers))

	lastHash := handler.state.LastHash()
	if lastHash == nil {
		lastHash = handler.blocks.LastHash()
	}

	if !handler.state.IsReady() && (len(message.Headers) == 0 || (len(message.Headers) == 1 && message.Headers[0].BlockHash() == *lastHash)) {
		logger.Info(ctx, "Headers in sync at height %d", handler.blocks.LastHeight())
		handler.state.SetPendingSync() // We are in sync
		if handler.state.StartHeight() == -1 {
			handler.state.SetInSync()
			logger.Error(ctx, "Headers in sync before start block found")
		} else if handler.state.BlockRequestsEmpty() {
			handler.state.SetInSync()
			logger.Info(ctx, "Blocks in sync at height %d", handler.blocks.LastHeight())
		}
		handler.state.ClearHeadersRequested()
		handler.blocks.Save(ctx) // Save when we get in sync
		return response, nil
	}

	// Process headers
	getBlocks := wire.NewMsgGetData()
	for _, header := range message.Headers {
		if len(header.PrevBlock) == 0 {
			continue
		}

		hash := header.BlockHash()

		if header.PrevBlock == *lastHash {
			request, err := handler.addHeader(ctx, header)
			if err != nil {
				return response, err
			}
			if request {
				// Request it if it isn't already requested.
				if handler.state.AddBlockRequest(&hash) {
					logger.Debug(ctx, "Requesting block : %s", hash)
					getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, &hash))
					if len(getBlocks.InvList) == wire.MaxInvPerMsg {
						// Start new get data (blocks) message
						response = append(response, getBlocks)
						getBlocks = wire.NewMsgGetData()
					}
				}
			}

			lastHash = &hash
			continue
		}

		if hash == *lastHash {
			continue
		}

		// Check if we already have this block
		if handler.blocks.Contains(hash) || handler.state.BlockIsRequested(&hash) ||
			handler.state.BlockIsToBeRequested(&hash) {
			continue
		}

		// Check for a reorg
		reorgHeight, exists := handler.blocks.Height(&header.PrevBlock)
		if exists {
			logger.Info(ctx, "Reorging to height %d", reorgHeight)
			handler.state.ClearInSync()

			reorg := storage.Reorg{
				BlockHeight: reorgHeight,
			}

			// Call reorg listener for all blocks above reorg height.
			for height := handler.blocks.LastHeight(); height > reorgHeight; height-- {
				// Add block to reorg
				header, err := handler.blocks.Header(ctx, height)
				if err != nil {
					return response, errors.Wrap(err, "Failed to get reverted block header")
				}
				reorgBlock := storage.ReorgBlock{
					Header: *header,
				}

				revertTxs, err := handler.txs.GetBlock(ctx, height)
				if err != nil {
					return response, errors.Wrap(err, "Failed to get reverted txs")
				}
				for _, tx := range revertTxs {
					reorgBlock.TxIds = append(reorgBlock.TxIds, tx)
				}

				reorg.Blocks = append(reorg.Blocks, reorgBlock)

				// Notify listeners
				if len(handler.listeners) > 0 {
					// Send block revert notification
					hash := header.BlockHash()
					blockMessage := BlockMessage{Hash: hash, Height: height}
					for _, listener := range handler.listeners {
						listener.HandleBlock(ctx, ListenerMsgBlockRevert, &blockMessage)
					}
					for _, tx := range revertTxs {
						for _, listener := range handler.listeners {
							listener.HandleTxState(ctx, ListenerMsgTxStateRevert, tx)
						}
					}
				}

				if len(revertTxs) > 0 {
					if err := handler.txs.RemoveBlock(ctx, height); err != nil {
						return response, errors.Wrap(err, "Failed to remove reverted txs")
					}
				} else {
					if err := handler.txs.ReleaseBlock(ctx, height); err != nil {
						return response, errors.Wrap(err, "Failed to remove reverted txs")
					}
				}
			}

			if err := handler.reorgs.Save(ctx, &reorg); err != nil {
				return response, err
			}

			// Revert block repository
			if err := handler.blocks.Revert(ctx, reorgHeight); err != nil {
				return response, err
			}

			// Assert this header is now next
			lastHash := handler.state.LastHash()
			if lastHash == nil {
				lastHash = handler.blocks.LastHash()
			}
			if lastHash == nil || header.PrevBlock != *lastHash {
				return response, errors.New(fmt.Sprintf("Revert failed to produce correct last hash : %s", lastHash))
			}

			// Add this header after the new top block
			request, err := handler.addHeader(ctx, header)
			if err != nil {
				return response, err
			}
			if request {
				// Request it if it isn't already requested.
				if handler.state.AddBlockRequest(&hash) {
					logger.Debug(ctx, "Requesting block : %s", hash)
					getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, &hash))
					if len(getBlocks.InvList) == wire.MaxInvPerMsg {
						// Start new get data (blocks) message
						response = append(response, getBlocks)
						getBlocks = wire.NewMsgGetData()
					}
				}
			}
			continue
		}

		// Ignore unknown blocks as they might happen when there is a reorg.
		return nil, nil //errors.New(fmt.Sprintf("Unknown header : %s", hash))
	}

	// Add any non-full requests.
	if len(getBlocks.InvList) > 0 {
		response = append(response, getBlocks)
	}

	handler.state.ClearHeadersRequested()
	return response, nil
}

func (handler HeadersHandler) addHeader(ctx context.Context, header *wire.BlockHeader) (bool, error) {
	startHeight := handler.state.StartHeight()
	if startHeight == -1 {
		// Check if it is the start block
		if handler.config.StartHash == header.BlockHash() {
			startHeight = handler.blocks.LastHeight() + 1
			handler.state.SetStartHeight(startHeight)
			logger.Verbose(ctx, "Found start block at height %d", startHeight)
		} else {
			err := handler.blocks.Add(ctx, header) // Just add hashes before the start block
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}

	return true, nil
}
