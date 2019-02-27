package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

// HeadersHandler exists to handle the headers command.
type HeadersHandler struct {
	config    data.Config
	state     *data.State
	blocks    *storage.BlockRepository
	txs       *storage.TxRepository
	listeners []Listener
}

// NewHeadersHandler returns a new HeadersHandler with the given Config.
func NewHeadersHandler(config data.Config, state *data.State, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository, listeners []Listener) *HeadersHandler {
	result := HeadersHandler{
		config:    config,
		state:     state,
		blocks:    blockRepo,
		txs:       txRepo,
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

	lastHash := handler.state.LastHash()
	if lastHash == nil {
		lastHash = handler.blocks.LastHash()
	}

	if len(message.Headers) == 0 {
		// We must be in sync. This shouldn't happen since we request headers from one back from the top.
		handler.state.IsInSync = true
		handler.state.HeadersRequested = nil
		handler.blocks.Save(ctx) // Save when we get in sync
		return response, nil
	}

	if len(message.Headers) == 1 && message.Headers[0].BlockHash() == *lastHash {
		handler.state.IsInSync = true // We are in sync
		handler.state.HeadersRequested = nil
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
			request, err := handler.addHeader(ctx, &hash)
			if err != nil {
				return response, err
			}
			if request {
				// Request it if it isn't already requested.
				if offset := handler.state.AddBlockRequest(&hash); offset != -1 {
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

		// Check if we already have this block
		if handler.blocks.Contains(hash) || handler.state.BlockIsRequested(&hash) ||
			handler.state.BlockIsToBeRequested(&hash) {
			continue
		}

		// Check for a reorg
		reorgHeight, exists := handler.blocks.Height(&header.PrevBlock)
		if exists {
			// Call reorg listener for all blocks above reorg height.
			for height := handler.blocks.LastHeight(); height > reorgHeight; height-- {
				// Notify listeners
				if len(handler.listeners) > 0 {
					// Send block revert notification
					hash, err := handler.blocks.Hash(ctx, height)
					if err != nil {
						return response, errors.Wrap(err, "Failed to get reverted block hash")
					}
					blockMessage := BlockMessage{Hash: *hash, Height: height}
					for _, listener := range handler.listeners {
						listener.Handle(ctx, ListenerMsgBlockRevert, blockMessage)
					}

					// Notify of relevant txs in this block that are now reverted.
					revertTxs, err := handler.txs.GetBlock(ctx, height)
					if err != nil {
						return response, errors.Wrap(err, "Failed to get reverted txs")
					}
					if len(revertTxs) > 0 {
						for _, tx := range revertTxs {
							for _, listener := range handler.listeners {
								listener.Handle(ctx, ListenerMsgTxRevert, tx)
							}
						}
						if err := handler.txs.RemoveBlock(ctx, height); err != nil {
							return response, errors.Wrap(err, "Failed to remove reverted txs")
						}
					} else {
						if err := handler.txs.ReleaseBlock(ctx, height); err != nil {
							return response, errors.Wrap(err, "Failed to remove reverted txs")
						}
					}
				}
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
				return response, errors.New(fmt.Sprintf("Revert failed to produce correct last hash : %s", lastHash.String()))
			}

			// Add this header after the new top block
			request, err := handler.addHeader(ctx, &hash)
			if err != nil {
				return response, err
			}
			if request {
				// Request it if it isn't already requested.
				if offset := handler.state.AddBlockRequest(&hash); offset != -1 {
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
		return nil, nil //errors.New(fmt.Sprintf("Unknown header : %s", hash.String()))
	}

	// Add any non-full requests.
	if len(getBlocks.InvList) > 0 {
		response = append(response, getBlocks)
	}

	handler.state.HeadersRequested = nil
	return response, nil
}

func (handler HeadersHandler) addHeader(ctx context.Context, hash *chainhash.Hash) (bool, error) {
	if handler.state.StartHeight == -1 {
		// Check if it is the start block
		if handler.config.StartHash == *hash {
			handler.state.StartHeight = handler.blocks.LastHeight() + 1
		} else {
			err := handler.blocks.Add(ctx, *hash) // Just add hashes before the start block
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}

	return true, nil
}
