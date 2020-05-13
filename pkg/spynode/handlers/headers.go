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
	config         data.Config
	state          *data.State
	txStateChannel *TxStateChannel
	blocks         *storage.BlockRepository
	txs            *storage.TxRepository
	reorgs         *storage.ReorgRepository
	listeners      []Listener
}

// NewHeadersHandler returns a new HeadersHandler with the given Config.
func NewHeadersHandler(config data.Config, state *data.State, txStateChannel *TxStateChannel,
	blockRepo *storage.BlockRepository, txRepo *storage.TxRepository,
	reorgs *storage.ReorgRepository, listeners []Listener) *HeadersHandler {

	result := HeadersHandler{
		config:         config,
		state:          state,
		txStateChannel: txStateChannel,
		blocks:         blockRepo,
		txs:            txRepo,
		reorgs:         reorgs,
		listeners:      listeners,
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

	if !handler.state.IsReady() && (len(message.Headers) == 0 || (len(message.Headers) == 1 &&
		lastHash.Equal(message.Headers[0].BlockHash()))) {

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

		if lastHash.Equal(&header.PrevBlock) {
			logger.Debug(ctx, "Header (next) : %s", hash.String())
			request, err := handler.addHeader(ctx, header)
			if err != nil {
				return response, err
			}
			if request { // block should be processed
				// Request it if it isn't already requested.
				sendRequest, err := handler.state.AddBlockRequest(&header.PrevBlock, hash)
				if err != nil {
					if err == data.ErrWrongPreviousHash {
						logger.Warn(ctx, "Wrong previous hash : %s", header.PrevBlock.String())
					}
				} else if sendRequest {
					logger.Debug(ctx, "Requesting block : %s", hash.String())
					getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, hash))
					if len(getBlocks.InvList) == wire.MaxInvPerMsg {
						// Start new get data (blocks) message
						response = append(response, getBlocks)
						getBlocks = wire.NewMsgGetData()
					}
				}
			} else {
				// A block is not requested if the header matching the start hash has not been
				//   found yet. The last hash must still be updated.
				handler.state.SetLastHash(*hash)
			}

			lastHash = *hash
			continue
		}

		logger.Debug(ctx, "Header (not next) : %s", hash.String())

		if hash.Equal(&lastHash) {
			continue // Already latest header
		}

		// Check if we already have this block
		if handler.blocks.Contains(hash) || handler.state.BlockIsRequested(hash) ||
			handler.state.BlockIsToBeRequested(hash) {
			continue
		}

		// Check for a reorg
		logger.Info(ctx, "Header previous block : %s", header.PrevBlock.String())
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
				revertHeader, err := handler.blocks.Header(ctx, height)
				if err != nil {
					return response, errors.Wrap(err, "Failed to get reverted block header")
				}

				reorgBlock := storage.ReorgBlock{
					Header: *revertHeader,
				}

				revertTxs, err := handler.txs.GetBlock(ctx, height)
				if err != nil {
					return response, errors.Wrap(err, "Failed to get reverted txs")
				}
				for _, txid := range revertTxs {
					reorgBlock.TxIds = append(reorgBlock.TxIds, txid)
				}

				reorg.Blocks = append(reorg.Blocks, reorgBlock)

				// Notify listeners
				if len(handler.listeners) > 0 {
					// Send block revert notification
					blockMessage := BlockMessage{
						Hash:   *revertHeader.BlockHash(),
						Height: height,
						Time:   revertHeader.Timestamp,
					}
					for _, listener := range handler.listeners {
						listener.HandleBlock(ctx, ListenerMsgBlockRevert, &blockMessage)
					}
					for _, txid := range revertTxs {
						handler.txStateChannel.Add(TxState{
							ListenerMsgTxStateRevert,
							txid,
						})
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
			newLastHash := handler.blocks.LastHash()
			if newLastHash == nil || !newLastHash.Equal(&header.PrevBlock) {
				return response, errors.New(fmt.Sprintf("Revert failed to produce correct last hash : %s", newLastHash))
			}
			handler.state.SetLastHash(*newLastHash)

			// Add this header after the new top block
			request, err := handler.addHeader(ctx, header)
			if err != nil {
				return response, err
			}
			if request {
				// Request it if it isn't already requested.
				sendRequest, err := handler.state.AddBlockRequest(&header.PrevBlock, hash)
				if err != nil {
					if err == data.ErrWrongPreviousHash {
						logger.Warn(ctx, "Wrong previous hash : %s", header.PrevBlock.String())
					}
				} else if sendRequest {
					logger.Debug(ctx, "Requesting block : %s", hash.String())
					getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, hash))
					if len(getBlocks.InvList) == wire.MaxInvPerMsg {
						// Start new get data (blocks) message
						response = append(response, getBlocks)
						getBlocks = wire.NewMsgGetData()
					}
				}
			}

			lastHash = *hash
			continue
		}

		// Ignore unknown blocks as they might happen when there is a reorg.
		logger.Debug(ctx, "Unknown header : %s", hash.String())
		logger.Debug(ctx, "Previous hash : %s", header.PrevBlock.String())
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
		if handler.config.StartHash.Equal(header.BlockHash()) {
			startHeight = handler.blocks.LastHeight() + 1
			handler.state.SetStartHeight(startHeight)
			handler.state.SetLastHash(header.PrevBlock)
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
