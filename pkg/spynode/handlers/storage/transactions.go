package storage

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// TxRepository is used for managing which txs for each block are "relevant" and which have been
//   sent to listeners.
type TxRepository struct {
	store storage.Storage
	mutex sync.Mutex
}

// NewTxRepository returns a new TxRepository.
func NewTxRepository(store storage.Storage) *TxRepository {
	result := TxRepository{
		store: store,
	}
	return &result
}

// Adds a "relevant" tx id for a specified block
// Height of -1 means unconfirmed
// Returns true if the txid was not already in the repo for the specified height
func (repo *TxRepository) Add(ctx context.Context, txid chainhash.Hash, height int) (bool, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	path := repo.buildPath(height)

	// Get current tx data for block
	data, err := repo.store.Read(ctx, path)
	if err == storage.ErrNotFound {
		// Create new tx block file with only one hash
		return true, repo.store.Write(ctx, path, txid[:], nil)
	}
	if err != nil {
		return false, err
	}

	// Check for already existing
	for i := 0; i < len(data); i += chainhash.HashSize {
		if bytes.Equal(data[i:i+chainhash.HashSize], txid[:]) {
			return false, nil
		}
	}

	// Append txid to end of file
	newData := make([]byte, len(data)+chainhash.HashSize)
	copy(newData, data) // Copy in previous data
	copy(newData[len(data):], txid[:])
	return true, repo.store.Write(ctx, path, newData, nil)
}

// Returns true if the tx id is in the specified block
// Height of -1 means unconfirmed
func (repo *TxRepository) Contains(ctx context.Context, txid chainhash.Hash, height int) (bool, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	path := repo.buildPath(height)

	// Get current tx data for block
	data, err := repo.store.Read(ctx, path)
	if err == storage.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check for already existing
	for i := 0; i < len(data); i += chainhash.HashSize {
		if bytes.Equal(data[i:i+chainhash.HashSize], txid[:]) {
			return true, nil
		}
	}

	return false, nil
}

// Returns all "relevant" tx ids in a specified block
// RemoveBlock, SetBlock, or ReleaseBlock must be called after this to release the lock
// Height of -1 means unconfirmed
func (repo *TxRepository) GetBlock(ctx context.Context, height int) ([]chainhash.Hash, error) {
	repo.mutex.Lock()

	data, err := repo.store.Read(ctx, repo.buildPath(height))
	if err == storage.ErrNotFound {
		return make([]chainhash.Hash, 0), nil
	}
	if err != nil {
		return nil, err
	}

	// Parse hashes from key
	hashes := make([]chainhash.Hash, 0, 100)
	endOffset := len(data)
	for offset := 0; offset < endOffset; offset += chainhash.HashSize {
		if offset+chainhash.HashSize > endOffset {
			return make([]chainhash.Hash, 0), errors.New(fmt.Sprintf("TX file %08x has invalid size : %d", height, len(data)))
		}
		newhash, err := chainhash.NewHash(data[offset : offset+chainhash.HashSize])
		if err != nil {
			return hashes, err
		}
		hashes = append(hashes, *newhash)
	}

	return hashes, nil
}

// Rewrites all "relevant" tx ids in a specified block and unconfirmed
// Must only be called after GetBlock
// Height of -1 means unconfirmed
func (repo *TxRepository) FinalizeBlock(ctx context.Context, unconfirmed []chainhash.Hash, txids []chainhash.Hash, height int) error {
	if err := repo.writeBlock(ctx, txids, height); err != nil {
		repo.mutex.Unlock()
		return err
	}

	if err := repo.writeBlock(ctx, unconfirmed, -1); err != nil {
		repo.mutex.Unlock()
		return err
	}

	repo.mutex.Unlock()
	return nil
}

func (repo *TxRepository) writeBlock(ctx context.Context, txids []chainhash.Hash, height int) error {
	data := make([]byte, 0, len(txids)*chainhash.HashSize)

	// Write all hashes to data
	for _, txid := range txids {
		data = append(data, txid[:]...)
	}

	return repo.store.Write(ctx, repo.buildPath(height), data, nil)
}

// Removes all "relevant" tx ids in a specified block and releases lock
// Must only be called after GetBlock
// Height of -1 means unconfirmed
func (repo *TxRepository) RemoveBlock(ctx context.Context, height int) error {
	err := repo.store.Remove(ctx, repo.buildPath(height))
	repo.mutex.Unlock()
	return err
}

// Releases lock
// Must only be called after GetBlock
// Height of -1 means unconfirmed
func (repo *TxRepository) ReleaseBlock(ctx context.Context, height int) error {
	repo.mutex.Unlock()
	return nil
}

// Clears all "relevant" tx ids in a specified block
// Height of -1 means unconfirmed
func (repo *TxRepository) ClearBlock(ctx context.Context, height int) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.store.Remove(ctx, repo.buildPath(height))
}

func (repo *TxRepository) buildPath(height int) string {
	if height == -1 {
		return "txs/unconfirmed"
	}
	return fmt.Sprintf("txs/%08x", height)
}
