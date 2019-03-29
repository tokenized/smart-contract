package listeners

import (
	"context"
	"encoding/json"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	storageKey = "tx_cache"
)

type TxCache struct {
	cache map[chainhash.Hash]*inspector.Transaction
}

func NewTxCache() *TxCache {
	result := TxCache{
		cache: make(map[chainhash.Hash]*inspector.Transaction),
	}
	return &result
}

func (txCache *TxCache) Save(ctx context.Context, masterDB *db.DB) error {
	// Build array
	list := make([]*inspector.Transaction, 0, len(txCache.cache))
	for _, tx := range txCache.cache {
		list = append(list, tx)
	}

	// Save the cache list
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	logger.Verbose(ctx, "Saving %d tx from tx cache", len(txCache.cache))

	return masterDB.Put(ctx, storageKey, data)
}

func (txCache *TxCache) Load(ctx context.Context, masterDB *db.DB) error {
	data, err := masterDB.Fetch(ctx, storageKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil
		}
		return err
	}

	// Prepare the contract object
	list := make([]*inspector.Transaction, 0)
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	// Load into cache
	for _, tx := range list {
		txCache.cache[tx.Hash] = tx
	}
	logger.Verbose(ctx, "Loaded %d tx into tx cache", len(txCache.cache))

	return nil
}

func (txCache *TxCache) GetTx(ctx context.Context, txid *chainhash.Hash) *inspector.Transaction {
	result, exists := txCache.cache[*txid]
	if !exists {
		return nil
	}
	return result
}

func (txCache *TxCache) RemoveTx(ctx context.Context, txid *chainhash.Hash) {
	delete(txCache.cache, *txid)
}

func (txCache *TxCache) SaveTx(ctx context.Context, tx *inspector.Transaction) error {
	txCache.cache[tx.Hash] = tx
	return nil
}
