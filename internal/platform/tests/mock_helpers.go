package tests

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"

	"github.com/pkg/errors"
)

// Generate a fake funding tx so inspector can build off of it.
func MockFundingTx(ctx context.Context, node *mockRpcNode, value uint64, address bitcoin.RawAddress) *wire.MsgTx {
	result := wire.NewMsgTx(2)
	script, _ := address.LockingScript()
	result.TxOut = append(result.TxOut, wire.NewTxOut(value, script))
	node.SaveTX(ctx, result)
	return result
}

// ============================================================
// RPC Node

type mockRpcNode struct {
	txs  map[bitcoin.Hash32]*wire.MsgTx
	lock sync.Mutex
}

func newMockRpcNode() *mockRpcNode {
	result := mockRpcNode{
		txs: make(map[bitcoin.Hash32]*wire.MsgTx),
	}
	return &result
}

func (cache *mockRpcNode) SaveTX(ctx context.Context, tx *wire.MsgTx) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.txs[*tx.TxHash()] = tx.Copy()
	return nil
}

func (cache *mockRpcNode) GetTX(ctx context.Context, txid *bitcoin.Hash32) (*wire.MsgTx, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	tx, ok := cache.txs[*txid]
	if ok {
		return tx, nil
	}
	return nil, errors.New("Couldn't find tx in cache")
}

func (cache *mockRpcNode) GetOutputs(ctx context.Context, outpoints []wire.OutPoint) ([]bitcoin.UTXO, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	results := make([]bitcoin.UTXO, len(outpoints))
	for i, outpoint := range outpoints {
		tx, ok := cache.txs[outpoint.Hash]
		if !ok {
			return results, fmt.Errorf("Couldn't find tx in cache : %s", outpoint.Hash.String())
		}

		if int(outpoint.Index) >= len(tx.TxOut) {
			return results, fmt.Errorf("Invalid output index for txid %d/%d : %s", outpoint.Index,
				len(tx.TxOut), outpoint.Hash.String())
		}

		results[i] = bitcoin.UTXO{
			Hash:          outpoint.Hash,
			Index:         outpoint.Index,
			Value:         tx.TxOut[outpoint.Index].Value,
			LockingScript: tx.TxOut[outpoint.Index].PkScript,
		}
	}
	return results, nil
}

// ============================================================
// Headers

type mockHeaders struct {
	height int
	hashes []*bitcoin.Hash32
	times  []uint32
	lock   sync.Mutex
}

func newMockHeaders() *mockHeaders {
	h := &mockHeaders{}
	h.Reset()
	return h
}

func (h *mockHeaders) LastHeight(ctx context.Context) int {
	h.lock.Lock()
	defer h.lock.Unlock()

	return h.height
}

func (h *mockHeaders) Hash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if height > h.height {
		return nil, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return nil, errors.New("Hash unavailable")
	}
	return h.hashes[h.height-height], nil
}

func (h *mockHeaders) Time(ctx context.Context, height int) (uint32, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if height > h.height {
		return 0, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return 0, errors.New("Time unavailable")
	}
	return h.times[h.height-height], nil
}

func (h *mockHeaders) Reset() {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.height = 0
	h.hashes = nil
	h.times = nil
}

func (h *mockHeaders) Populate(ctx context.Context, height, count int) error {
	h.lock.Lock()
	defer h.lock.Unlock()

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
