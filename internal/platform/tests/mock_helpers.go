package tests

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Generate a fake funding tx so inspector can build off of it.
func MockFundingTx(ctx context.Context, node *mockRpcNode, value uint64, address bitcoin.Address) *wire.MsgTx {
	result := wire.NewMsgTx(2)
	result.TxOut = append(result.TxOut, wire.NewTxOut(int64(value), address.LockingScript()))
	node.SaveTX(ctx, result)
	return result
}

// ============================================================
// RPC Node

type mockRpcNode struct {
	txs    map[chainhash.Hash]*wire.MsgTx
	params *chaincfg.Params
	lock   sync.Mutex
}

func newMockRpcNode(params *chaincfg.Params) *mockRpcNode {
	result := mockRpcNode{
		txs:    make(map[chainhash.Hash]*wire.MsgTx),
		params: params,
	}
	return &result
}

func (cache *mockRpcNode) SaveTX(ctx context.Context, tx *wire.MsgTx) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.txs[tx.TxHash()] = tx
	return nil
}

func (cache *mockRpcNode) GetTX(ctx context.Context, txid *chainhash.Hash) (*wire.MsgTx, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	tx, ok := cache.txs[*txid]
	if ok {
		return tx, nil
	}
	return nil, errors.New("Couldn't find tx in cache")
}

func (cache *mockRpcNode) GetTXs(ctx context.Context, txids []*chainhash.Hash) ([]*wire.MsgTx, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	result := make([]*wire.MsgTx, len(txids))
	for i, txid := range txids {
		tx, ok := cache.txs[*txid]
		if !ok {
			return result, fmt.Errorf("Couldn't find tx in cache : %s", txid.String())
		}
		result[i] = tx
	}
	return result, nil
}

func (cache *mockRpcNode) GetChainParams() *chaincfg.Params {
	return cache.params
}

// ============================================================
// Headers

type mockHeaders struct {
	height int
	hashes []*chainhash.Hash
	times  []uint32
}

func newMockHeaders() *mockHeaders {
	h := &mockHeaders{}
	h.Reset()
	return h
}

func (h *mockHeaders) LastHeight(ctx context.Context) int {
	return h.height
}

func (h *mockHeaders) Hash(ctx context.Context, height int) (*chainhash.Hash, error) {
	if height > h.height {
		return nil, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return nil, errors.New("Hash unavailable")
	}
	return h.hashes[h.height-height], nil
}

func (h *mockHeaders) Time(ctx context.Context, height int) (uint32, error) {
	if height > h.height {
		return 0, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return 0, errors.New("Time unavailable")
	}
	return h.times[h.height-height], nil
}

func (h *mockHeaders) Reset() {
	h.height = 0
	h.hashes = nil
	h.times = nil
}

func (h *mockHeaders) Populate(ctx context.Context, height, count int) error {
	h.height = height
	h.hashes = nil
	h.times = nil

	timestamp := uint32(time.Now().Unix())
	for i := 0; i < count; i++ {
		h.hashes = append(h.hashes, RandomHash())
		h.times = append(h.times, timestamp)
		timestamp -= 600
	}
	return nil
}
