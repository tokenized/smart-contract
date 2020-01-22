package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// BlockHandler exists to handle the block command.
type BlockHandler struct {
	state          *data.State
	txChannel      *TxChannel
	txStateChannel *TxStateChannel
	memPool        *data.MemPool
	blocks         *storage.BlockRepository
	txs            *storage.TxRepository
	listeners      []Listener
	txFilters      []TxFilter
	blockProcessor BlockProcessor
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(state *data.State, txChannel *TxChannel, txStateChannel *TxStateChannel,
	memPool *data.MemPool, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository,
	listeners []Listener, txFilters []TxFilter, blockProcessor BlockProcessor) *BlockHandler {

	result := BlockHandler{
		state:          state,
		txChannel:      txChannel,
		txStateChannel: txStateChannel,
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
	if !handler.state.AddBlock(receivedHash, message) {
		logger.Warn(ctx, "Block not requested : %s", receivedHash)
		if message.Header.PrevBlock == *handler.blocks.LastHash() {
			if !handler.state.AddNewBlock(receivedHash, message) {
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
			height, _ := handler.blocks.Height(hash)
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
		if err = handler.blocks.Add(ctx, &block.Header); err != nil {
			return nil, err
		}

		// If we are in sync we can save after every block
		if handler.state.IsReady() {
			handler.blocks.Save(ctx)
		}

		// Get unconfirmed "relevant" txs
		var unconfirmed []bitcoin.Hash32
		// This locks the tx repo so that propagated txs don't interfere while a block is being
		//   processed.
		unconfirmed, err = handler.txs.GetUnconfirmed(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get unconfirmed tx hashes")
		}

		// Send block notification
		height := handler.blocks.LastHeight()
		blockMessage := BlockMessage{Hash: *hash, Height: height, Time: block.Header.Timestamp}
		for _, listener := range handler.listeners {
			listener.HandleBlock(ctx, ListenerMsgBlock, &blockMessage)
		}

		// Notify Tx for block and tx listeners
		hashes, err := block.TxHashes()
		if err != nil {
			handler.txs.ReleaseUnconfirmed(ctx) // Release unconfirmed
			return nil, errors.Wrap(err, "Failed to get block tx hashes")
		}

		logger.Debug(ctx, "Processing block %d (%d tx) : %s", height, len(hashes), hash)
		inUnconfirmed := false
		for i, txHash := range hashes {
			// Remove from unconfirmed. Only matching are in unconfirmed.
			inUnconfirmed, unconfirmed = removeHash(*txHash, unconfirmed)

			// Remove from mempool
			inMemPool := false
			if handler.state.IsReady() {
				inMemPool = handler.memPool.RemoveTransaction(*txHash)
			}

			if inUnconfirmed {
				// Already seen and marked relevant
				handler.txStateChannel.Add(TxState{
					ListenerMsgTxStateConfirm,
					*txHash,
				})
			} else if !inMemPool {
				// Not seen yet
				handler.txChannel.Add(TxData{
					Msg:             block.Transactions[i],
					ConfirmedHeight: height,
				})

				// Transaction wasn't in the mempool.
				// Check for transactions in the mempool with conflicting inputs (double spends).
				if conflicting := handler.memPool.Conflicting(block.Transactions[i]); len(conflicting) > 0 {
					for _, confHash := range conflicting {
						if containsHash(confHash, unconfirmed) { // Only send for txs that previously matched filters.
							handler.txStateChannel.Add(TxState{
								ListenerMsgTxStateCancel,
								confHash,
							})
						}
					}
				}
			}
		}

		// Perform any block cleanup
		if err := handler.blockProcessor.ProcessBlock(ctx, block); err != nil {
			logger.Debug(ctx, "Failed clean up after block : %s", hash)
			handler.txs.ReleaseUnconfirmed(ctx) // Release unconfirmed
			return nil, err
		}

		if !handler.state.IsReady() {
			if handler.state.IsPendingSync() && handler.state.BlockRequestsEmpty() {
				handler.state.SetInSync()
				logger.Info(ctx, "Blocks in sync at height %d", handler.blocks.LastHeight())
			}
		}

		if err := handler.txs.FinalizeUnconfirmed(ctx, unconfirmed); err != nil {
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
func CalculateMerkleHash(ctx context.Context, txs []*wire.MsgTx) (*bitcoin.Hash32, error) {
	if len(txs) == 0 {
		// Zero hash
		result, err := bitcoin.NewHash32FromStr("0000000000000000000000000000000000000000000000000000000000000000")
		return result, err
	}
	if len(txs) == 1 {
		// Hash of only tx
		result := txs[0].TxHash()
		return result, nil
	}

	// Tree root hash
	hashes := make([]*bitcoin.Hash32, 0, len(txs))
	for _, tx := range txs {
		hash := tx.TxHash()
		hashes = append(hashes, hash)
	}
	return CalculateMerkleLevel(ctx, hashes), nil
}

// CalculateMerkleLevel calculates one level of the merkle tree
func CalculateMerkleLevel(ctx context.Context, txids []*bitcoin.Hash32) *bitcoin.Hash32 {
	if len(txids) == 1 {
		return combinedHash(ctx, txids[0], txids[0]) // Hash it with itself
	}

	if len(txids) == 2 {
		return combinedHash(ctx, txids[0], txids[1]) // Hash both together
	}

	// More level calculations required (recursive)
	// Combine every two hashes and put them in a list to process again.
	nextLevel := make([]*bitcoin.Hash32, 0, (len(txids)/2)+1)
	var tx1 *bitcoin.Hash32 = nil
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
func combinedHash(ctx context.Context, hash1 *bitcoin.Hash32, hash2 *bitcoin.Hash32) *bitcoin.Hash32 {
	data := make([]byte, bitcoin.Hash32Size*2)
	copy(data[:bitcoin.Hash32Size], hash1[:])
	copy(data[bitcoin.Hash32Size:], hash2[:])
	result, _ := bitcoin.NewHash32(bitcoin.DoubleSha256(data))
	return result
}
