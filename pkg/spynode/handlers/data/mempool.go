package data

import (
	sync "github.com/sasha-s/go-deadlock"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// MemPool is used for managing announced transactions that haven't confirmed yet.
// The mempool is non-persistent and is mainly used to prevent duplicate tx requests.
type MemPool struct {
	txs      map[bitcoin.Hash32]*memPoolTx        // Lookup of block height by hash.
	inputs   map[bitcoin.Hash32][]*bitcoin.Hash32 // Lookup by hash of outpoint. Used to find conflicting inputs.
	requests map[bitcoin.Hash32]time.Time         // Transactions that have been requested
	mutex    sync.Mutex
}

// NewMemPool returns a new MemPool.
func NewMemPool() *MemPool {
	result := MemPool{
		txs:      make(map[bitcoin.Hash32]*memPoolTx),
		inputs:   make(map[bitcoin.Hash32][]*bitcoin.Hash32),
		requests: make(map[bitcoin.Hash32]time.Time),
	}
	return &result
}

// Adds an active request for a tx.
// This is to prevent duplicate requests and receiving the same tx from multiple peers.
// Returns:
//   bool - True if we already have the tx
//   bool - True if the tx should be requested
func (memPool *MemPool) AddRequest(txid *bitcoin.Hash32) (bool, bool) {
	memPool.mutex.Lock()
	defer memPool.mutex.Unlock()

	now := time.Now()
	tx, exists := memPool.txs[*txid]
	if exists {
		if len(tx.outPoints) > 0 {
			return true, false // Already in the mempool
		}
	} else {
		// Add tx
		memPool.txs[*txid] = newMemPoolTx(now)
	}

	requestTime, requested := memPool.requests[*txid]
	if !requested || now.Sub(requestTime).Seconds() > 3 {
		// Tx has not been requested yet or the previous request is old
		memPool.requests[*txid] = now
		return false, true
	}

	return false, false // Another request is still active
}

// Adds a timestamped tx hash to the mempool
// Returns:
//   []*bitcoin.Hash32 - list of conflicting transactions (not including this tx) if there are
//     conflicts with inputs (double spends).
//   bool - true if the tx isn't already in the mempool and was added
func (memPool *MemPool) AddTransaction(tx *wire.MsgTx) ([]*bitcoin.Hash32, bool) {
	memPool.mutex.Lock()
	defer memPool.mutex.Unlock()

	result := make([]*bitcoin.Hash32, 0)
	hash := tx.TxHash()

	memTx, exists := memPool.txs[*hash]
	if exists {
		if len(memTx.outPoints) > 0 {
			return result, false // Already in the mempool
		}
	} else {
		// Add tx
		memTx = newMemPoolTx(time.Now())
		memPool.txs[*hash] = memTx
	}

	// Add outpoints to mempool tx
	memTx.populateMemPoolTx(tx)

	// Add inputs while checking for conflicts
	for _, outpoint := range memTx.outPoints {
		outpointHash := outpoint.OutpointHash()
		list, exists := memPool.inputs[*outpointHash]
		if exists {
			// Append conflicting
			// It is possible tx conflict on more than one input and we don't want duplicates in
			//   the result list.
			appendIfNotContained(result, list)
			list = append(list, hash)
		} else {
			// Create new list with only this tx hash
			list := make([]*bitcoin.Hash32, 1)
			list[0] = hash
			memPool.inputs[*outpointHash] = list
		}
	}

	return result, true
}

// Appends the items in add to list if they are not already in list
func appendIfNotContained(list []*bitcoin.Hash32, add []*bitcoin.Hash32) {
	for _, addHash := range add {
		found := false
		for _, hash := range list {
			if *hash == *addHash {
				found = true
				break
			}
		}

		if !found {
			list = append(list, addHash)
		}
	}
}

// Removes a tx hash from the mempool
// Returns true if the tx was in the mempool
func (memPool *MemPool) RemoveTransaction(hash *bitcoin.Hash32) bool {
	memPool.mutex.Lock()
	defer memPool.mutex.Unlock()

	return memPool.removeTransaction(hash)
}

// Removes a tx hash from the mempool
// Returns true if the tx was in the mempool
func (memPool *MemPool) removeTransaction(hash *bitcoin.Hash32) bool {
	tx, exists := memPool.txs[*hash]
	if exists {
		// Remove outpoints
		for _, outpoint := range tx.outPoints {
			outpointHash := outpoint.OutpointHash()
			otherHashes, exists := memPool.inputs[*outpointHash]
			if exists { // It should always exist
				if len(otherHashes) > 1 {
					// Remove this outpoint hash from the list
					for i, otherHash := range otherHashes {
						if otherHash.Equal(outpointHash) {
							otherHashes = append(otherHashes[:i], otherHashes[i+1:]...)
							break
						}
					}
				} else {
					delete(memPool.inputs, *outpointHash)
				}
			}
		}

		// Remove tx
		delete(memPool.txs, *hash)
	}
	return exists
}

// Returns true if the transaction is in the mempool
func (memPool *MemPool) TransactionExists(hash *bitcoin.Hash32) bool {
	memPool.mutex.Lock()
	defer memPool.mutex.Unlock()

	tx, exists := memPool.txs[*hash]
	if !exists {
		return false
	}

	return len(tx.outPoints) > 0
}

// Returns txids of any transactions from the mempool with inputs that conflict with the specified
//   transaction.
// Also removes them from the mempool.
func (memPool *MemPool) Conflicting(tx *wire.MsgTx) []*bitcoin.Hash32 {
	memPool.mutex.Lock()
	defer memPool.mutex.Unlock()

	result := make([]*bitcoin.Hash32, 0, 1)
	// Check for conflicting inputs
	for _, input := range tx.TxIn {
		if list, exists := memPool.inputs[*input.PreviousOutPoint.OutpointHash()]; exists {
			for _, hash := range list {
				result = append(result, hash)
				memPool.removeTransaction(hash)
			}
		}
	}
	return result
}

type memPoolTx struct {
	time      time.Time
	outPoints []wire.OutPoint
}

func newMemPoolTx(t time.Time) *memPoolTx {
	result := memPoolTx{
		time: t,
	}
	return &result
}

func (tx *memPoolTx) populateMemPoolTx(txMsg *wire.MsgTx) {
	if len(tx.outPoints) > 0 {
		return // Already populated
	}
	tx.outPoints = make([]wire.OutPoint, 0, len(txMsg.TxIn))

	for _, input := range txMsg.TxIn {
		tx.outPoints = append(tx.outPoints, input.PreviousOutPoint)
	}
}
