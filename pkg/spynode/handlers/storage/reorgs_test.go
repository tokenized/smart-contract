package storage

import (
	"bytes"
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

func TestReorgs(test *testing.T) {
	// Generate reorg
	reorg := Reorg{}
	seed := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(seed)

	ctx := context.Background()
	storageConfig := storage.NewConfig("standalone", "./tmp/test")
	store := storage.NewFilesystemStorage(storageConfig)
	repo := NewReorgRepository(store)

	t := uint32(time.Now().Unix())
	header := wire.BlockHeader{Version: 1}
	for i := 0; i < 5; i++ {
		header.Timestamp = time.Unix(int64(t), 0)
		header.Nonce = uint32(randGen.Int())
		// repo.Add(ctx, &header)
		reorg.Blocks = append(reorg.Blocks, ReorgBlock{Header: header})
		header.PrevBlock = *header.BlockHash()
		t += 600
	}

	if err := repo.Save(ctx, &reorg); err != nil {
		test.Errorf("Failed to save reorg : %v", err)
	}

	savedReorg, err := repo.GetActive(ctx)
	if err != nil {
		test.Errorf("Failed to get active reorg : %v", err)
	}
	if savedReorg == nil {
		test.Errorf("Active reorg not found : %v", err)
	}

	hash := reorg.Id()
	hash2 := savedReorg.Id()
	if !bytes.Equal(hash[:], hash2[:]) {
		test.Errorf("Active reorg doesn't match")
	}
}
