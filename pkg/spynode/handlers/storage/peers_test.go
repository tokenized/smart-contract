package storage

import (
	"context"
	"testing"

	"github.com/tokenized/smart-contract/pkg/storage"
)

func TestPeers(test *testing.T) {
	addresses := []string{
		"test 1",
		"test 2",
		"test 3",
		"test 4",
		"test 5",
		"test 6",
	}

	ctx := context.Background()
	storageConfig := storage.NewConfig("standalone", "./tmp/test")
	store := storage.NewFilesystemStorage(storageConfig)
	repo := NewPeerRepository(store)

	// For logging to test from within functions
	ctx = context.WithValue(ctx, 999, test)
	// Use this to get the test value from within non-test code.
	// testValue := ctx.Value(999)
	// test, ok := testValue.(*testing.T)
	// if ok {
	// test.Logf("Test Debug Message")
	// }

	repo.Clear(ctx)

	// Load
	if err := repo.Load(ctx); err != nil {
		test.Errorf("Failed to load repo : %v", err)
	}

	// Add
	for _, address := range addresses {
		added, err := repo.Add(ctx, address)
		if err != nil {
			test.Errorf("Failed to add address : %v", err)
		}
		if !added {
			test.Errorf("Didn't add address : %s", address)
		}
	}

	// Get min score 0
	peers, err := repo.Get(ctx, 0)
	if err != nil {
		test.Errorf("Failed to get addresses : %v", err)
	}

	for _, address := range addresses {
		found := false
		for _, peer := range peers {
			if peer.Address == address {
				test.Logf("Found address : %s", address)
				found = true
				break
			}
		}

		if !found {
			test.Errorf("Failed to find address : %s", address)
		}
	}

	// Get min score 0
	peers, err = repo.Get(ctx, 1)
	if err != nil {
		test.Errorf("Failed to get addresses : %v", err)
	}

	if len(peers) > 0 {
		test.Errorf("Pulled high score peers")
	}

	// Save
	test.Logf("Saving")
	if err := repo.Save(ctx); err != nil {
		test.Errorf("Failed to save repo : %v", err)
	}

	// Load
	test.Logf("Reloading")
	if err := repo.Load(ctx); err != nil {
		test.Errorf("Failed to re-load repo : %v", err)
	}

	// Get min score 0
	peers, err = repo.Get(ctx, 0)
	if err != nil {
		test.Errorf("Failed to get addresses : %v", err)
	}

	for _, address := range addresses {
		found := false
		for _, peer := range peers {
			if peer.Address == address {
				test.Logf("Found address : %s", address)
				found = true
				break
			}
		}

		if !found {
			test.Errorf("Failed to find address : %s", address)
		}
	}

	// Get min score 0
	peers, err = repo.Get(ctx, 1)
	if err != nil {
		test.Errorf("Failed to get addresses : %v", err)
	}

	if len(peers) > 0 {
		test.Errorf("Pulled high score peers")
	}
}
