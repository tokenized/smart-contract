package tests

import (
	"bytes"
	"context"
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// ============================================================
// RPC Node

type mockRpcNode struct {
	txs    []*wire.MsgTx
	params *chaincfg.Params
}

func (cache *mockRpcNode) AddTX(ctx context.Context, tx *wire.MsgTx) error {
	cache.txs = append(cache.txs, tx)
	return nil
}

func (cache *mockRpcNode) GetTX(ctx context.Context, txid *chainhash.Hash) (*wire.MsgTx, error) {
	for _, tx := range cache.txs {
		hash := tx.TxHash()
		if bytes.Equal(hash[:], txid[:]) {
			return tx, nil
		}
	}
	return nil, errors.New("Couldn't find tx in cache")
}

func (cache *mockRpcNode) GetChainParams() *chaincfg.Params {
	return cache.params
}
