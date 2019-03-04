package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
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

// Receives messages about blocks
// The second parameter will either be a BlockMessage or a chainhash.Hash with the hash of a
//   transaction contained in the previous Block.
type BlockListener interface {
	Handle(context.Context, interface{}) error
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

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the contrete
// handler.
func (handler *BlockHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgBlock)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgBlock")
	}

	hash := msg.BlockHash()
	handler.state.RemoveBlockRequest(&hash) // Remove from requested

	// If we already have this block, we don't need to ask for more
	if handler.blocks.Contains(hash) {
		height, _ := handler.blocks.Height(&hash)
		return nil, errors.New(fmt.Sprintf("Already have block (%d) : %s", height, hash.String()))
	}

	if msg.Header.PrevBlock != *handler.blocks.LastHash() {
		// Ignore this as it can happen when there is a reorg.
		return nil, nil // Unknown or out of order block
	}

	// Validate
	valid, err := validateMerkleHash(ctx, msg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to validate merkle hash")
	}
	if !valid {
		return nil, errors.New(fmt.Sprintf("Invalid merkle hash for block %s", hash.String()))
	}

	// Add to repo
	if err := handler.blocks.Add(ctx, hash); err != nil {
		return nil, err
	}

	// If we are in sync we can save after every block
	if handler.state.IsInSync {
		handler.blocks.Save(ctx)
	}

	// Request more blocks
	response := []wire.Message{}
	getBlocks := wire.NewMsgGetData() // Block request message

	for { // This should probably never loop more than once
		requestHash, _ := handler.state.GetNextBlockToRequest(&hash)
		if requestHash == nil {
			break
		}

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

	var unconfirmed []chainhash.Hash
	if handler.state.IsInSync {
		// Get unconfirmed "relevant" txs
		var err error
		unconfirmed, err = handler.txs.GetBlock(ctx, -1)
		if err != nil {
			return response, errors.Wrap(err, "Failed to get unconfirmed tx hashes")
		}
	}

	// Send block notification
	var removed bool = false
	relevant := make([]chainhash.Hash, 0)
	height := handler.blocks.LastHeight()
	blockMessage := BlockMessage{Hash: hash, Height: height}
	for _, listener := range handler.listeners {
		if _, err := listener.Handle(ctx, ListenerMsgBlock, blockMessage); err != nil {
			return response, err
		}
	}

	// Notify Tx for block and tx listeners
	hashes, err := msg.TxHashes()
	if err != nil {
		if handler.state.IsInSync {
			handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
		}
		return response, errors.Wrap(err, "Failed to get block tx hashes")
	}
	logger.Log(ctx, logger.Debug, "Processing block %d (%d tx) : %s", height, len(hashes), hash.String())
	for i, txHash := range hashes {
		if handler.state.IsInSync {
			// Remove from unconfirmed. Only matching are in unconfirmed.
			removed, unconfirmed = removeHash(&txHash, unconfirmed)
		}

		// Send full tx to listener if we aren't in sync yet and don't have a populated mempool.
		// Or if it isn't in the mempool (not sent to listener yet).
		marked := false
		if !removed { // Full tx hasn't been sent to listener yet
			if matchesFilter(ctx, msg.Transactions[i], handler.txFilters) {
				var mark bool
				var err error
				for _, listener := range handler.listeners {
					if mark, err = listener.Handle(ctx, ListenerMsgTx, *msg.Transactions[i]); err != nil {
						return response, err
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

		if handler.state.IsInSync && !handler.memPool.RemoveTransaction(&txHash) {
			// Transaction wasn't in the mempool.
			// Check for transactions in the mempool with conflicting inputs (double spends).
			if conflicting := handler.memPool.Conflicting(msg.Transactions[i]); len(conflicting) > 0 {
				for _, hash := range conflicting {
					if containsHash(&txHash, unconfirmed) { // Only send for txs that previously matched filters.
						for _, listener := range handler.listeners {
							if _, err := listener.Handle(ctx, ListenerMsgTxCancel, *hash); err != nil {
								return response, err
							}
						}
					}
				}
			}
		}

		if marked {
			// Notify of confirm
			for _, listener := range handler.listeners {
				if _, err := listener.Handle(ctx, ListenerMsgTxConfirm, txHash); err != nil {
					return response, err
				}
			}
		}
	}

	// Perform any block cleanup
	err = handler.blockProcessor.ProcessBlock(ctx, msg)
	if err != nil {
		if handler.state.IsInSync {
			handler.txs.ReleaseBlock(ctx, -1) // Release unconfirmed
		}
		return response, err
	}

	if handler.state.IsInSync {
		return response, handler.txs.FinalizeBlock(ctx, unconfirmed, handler.state.IsInSync, relevant, height)
	} else {
		if handler.state.PendingSync && handler.state.BlockRequestsEmpty() {
			handler.state.IsInSync = true
			logger.Log(ctx, logger.Info, "Blocks in sync at height %d", handler.blocks.LastHeight())
		}
		return response, handler.txs.SetBlock(ctx, relevant, height)
	}
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

// Validate the merkle root hash against the transactions contained.
// Returns true if the merkle root hash is valid.
func validateMerkleHash(ctx context.Context, block *wire.MsgBlock) (bool, error) {
	merkleHash, err := CalculateMerkleHash(ctx, block.Transactions)
	if err != nil {
		return false, err
	}
	return *merkleHash == block.Header.MerkleRoot, nil
}

// Calculate a merkle tree root hash for a set of transactions
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

// Calculate one level of the merkle tree
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

// Combine two hashes
func combinedHash(ctx context.Context, hash1 *chainhash.Hash, hash2 *chainhash.Hash) *chainhash.Hash {
	data := make([]byte, chainhash.HashSize*2)
	copy(data[:chainhash.HashSize], hash1[:])
	copy(data[chainhash.HashSize:], hash2[:])
	result := chainhash.DoubleHashH(data)
	return &result
}
