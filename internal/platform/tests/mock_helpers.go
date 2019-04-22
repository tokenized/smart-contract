package tests

import (
	"bytes"
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Generate a fake funding tx so inspector can build off of it.
func MockFundingTx(ctx context.Context, node *mockRpcNode, value uint64, pkh []byte) *wire.MsgTx {
	result := wire.NewMsgTx(2)
	result.TxOut = append(result.TxOut, wire.NewTxOut(int64(value), txbuilder.P2PKHScriptForPKH(pkh)))
	node.AddTX(ctx, result)
	return result
}

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
