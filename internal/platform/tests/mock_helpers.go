package tests

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/spynode/pkg/client"

	"github.com/pkg/errors"
)

// Generate a fake funding tx so inspector can build off of it.
func MockFundingTx(ctx context.Context, node *mockRpcNode, value uint64,
	address bitcoin.RawAddress) *wire.MsgTx {
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

func (cache *mockRpcNode) GetOutputs(ctx context.Context,
	outpoints []wire.OutPoint) ([]bitcoin.UTXO, error) {
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
	height  int
	headers []*wire.BlockHeader
	lock    sync.Mutex
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

func (h *mockHeaders) BlockHash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if height > h.height {
		return nil, errors.New("Above current height")
	}
	if h.height-height >= len(h.headers) {
		return nil, errors.New("Hash unavailable")
	}
	return h.headers[h.height-height].BlockHash(), nil
}

func (h *mockHeaders) GetHeaders(ctx context.Context, height, count int) (*client.Headers, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if height+count > h.height {
		return nil, errors.New("Above current height")
	}
	if h.height-height >= len(h.headers) {
		return nil, errors.New("Headers unavailable")
	}

	return &client.Headers{
		RequestHeight: int32(height),
		StartHeight:   uint32(h.height - height),
		Headers:       h.headers[h.height-height : h.height-height+count],
	}, nil
}

func (h *mockHeaders) Reset() {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.height = 0
	h.headers = nil
}

func (h *mockHeaders) Populate(ctx context.Context, height, count int) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.height = height
	timestamp := time.Now()
	h.headers = make([]*wire.BlockHeader, count)
	prevHash := &bitcoin.Hash32{}
	rand.Read(prevHash[:])
	for i := 0; i < count; i++ {
		newHeader := &wire.BlockHeader{
			Version:   0,
			PrevBlock: *prevHash,
			Timestamp: timestamp,
			Bits:      rand.Uint32(),
			Nonce:     rand.Uint32(),
		}
		rand.Read(newHeader.MerkleRoot[:])
		h.headers[i] = newHeader
		timestamp.Add(10 * time.Minute)
		prevHash = newHeader.BlockHash()
	}
	return nil
}
