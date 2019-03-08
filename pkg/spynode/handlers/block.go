package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

// BlockHandler exists to handle the block command.
type BlockHandler struct {
	state          *data.State
	memPool        *data.MemPool
	blocks         *storage.BlockRepository
	txs            *storage.TxRepository
	listeners      []Listener
	txFilters      []TxFilter
	blockProcessor BlockProcessor
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(state *data.State, memPool *data.MemPool, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository, listeners []Listener, txFilters []TxFilter, blockProcessor BlockProcessor) *BlockHandler {
	result := BlockHandler{
		state:          state,
		memPool:        memPool,
		blocks:         blockRepo,
		txs:            txRepo,
		listeners:      listeners,
		txFilters:      txFilters,
		blockProcessor: blockProcessor,
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
	logger.Debug(ctx, "Received block : %s", receivedHash)
	if !handler.state.AddBlock(&receivedHash, message) {
		logger.Warn(ctx, "Block not requested : %s", receivedHash)
		if message.Header.PrevBlock == *handler.blocks.LastHash() {
			if !handler.state.AddNewBlock(&receivedHash, message) {
				return nil, nil
			}
		} else {
			return nil, nil
		}
	}

	var err error
	var valid bool
	for {
		block := handler.state.NextBlock()

		if block == nil {
			break // No more blocks available to process
		}

		hash := block.BlockHash()

		// If we already have this block, we don't need to ask for more
		if handler.blocks.Contains(hash) {
			height, _ := handler.blocks.Height(&hash)
			logger.Warn(ctx, "Already have block (%d) : %s", height, hash)
			return nil, nil
		}

		if block.Header.PrevBlock != *handler.blocks.LastHash() {
			// Ignore this as it can happen when there is a reorg.
			logger.Warn(ctx, "Not next block : %s", hash)
			logger.Warn(ctx, "Previous hash : %s", block.Header.PrevBlock)
			return nil, nil // Unknown or out of order block
		}

		// Validate
		valid, err = validateMerkleHash(ctx, block)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to validate merkle hash")
		}
		if !valid {
			return nil, errors.New(fmt.Sprintf("Invalid merkle hash for block %s", hash))
		}

		// Add to repo
		if err = handler.blocks.Add(ctx, hash); err != nil {
			return nil, err
		}

		// If we are in sync we can save after every block
		if handler.state.IsReady() {
			handler.blocks.Save(ctx)
		}

		// Get unconfirmed "relevant" txs
		var unconfirmed []chainhash.Hash
		// This locks the tx repo so that propagated txs don't interfere while a block is being
		//   processed.
		unconfirmed, err = handler.txs.GetBlock(ctx, -1)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get unconfirmed tx hashes")
		}

		// Send block notification
		var removed bool = false
		relevant := make([]chainhash.Hash, 0)
		height := handler.blocks.LastHeight()
		blockMessage := BlockMessage{Hash: hash, Height: height}
		for _, listener := range handler.listeners {
			if err = listener.HandleBlock(ctx, ListenerMsgBlock, &blockMessage); err != nil {
				handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
				return nil, err
			}
		}

		// Notify Tx for block and tx listeners
		var hashes []chainhash.Hash
		hashes, err = block.TxHashes()
		if err != nil {
			handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
			return nil, errors.Wrap(err, "Failed to get block tx hashes")
		}

		logger.Debug(ctx, "Processing block %d (%d tx) : %s", height, len(hashes), hash)
		for i, txHash := range hashes {
			// Remove from unconfirmed. Only matching are in unconfirmed.
			removed, unconfirmed = removeHash(&txHash, unconfirmed)

			// Send full tx to listener if we aren't in sync yet and don't have a populated mempool.
			// Or if it isn't in the mempool (not sent to listener yet).
			marked := false
			if !removed { // Full tx hasn't been sent to listener yet
				if matchesFilter(ctx, block.Transactions[i], handler.txFilters) {
					var mark bool
					for _, listener := range handler.listeners {
						if mark, err = listener.HandleTx(ctx, block.Transactions[i]); err != nil {
							handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
							return nil, err
						}
						if mark {
							marked = true
						}
					}
					if marked {
						relevant = append(relevant, txHash)
					}
				}
			}

			if handler.state.IsReady() && !handler.memPool.RemoveTransaction(&txHash) {
				// Transaction wasn't in the mempool.
				// Check for transactions in the mempool with conflicting inputs (double spends).
				if conflicting := handler.memPool.Conflicting(block.Transactions[i]); len(conflicting) > 0 {
					for _, hash := range conflicting {
						if containsHash(&txHash, unconfirmed) { // Only send for txs that previously matched filters.
							for _, listener := range handler.listeners {
								if err = listener.HandleTxState(ctx, ListenerMsgTxStateCancel, *hash); err != nil {
									handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
									return nil, err
								}
							}
						}
					}
				}
			}

			if marked || removed {
				// Notify of confirm
				for _, listener := range handler.listeners {
					if err = listener.HandleTxState(ctx, ListenerMsgTxStateConfirm, txHash); err != nil {
						handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
						return nil, err
					}
				}
			}
		}

		// Perform any block cleanup
		err = handler.blockProcessor.ProcessBlock(ctx, block)
		if err != nil {
			logger.Debug(ctx, "Failed clean up after block : %s", hash)
			handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
			return nil, err
		}

		if !handler.state.IsReady() {
			if handler.state.IsPendingSync() && handler.state.BlockRequestsEmpty() {
				handler.state.SetInSync()
				logger.Info(ctx, "Blocks in sync at height %d", handler.blocks.LastHeight())
			}
		}

		logger.Debug(ctx, "Finalizing block : %s", hash)
		if err := handler.txs.FinalizeBlock(ctx, unconfirmed, relevant, height); err != nil {
			return nil, err
		}
	}

	// Request more blocks
	response := []wire.Message{}
	getBlocks := wire.NewMsgGetData() // Block request message

	for {
		requestHash, _ := handler.state.GetNextBlockToRequest()
		if requestHash == nil {
			break
		}

		logger.Debug(ctx, "Requesting block : %s", requestHash)
		getBlocks.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, requestHash))
		if len(getBlocks.InvList) == wire.MaxInvPerMsg {
			// Start new get data (block request) message
			response = append(response, getBlocks)
			getBlocks = wire.NewMsgGetData()
		}
	}

	// Add any non-full requests.
	if len(getBlocks.InvList) > 0 {
		response = append(response, getBlocks)
	}

	return response, nil
}

func containsHash(hash *chainhash.Hash, list []chainhash.Hash) bool {
	for _, listhash := range list {
		if *hash == listhash {
			return true
		}
	}
	return false
}

func removeHash(hash *chainhash.Hash, list []chainhash.Hash) (bool, []chainhash.Hash) {
	for i, listhash := range list {
		if *hash == listhash {
			return true, append(list[:i], list[i+1:]...)
		}
	}
	return false, list
}

// validateMerkleHash validates the merkle root hash against the transactions contained.
// Returns true if the merkle root hash is valid.
func validateMerkleHash(ctx context.Context, block *wire.MsgBlock) (bool, error) {
	merkleHash, err := CalculateMerkleHash(ctx, block.Transactions)
	if err != nil {
		return false, err
	}
	return *merkleHash == block.Header.MerkleRoot, nil
}

// CalculateMerkleHash calculates a merkle tree root hash for a set of transactions
func CalculateMerkleHash(ctx context.Context, txs []*wire.MsgTx) (*chainhash.Hash, error) {
	if len(txs) == 0 {
		// Zero hash
		result, err := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
		return result, err
	}
	if len(txs) == 1 {
		// Hash of only tx
		result := txs[0].TxHash()
		return &result, nil
	}

	// Tree root hash
	hashes := make([]*chainhash.Hash, 0, len(txs))
	for _, tx := range txs {
		hash := tx.TxHash()
		hashes = append(hashes, &hash)
	}
	return CalculateMerkleLevel(ctx, hashes), nil
}

// CalculateMerkleLevel calculates one level of the merkle tree
func CalculateMerkleLevel(ctx context.Context, txids []*chainhash.Hash) *chainhash.Hash {
	if len(txids) == 1 {
		return combinedHash(ctx, txids[0], txids[0]) // Hash it with itself
	}

	if len(txids) == 2 {
		return combinedHash(ctx, txids[0], txids[1]) // Hash both together
	}

	// More level calculations required (recursive)
	// Combine every two hashes and put them in a list to process again.
	nextLevel := make([]*chainhash.Hash, 0, (len(txids)/2)+1)
	var tx1 *chainhash.Hash = nil
	for _, txid := range txids {
		if tx1 == nil {
			tx1 = txid
			continue
		}
		nextLevel = append(nextLevel, combinedHash(ctx, tx1, txid))
		tx1 = nil
	}

	// If there is a remainder, hash it with itself
	if tx1 != nil {
		nextLevel = append(nextLevel, combinedHash(ctx, tx1, tx1))
	}

	return CalculateMerkleLevel(ctx, nextLevel)
}

// combinedHash combines two hashes
func combinedHash(ctx context.Context, hash1 *chainhash.Hash, hash2 *chainhash.Hash) *chainhash.Hash {
	data := make([]byte, chainhash.HashSize*2)
	copy(data[:chainhash.HashSize], hash1[:])
	copy(data[chainhash.HashSize:], hash2[:])
	result := chainhash.DoubleHashH(data)
	return &result
}
