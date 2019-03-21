package handlers

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/inspector"
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

func (txCache *TxCache) GetTx(ctx context.Context, txid *chainhash.Hash) *inspector.Transaction {
	result, exists := txCache.cache[*txid]
	if !exists {
		return nil
	}
	return result
}

func (txCache *TxCache) SaveTx(ctx context.Context, tx *inspector.Transaction) error {
	txCache.cache[tx.Hash] = tx
	return nil
}
