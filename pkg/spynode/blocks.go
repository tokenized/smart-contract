package spynode

import (
	"context"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

var (
	ErrBlockNotNextBlock = errors.New("Not next block")
	ErrBlockNotAdded     = errors.New("Block not added")
)

func (node *Node) processBlocks(ctx context.Context) error {

	for !node.isStopping() {

		block, height, refeederActive := node.blockRefeeder.GetBlock()
		if refeederActive {
			if block != nil {
				node.provideBlock(ctx, block, height)

				if height != 0 { // block header still active
					if node.blocks.LastHeight() == height {
						logger.Info(ctx, "Refeed complete at block %d", height)
						node.blockRefeeder.Clear(height)
					} else {
						logger.Info(ctx, "Refeed setting next block %d", height+1)
						nextHash, err := node.blocks.Hash(ctx, height+1)
						if err != nil {
							return errors.Wrap(err, "get next hash")
						}
						node.blockRefeeder.Increment(height+1, *nextHash)
					}
				}
			}

			hash := node.blockRefeeder.GetBlockToRequest()
			if hash != nil {
				getBlocks := wire.NewMsgGetData()
				getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, hash))
				if !node.queueOutgoing(getBlocks) {
					return nil
				}
			}

			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Blocks are fed into the state when received by the block handler, then pulled out and
		//   processed here.
		block = node.state.NextBlock()
		if block == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if err := node.ProcessBlock(ctx, block); err != nil {
			c := errors.Cause(err)
			if c != ErrBlockNotNextBlock && c != ErrBlockNotAdded {
				header := block.GetHeader()
				logger.Warn(ctx, "Failed to process block : %s : %s",
					header.BlockHash().String(), err)
				return err
			}
		}

		// Request more blocks if necessary
		// TODO Send some requests to other nodes --ce
		getBlocks := wire.NewMsgGetData() // Block request message

		for {
			requestHash, _ := node.state.GetNextBlockToRequest()
			if requestHash == nil {
				break
			}

			logger.Debug(ctx, "Requesting block : %s", requestHash.String())
			getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, requestHash))
			if len(getBlocks.InvList) == wire.MaxInvPerMsg {
				// Start new get data (block request) message
				if !node.queueOutgoing(getBlocks) {
					return nil
				}
				getBlocks = wire.NewMsgGetData()
			}
		}

		// Add any non-full requests.
		if len(getBlocks.InvList) > 0 {
			if !node.queueOutgoing(getBlocks) {
				return nil
			}
		}
	}

	return nil
}

// provideBlock feeds the block to the listeners.
func (node *Node) provideBlock(ctx context.Context, block wire.Block, height int) error {
	header := block.GetHeader()
	hash := header.BlockHash()
	blockMessage := handlers.BlockMessage{Hash: *hash, Height: height, Time: header.Timestamp}
	for _, listener := range node.listeners {
		listener.HandleBlock(ctx, handlers.ListenerMsgBlock, &blockMessage)
	}

	logger.Debug(ctx, "Providing block %d (%d tx) : %s", height, block.GetTxCount(), hash.String())
	for {
		tx, err := block.GetNextTx()
		if err != nil {
			return errors.Wrap(err, "get next tx")
		}
		if tx == nil {
			break // parsed all txs
		}

		node.confTxChannel.Add(handlers.TxData{
			Msg:             tx,
			ConfirmedHeight: height,
		})
	}

	return nil
}

func (node *Node) ProcessBlock(ctx context.Context, block wire.Block) error {
	node.blockLock.Lock()
	defer node.blockLock.Unlock()

	header := block.GetHeader()
	hash := header.BlockHash()
	logger.Debug(ctx, "Block : %s", hash.String())

	if node.blocks.Contains(hash) {
		height, _ := node.blocks.Height(hash)
		logger.Warn(ctx, "Already have block (%d) : %s", height, hash.String())
		node.state.FinalizeBlock(*hash)
		return ErrBlockNotAdded
	}

	if header.PrevBlock != *node.blocks.LastHash() {
		// Ignore this as it can happen when there is a reorg.
		logger.Warn(ctx, "Not next block : %s", hash.String())
		logger.Warn(ctx, "Previous hash : %s", header.PrevBlock.String())
		node.state.FinalizeBlock(*hash)
		return ErrBlockNotNextBlock // Unknown or out of order block
	}

	// Validate
	if !block.IsMerkleRootValid() {
		logger.Warn(ctx, "Invalid merkle hash for block %s", hash.String())
		node.state.FinalizeBlock(*hash)
		return ErrBlockNotAdded
	}

	// Add to repo
	if err := node.blocks.Add(ctx, &header); err != nil {
		node.state.FinalizeBlock(*hash)
		return errors.Wrap(err, "add block")
	}

	// Remove from requested blocks
	if err := node.state.FinalizeBlock(*hash); err != nil {
		return errors.Wrap(err, "finialize block")
	}

	// If we are in sync we can save after every block
	if node.state.IsReady() {
		if err := node.blocks.Save(ctx); err != nil {
			return errors.Wrap(err, "save blocks")
		}
	}

	// Get unconfirmed "relevant" txs
	var unconfirmed []bitcoin.Hash32
	var err error
	// This locks the tx repo so that propagated txs don't interfere while a block is being
	//   processed.
	unconfirmed, err = node.txs.GetUnconfirmed(ctx)
	if err != nil {
		return errors.Wrap(err, "get unconfirmed txs")
	}

	// Send block notification
	height := node.blocks.LastHeight()
	blockMessage := handlers.BlockMessage{Hash: *hash, Height: height, Time: header.Timestamp}
	for _, listener := range node.listeners {
		listener.HandleBlock(ctx, handlers.ListenerMsgBlock, &blockMessage)
	}

	// Notify Tx for block and tx listeners
	logger.Debug(ctx, "Processing block %d (%d tx) : %s", height, block.GetTxCount(), hash)
	inUnconfirmed := false
	txids := make([]*bitcoin.Hash32, 0, block.GetTxCount())
	for {
		tx, err := block.GetNextTx()
		if err != nil {
			return errors.Wrap(err, "get next tx")
		}
		if tx == nil {
			break // parsed all txs
		}

		txHash := tx.TxHash()
		txids = append(txids, txHash)

		// Remove from unconfirmed. Only matching are in unconfirmed.
		inUnconfirmed, unconfirmed = removeHash(*txHash, unconfirmed)

		// Remove from mempool
		inMemPool := false
		if node.state.IsReady() {
			inMemPool = node.memPool.RemoveTransaction(*txHash)
		}

		if inUnconfirmed {
			// Already seen and marked relevant
			node.txStateChannel.Add(handlers.TxState{
				handlers.ListenerMsgTxStateConfirm,
				*txHash,
			})
		} else if !inMemPool {
			// Not seen yet
			node.confTxChannel.Add(handlers.TxData{
				Msg:             tx,
				ConfirmedHeight: height,
			})

			// Transaction wasn't in the mempool.
			// Check for transactions in the mempool with conflicting inputs (double spends).
			if conflicting := node.memPool.Conflicting(tx); len(conflicting) > 0 {
				for _, confHash := range conflicting {
					if containsHash(confHash, unconfirmed) { // Only send for txs that previously matched filters.
						if err := node.txStateChannel.Add(handlers.TxState{
							handlers.ListenerMsgTxStateCancel,
							confHash,
						}); err != nil {
							logger.Warn(ctx, "Aborting block : tx channel : %s", err)
							break
						}
					}
				}
			}
		}
	}

	// Perform any block cleanup
	if err := node.CleanupBlock(ctx, txids); err != nil {
		logger.Debug(ctx, "Failed clean up after block : %s", hash)
		node.txs.ReleaseUnconfirmed(ctx) // Release unconfirmed
		return err
	}

	if !node.state.IsReady() {
		if node.state.IsPendingSync() && node.state.BlockRequestsEmpty() {
			node.state.SetInSync()
			logger.Info(ctx, "Blocks in sync at height %d", node.blocks.LastHeight())
		}
	}

	if err := node.txs.FinalizeUnconfirmed(ctx, unconfirmed); err != nil {
		return err
	}

	return nil
}

func containsHash(hash bitcoin.Hash32, list []bitcoin.Hash32) bool {
	for _, listhash := range list {
		if hash.Equal(&listhash) {
			return true
		}
	}
	return false
}

func removeHash(hash bitcoin.Hash32, list []bitcoin.Hash32) (bool, []bitcoin.Hash32) {
	for i, listhash := range list {
		if hash.Equal(&listhash) {
			return true, append(list[:i], list[i+1:]...)
		}
	}
	return false, list
}
