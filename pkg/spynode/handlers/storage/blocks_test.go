package storage

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/storage"
)

func TestBlocks(test *testing.T) {
	testBlockCount := 2500
	testRevertHeights := [...]int{2400, 2000, 1999, 500}

	// Generate block hashes
	blocks := make([]chainhash.Hash, 0, testBlockCount)
	seed := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(seed)
	var newHash chainhash.Hash
	bytes := make([]byte, chainhash.HashSize)
	for i := 0; i < testBlockCount; i++ {
		// Randomize bytes
		for j := 0; j < chainhash.HashSize; j++ {
			bytes[j] = byte(randGen.Intn(256))
		}
		newHash.SetBytes(bytes)
		blocks = append(blocks, newHash)
	}

	ctx := context.Background()
	storageConfig := storage.NewConfig("ap-southeast-2", "", "", "standalone", "./tmp/test")
	store := storage.NewFilesystemStorage(storageConfig)
	repo := NewBlockRepository(store)

	for _, hash := range blocks {
		repo.Add(ctx, hash)
	}

	if err := repo.Save(ctx); err != nil {
		test.Errorf("Failed to save repo : %v", err)
	}

	for _, revertHeight := range testRevertHeights {
		test.Logf("Test revert to (%d) : %s", revertHeight, blocks[revertHeight].String())

		if err := repo.Revert(ctx, revertHeight); err != nil {
			test.Errorf("Failed to revert repo : %v", err)
		}

		if *repo.LastHash() != blocks[revertHeight] {
			test.Errorf("Failed to revert repo to height %d", revertHeight)
		}

		if err := repo.Load(ctx); err != nil {
			test.Errorf("Failed to load repo after revert to %d : %v", revertHeight, err)
		}
	}
}
