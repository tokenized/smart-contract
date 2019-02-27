package storage

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestTransactions(test *testing.T) {
	testTxCount := 100
	testBlockHeight := 500

	// Generate block hashes
	txs := make([]chainhash.Hash, 0, testTxCount)
	seed := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(seed)
	var newHash chainhash.Hash
	bytes := make([]byte, chainhash.HashSize)
	for i := 0; i < testTxCount; i++ {
		// Randomize bytes
		for j := 0; j < chainhash.HashSize; j++ {
			bytes[j] = byte(randGen.Intn(256))
		}
		newHash.SetBytes(bytes)
		txs = append(txs, newHash)
	}

	ctx := context.Background()
	storageConfig := storage.NewConfig("ap-southeast-2", "", "", "standalone", "./tmp/test")
	store := storage.NewFilesystemStorage(storageConfig)
	repo := NewTxRepository(store)

	// Remove any previous data
	repo.ClearBlock(ctx, testBlockHeight)

	for i, hash := range txs {
		if _, err := repo.Add(ctx, hash, testBlockHeight); err != nil {
			test.Errorf("Failed to add tx %d : %v", i, err)
		}
	}

	returnedtxs, err := repo.GetBlock(ctx, testBlockHeight)
	if err != nil {
		test.Errorf("Failed to get tx for block : %v", err)
	}

	if len(returnedtxs) != len(txs) {
		test.Errorf("Returned tx count %d should be %d", len(returnedtxs), len(txs))
	}

	for i, txid := range returnedtxs {
		if txid != txs[i] {
			test.Errorf("Tx %d hash doesn't match : %s", i, txid)
		}
	}

	if err := repo.RemoveBlock(ctx, testBlockHeight); err != nil {
		test.Errorf("Failed to remove block file")
	}

	returnedtxs, err = repo.GetBlock(ctx, testBlockHeight)
	if err != nil {
		test.Errorf("Failed to get tx for block : %v", err)
	}
	if len(returnedtxs) > 0 {
		test.Errorf("Previous remove failed")
	}
}
