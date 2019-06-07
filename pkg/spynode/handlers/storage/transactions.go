package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

const (
	unconfirmedPath = "spynode/txs/unconfirmed"
)

// TxRepository is used for managing which txs for each block are "relevant" and which have been
//   sent to listeners.
type TxRepository struct {
	store           storage.Storage
	unconfirmed     map[chainhash.Hash]*unconfirmedTx
	unconfirmedLock sync.Mutex
	blockLock       sync.Mutex
}

// NewTxRepository returns a new TxRepository.
func NewTxRepository(store storage.Storage) *TxRepository {
	result := TxRepository{
		store:       store,
		unconfirmed: make(map[chainhash.Hash]*unconfirmedTx),
	}
	return &result
}

func (repo *TxRepository) Load(ctx context.Context) error {
	repo.unconfirmed = make(map[chainhash.Hash]*unconfirmedTx)

	data, err := repo.store.Read(ctx, unconfirmedPath)
	if err == storage.ErrNotFound {
		logger.Verbose(ctx, "No unconfirmed txs to load")
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		logger.Verbose(ctx, "No unconfirmed txs to load")
		return nil // Empty
	}

	reader := bytes.NewReader(data)

	var version uint8
	err = binary.Read(reader, binary.LittleEndian, &version)
	if err != nil {
		return err
	}

	for {
		txid, tx, err := readUnconfirmedTx(reader, version)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		repo.unconfirmed[txid] = tx
	}

	logger.Verbose(ctx, "Loaded %d unconfirmed txs", len(repo.unconfirmed))
	return nil
}

func (repo *TxRepository) Save(ctx context.Context) error {
	logger.Verbose(ctx, "Saving %d unconfirmed txs", len(repo.unconfirmed))
	repo.unconfirmedLock.Lock()
	defer repo.unconfirmedLock.Unlock()
	return repo.save(ctx)
}

// Add a "relevant" tx id for a specified block
// Height of -1 means unconfirmed
// Returns true if the txid was not already in the repo for the specified height, and was added
func (repo *TxRepository) Add(ctx context.Context, txid chainhash.Hash, trusted, safe bool, height int) (bool, error) {
	if height == -1 {
		repo.unconfirmedLock.Lock()
		defer repo.unconfirmedLock.Unlock()
		if tx, exists := repo.unconfirmed[txid]; exists {
			if trusted {
				tx.trusted = true
			}
			if safe {
				tx.safe = true
			}
			return false, nil
		}

		repo.unconfirmed[txid] = newUnconfirmedTx(trusted, safe)
		return true, nil
	}

	path := repo.buildPath(height)

	// Get current tx data for block
	repo.blockLock.Lock()
	defer repo.blockLock.Unlock()
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

// Remove a "relevant" tx id for a specified block
// Height of -1 means unconfirmed
// Returns true if the txid was removed
func (repo *TxRepository) Remove(ctx context.Context, txid chainhash.Hash, height int) (bool, error) {
	if height == -1 {
		repo.unconfirmedLock.Lock()
		defer repo.unconfirmedLock.Unlock()
		if _, exists := repo.unconfirmed[txid]; exists {
			delete(repo.unconfirmed, txid)
			return true, nil
		}
		return false, nil
	}

	path := repo.buildPath(height)

	// Get current tx data for block
	repo.blockLock.Lock()
	defer repo.blockLock.Unlock()
	data, err := repo.store.Read(ctx, path)
	if err == storage.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check for match to remove
	for i := 0; i < len(data); i += chainhash.HashSize {
		if bytes.Equal(data[i:i+chainhash.HashSize], txid[:]) {
			data = append(data[:i], data[i+chainhash.HashSize:]...)
			return true, repo.store.Write(ctx, path, data, nil)
		}
	}

	return false, nil
}

// Contains returns true if the tx id is in the specified block
// Height of -1 means unconfirmed
func (repo *TxRepository) Contains(ctx context.Context, txid chainhash.Hash, height int) (bool, error) {
	if height == -1 {
		repo.unconfirmedLock.Lock()
		defer repo.unconfirmedLock.Unlock()
		_, exists := repo.unconfirmed[txid]
		return exists, nil
	}

	path := repo.buildPath(height)

	// Get current tx data for block
	repo.blockLock.Lock()
	defer repo.blockLock.Unlock()
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

// GetBlock returns all "relevant" tx ids in a specified block
// Locks the tx repo.
// RemoveBlock, SetBlock, or ReleaseBlock must be called after this to release the lock
// Height of -1 means unconfirmed
func (repo *TxRepository) GetUnconfirmed(ctx context.Context) ([]chainhash.Hash, error) {
	repo.unconfirmedLock.Lock()
	result := make([]chainhash.Hash, 0, len(repo.unconfirmed))
	for hash, _ := range repo.unconfirmed {
		result = append(result, hash)
	}
	return result, nil
}

// GetBlock returns all "relevant" tx ids in a specified block
// Locks the tx repo.
// RemoveBlock, SetBlock, or ReleaseBlock must be called after this to release the lock
// Height of -1 is invalid for this function
func (repo *TxRepository) GetBlock(ctx context.Context, height int) ([]chainhash.Hash, error) {
	logger.Debug(ctx, "Get Block %d", height)
	repo.blockLock.Lock()
	hashes, err := repo.readBlock(ctx, height)
	if err != nil {
		repo.blockLock.Unlock()
		return nil, err
	}

	return hashes, nil
}

// FinalizeBlock updates all "relevant" tx ids in a specified block and unconfirmed
// Must only be called after GetBlock
// Releases the lock made in GetBlock
func (repo *TxRepository) FinalizeUnconfirmed(ctx context.Context, unconfirmed []chainhash.Hash) error {
	defer repo.unconfirmedLock.Unlock()

	// Update unconfirmed
	newUnconfirmed := make(map[chainhash.Hash]*unconfirmedTx)
	for _, hash := range unconfirmed {
		if tx, exists := repo.unconfirmed[hash]; exists {
			newUnconfirmed[hash] = tx
		} else {
			newUnconfirmed[hash] = newUnconfirmedTx(true, false)
		}
	}
	repo.unconfirmed = newUnconfirmed
	return repo.save(ctx)
}

// Removes all "relevant" tx ids in a specified block and releases lock
// Must only be called after GetBlock
// Releases the lock made in GetBlock
// Height is invalid for this function
func (repo *TxRepository) RemoveBlock(ctx context.Context, height int) error {
	logger.Debug(ctx, "Remove Block %d", height)
	defer repo.blockLock.Unlock()

	err := repo.store.Remove(ctx, repo.buildPath(height))
	if err == storage.ErrNotFound {
		return nil
	}
	return err
}

// ReleaseBlock releases the lock from GetBlock
// Must only be called after GetBlock
func (repo *TxRepository) ReleaseUnconfirmed(ctx context.Context) error {
	repo.unconfirmedLock.Unlock()
	return nil
}

// ReleaseBlock releases the lock from GetBlock
// Must only be called after GetBlock
func (repo *TxRepository) ReleaseBlock(ctx context.Context, height int) error {
	logger.Debug(ctx, "Release Block %d", height)
	repo.blockLock.Unlock()
	return nil
}

// SetBlock sets tx ids in a specified block
// Height of -1 means unconfirmed
func (repo *TxRepository) SetBlock(ctx context.Context, txids []chainhash.Hash, height int) error {
	if height == -1 {
		repo.unconfirmedLock.Lock()
		defer repo.unconfirmedLock.Unlock()
		newUnconfirmed := make(map[chainhash.Hash]*unconfirmedTx)
		for _, hash := range txids {
			if tx, exists := repo.unconfirmed[hash]; exists {
				newUnconfirmed[hash] = tx
			} else {
				newUnconfirmed[hash] = newUnconfirmedTx(true, false)
			}
		}
		repo.unconfirmed = newUnconfirmed
		return nil
	}

	repo.blockLock.Lock()
	defer repo.blockLock.Unlock()
	if len(txids) > 0 {
		if err := repo.writeBlock(ctx, txids, height); err != nil {
			return err
		}
	} else {
		if err := repo.store.Remove(ctx, repo.buildPath(height)); err != nil && err != storage.ErrNotFound {
			return err
		}
	}

	return nil
}

// ClearBlock clears all "relevant" tx ids in a specified block
// Height of -1 means unconfirmed
func (repo *TxRepository) ClearBlock(ctx context.Context, height int) error {
	if height == -1 {
		repo.unconfirmedLock.Lock()
		defer repo.unconfirmedLock.Unlock()
		repo.unconfirmed = make(map[chainhash.Hash]*unconfirmedTx)
		return nil
	}

	repo.blockLock.Lock()
	defer repo.blockLock.Unlock()
	err := repo.store.Remove(ctx, repo.buildPath(height))
	if err == storage.ErrNotFound {
		return nil
	}
	return err
}

func (repo *TxRepository) writeBlock(ctx context.Context, txids []chainhash.Hash, height int) error {
	if height == -1 {
		return errors.New("Can't write unconfirmed with this method")
	}

	data := make([]byte, 0, len(txids)*chainhash.HashSize)

	// Write all hashes to data
	for _, txid := range txids {
		data = append(data, txid[:]...)
	}

	return repo.store.Write(ctx, repo.buildPath(height), data, nil)
}

func (repo *TxRepository) readBlock(ctx context.Context, height int) ([]chainhash.Hash, error) {
	if height == -1 {
		return nil, errors.New("Can't read unconfirmed with this method")
	}

	data, err := repo.store.Read(ctx, repo.buildPath(height))
	if err == storage.ErrNotFound {
		return make([]chainhash.Hash, 0), nil
	}
	if err != nil {
		return nil, err
	}

	// Parse hashes from data
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

func (repo *TxRepository) buildPath(height int) string {
	return fmt.Sprintf("spynode/txs/%08x", height)
}

func (repo *TxRepository) save(ctx context.Context) error {
	if len(repo.unconfirmed) == 0 {
		if err := repo.store.Remove(ctx, unconfirmedPath); err != nil && err != storage.ErrNotFound {
			return err
		}
		return nil
	}

	data := make([]byte, 0, (unconfirmedTxSize*len(repo.unconfirmed))+1)
	writer := bytes.NewBuffer(data)
	version := uint8(0)
	err := binary.Write(writer, binary.LittleEndian, version) // Milliseconds
	if err != nil {
		return err
	}
	for hash, tx := range repo.unconfirmed {
		err = tx.Write(writer, &hash)
		if err != nil {
			return err
		}
	}

	return repo.store.Write(ctx, unconfirmedPath, writer.Bytes(), nil)
}
