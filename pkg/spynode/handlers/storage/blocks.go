package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

const (
	blocksPerKey = 1000 // Number of block hashes stored in each key
)

// Block represents a block on the blockchain.
type Block struct {
	Hash   chainhash.Hash
	Height int
}

// BlockRepository is used for managing Block data.
type BlockRepository struct {
	store      storage.Storage
	height     int                    // Height of the latest block
	lastHashes []chainhash.Hash       // Hashes in the latest key/file
	heights    map[chainhash.Hash]int // Lookup of block height by hash
	mutex      sync.Mutex
}

// NewBlockRepository returns a new BlockRepository.
func NewBlockRepository(store storage.Storage) *BlockRepository {
	result := BlockRepository{
		store:      store,
		height:     -1,
		lastHashes: make([]chainhash.Hash, 0, blocksPerKey),
		heights:    make(map[chainhash.Hash]int),
	}
	return &result
}

// Initialize as empty.
func (repo *BlockRepository) Initialize(ctx context.Context, genesisHash chainhash.Hash) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	repo.lastHashes = make([]chainhash.Hash, 0, blocksPerKey)
	repo.lastHashes = append(repo.lastHashes, genesisHash)
	repo.height = 0
	repo.heights[genesisHash] = repo.height
	return nil
}

// Load from storage
func (repo *BlockRepository) Load(ctx context.Context) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Clear
	repo.height = -1
	repo.heights = make(map[chainhash.Hash]int)

	// Build hash height map from genesis and load lastHashes
	previousFileSize := -1
	filesLoaded := 0
	for {
		hashes, err := repo.read(ctx, filesLoaded*blocksPerKey)
		if err == storage.ErrNotFound {
			break
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to open block file : %s",
				repo.buildPath(repo.height)))
		}
		if len(hashes) == 0 {
			break
		}

		if previousFileSize != -1 && previousFileSize != blocksPerKey {
			return errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", previousFileSize,
				repo.buildPath(repo.height-blocksPerKey)))
		}

		// Add this set of hashes to the heights map
		for i, hash := range hashes {
			repo.heights[hash] = repo.height + i + 1
		}

		previousFileSize = len(hashes)

		if filesLoaded == 0 {
			repo.height = len(hashes) - 1 // Account for genesis block 0
		} else {
			repo.height += len(hashes)
		}

		repo.lastHashes = hashes
		filesLoaded++
	}

	if filesLoaded == 0 {
		// Add genesis
		hash, err := chainhash.NewHashFromStr("000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f")
		if err != nil {
			return errors.Wrap(err, "Failed to create genesis hash")
		}
		repo.lastHashes = append(repo.lastHashes, *hash)
		repo.height = 0
		repo.heights[*hash] = repo.height
		logger.Log(ctx, logger.Verbose, "Added genesis block")
	}

	return nil
}

// Adds a block hash
func (repo *BlockRepository) Add(ctx context.Context, hash chainhash.Hash) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if len(repo.lastHashes) == blocksPerKey {
		// Save latest key
		if err := repo.save(ctx); err != nil {
			return err
		}

		// Start next key
		repo.lastHashes = make([]chainhash.Hash, 0, blocksPerKey)
	}

	repo.lastHashes = append(repo.lastHashes, hash)
	repo.height++
	repo.heights[hash] = repo.height
	return nil
}

// Return the block hash for the specified height
func (repo *BlockRepository) LastHeight() int {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.height
}

// Return the block hash for the specified height
func (repo *BlockRepository) LastHash() *chainhash.Hash {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return &repo.lastHashes[len(repo.lastHashes)-1]
}

func (repo *BlockRepository) Contains(hash chainhash.Hash) bool {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	_, exists := repo.heights[hash]
	return exists
}

// Returns:
//   int - height of hash if it exists
//   bool - true if the hash exists
func (repo *BlockRepository) Height(hash *chainhash.Hash) (int, bool) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	result, exists := repo.heights[*hash]
	return result, exists
}

// Return the block hash for the specified height
func (repo *BlockRepository) Hash(ctx context.Context, height int) (*chainhash.Hash, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.getHash(ctx, height)
}

// This function is internal and doesn't lock the mutex so it can be internally without double locking.
func (repo *BlockRepository) getHash(ctx context.Context, height int) (*chainhash.Hash, error) {
	if height > repo.height {
		return nil, nil // We don't know the hash for that height yet
	}

	if repo.height-height < len(repo.lastHashes) {
		// This height is in the lastHashes set
		return &repo.lastHashes[len(repo.lastHashes)-1-(repo.height-height)], nil
	}

	// Read from storage
	hashes, err := repo.read(ctx, height)
	if err != nil {
		return nil, nil
	}

	if len(hashes) != blocksPerKey {
		// This should only be reached on files that are not the latest, and they should all be full
		return nil, errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", len(hashes),
			repo.buildPath(repo.height-blocksPerKey)))
	}

	offset := height % blocksPerKey
	return &hashes[offset], nil
}

// Revert block repository to the specified height. Saves after
func (repo *BlockRepository) Revert(ctx context.Context, height int) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if height > repo.height {
		return errors.New(fmt.Sprintf("Revert height %d above current height %d", height, repo.height))
	}

	// Revert heights map
	for removeHeight := repo.height; removeHeight > height; removeHeight-- {
		hash, err := repo.getHash(ctx, removeHeight)
		if err != nil {
			return errors.Wrap(err, "Failed to revert block heights map")
		}
		delete(repo.heights, *hash)
	}

	// Height of last block of latest full file
	fullFileEndHeight := (((repo.height) / blocksPerKey) * blocksPerKey) - 1
	revertedHeight := fullFileEndHeight

	// Remove any files that need completely removed.
	for ; revertedHeight >= height; revertedHeight -= blocksPerKey {
		path := repo.buildPath(revertedHeight + blocksPerKey)
		if err := repo.store.Remove(ctx, path); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to remove block file for revert : %s", path))
		}
	}

	// Partially revert last remaining file if necessary. Otherwise just load it into cache.
	path := repo.buildPath(revertedHeight + blocksPerKey)
	newCount := height - revertedHeight
	data, err := repo.store.Read(ctx, path)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to read block file to truncate : %s", path))
	}

	if newCount < blocksPerKey {
		data = data[:chainhash.HashSize*newCount] // Truncate data

		// Re-write file with truncated data
		if err := repo.store.Write(ctx, path, data, nil); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to re-write block file to truncate : %s", path))
		}
	}

	// Cache needs to be reset with last file's data.
	repo.lastHashes = make([]chainhash.Hash, 0, blocksPerKey)
	endOffset := len(data)
	for offset := 0; offset < endOffset; offset += chainhash.HashSize {
		newhash, err := chainhash.NewHash(data[offset : offset+chainhash.HashSize])
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to parse latest block data during truncate : %s", path))
		}
		repo.lastHashes = append(repo.lastHashes, *newhash)
	}
	repo.height = height
	return nil
}

// Saves the latest key of block hashes
func (repo *BlockRepository) Save(ctx context.Context) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.save(ctx)
}

// This function is internal and doesn't lock the mutex so it can be internally without double locking.
func (repo *BlockRepository) save(ctx context.Context) error {
	// Create contiguous byte slice
	data := make([]byte, chainhash.HashSize*len(repo.lastHashes))
	for i, hash := range repo.lastHashes {
		copy(data[i*chainhash.HashSize:(i*chainhash.HashSize)+chainhash.HashSize], hash[:])
	}

	path := repo.buildPath(repo.height)

	err := repo.store.Write(ctx, path, data, nil)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to write block file %s", path))
	}

	return nil
}

// Read reads a key.
func (repo *BlockRepository) read(ctx context.Context, height int) ([]chainhash.Hash, error) {
	data, err := repo.store.Read(ctx, repo.buildPath(height))
	if err != nil {
		return nil, err
	}

	// Parse hashes from key
	hashes := make([]chainhash.Hash, 0, blocksPerKey)
	endOffset := len(data)
	for offset := 0; offset < endOffset; offset += chainhash.HashSize {
		newhash, err := chainhash.NewHash(data[offset : offset+chainhash.HashSize])
		if err != nil {
			return hashes, err
		}
		hashes = append(hashes, *newhash)
	}

	return hashes, nil
}

func (repo *BlockRepository) buildPath(height int) string {
	return fmt.Sprintf("blocks/%08x", height/blocksPerKey)
}
