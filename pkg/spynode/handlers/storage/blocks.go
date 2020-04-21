package storage

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	blocksPerKey = 1000 // Number of block hashes stored in each key
)

// Block represents a block on the blockchain.
type Block struct {
	Hash   bitcoin.Hash32
	Height int
}

// BlockRepository is used for managing Block data.
type BlockRepository struct {
	config      *data.Config
	store       storage.Storage
	height      int                    // Height of the latest block
	lastHeaders []wire.BlockHeader     // Hashes in the latest key/file
	heights     map[bitcoin.Hash32]int // Lookup of block height by hash
	mutex       sync.Mutex
}

// NewBlockRepository returns a new BlockRepository.
func NewBlockRepository(config *data.Config, store storage.Storage) *BlockRepository {
	result := BlockRepository{
		config:      config,
		store:       store,
		height:      -1,
		lastHeaders: make([]wire.BlockHeader, 0, blocksPerKey),
		heights:     make(map[bitcoin.Hash32]int),
	}
	return &result
}

// Initialize as empty.
func (repo *BlockRepository) Initialize(ctx context.Context, genesisTime uint32) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	repo.lastHeaders = make([]wire.BlockHeader, 0, blocksPerKey)
	header := wire.BlockHeader{Timestamp: time.Unix(int64(genesisTime), 0)}
	repo.lastHeaders = append(repo.lastHeaders, header)
	repo.height = 0
	repo.heights[*header.BlockHash()] = repo.height
	return nil
}

// Load from storage
func (repo *BlockRepository) Load(ctx context.Context) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Clear
	repo.height = -1
	repo.heights = make(map[bitcoin.Hash32]int)

	// Build hash height map from genesis and load lastHeaders
	previousFileSize := -1
	filesLoaded := 0
	for {
		headers, err := repo.read(ctx, filesLoaded*blocksPerKey)
		if err == storage.ErrNotFound {
			break
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to open block file : %s",
				repo.buildPath(repo.height)))
		}
		if len(headers) == 0 {
			break
		}

		if previousFileSize != -1 && previousFileSize != blocksPerKey {
			return errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", previousFileSize,
				repo.buildPath(repo.height-blocksPerKey)))
		}

		// Add this set of headers to the heights map
		for i, h := range headers {
			repo.heights[*h.BlockHash()] = repo.height + i + 1
		}

		previousFileSize = len(headers)

		if filesLoaded == 0 {
			repo.height = len(headers) - 1 // Account for genesis block 0
		} else {
			repo.height += len(headers)
		}

		repo.lastHeaders = headers
		filesLoaded++
	}

	if filesLoaded == 0 {
		// Add genesis
		logger.Verbose(ctx, "Adding %s genesis block", bitcoin.NetworkName(repo.config.Net))
		if repo.config.Net == bitcoin.MainNet {
			// Hash "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
			prevhash, err := bitcoin.NewHash32FromStr("0000000000000000000000000000000000000000000000000000000000000000")
			if err != nil {
				return errors.Wrap(err, "Failed to create genesis prev hash")
			}
			merklehash, err := bitcoin.NewHash32FromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
			if err != nil {
				return errors.Wrap(err, "Failed to create genesis merkle hash")
			}
			genesisHeader := wire.BlockHeader{
				Version:    1,
				PrevBlock:  *prevhash,
				MerkleRoot: *merklehash,
				Timestamp:  time.Unix(1231006505, 0),
				Bits:       0x1d00ffff,
				Nonce:      2083236893,
			}
			repo.lastHeaders = append(repo.lastHeaders, genesisHeader)
			repo.height = 0
			repo.heights[*genesisHeader.BlockHash()] = repo.height
			logger.Verbose(ctx, "Added genesis block : %s", genesisHeader.BlockHash().String())
		} else { // testnet
			// Hash "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
			prevhash, err := bitcoin.NewHash32FromStr("0000000000000000000000000000000000000000000000000000000000000000")
			if err != nil {
				return errors.Wrap(err, "Failed to create genesis prev hash")
			}
			merklehash, err := bitcoin.NewHash32FromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")
			if err != nil {
				return errors.Wrap(err, "Failed to create genesis merkle hash")
			}
			genesisHeader := wire.BlockHeader{
				Version:    1,
				PrevBlock:  *prevhash,
				MerkleRoot: *merklehash,
				Timestamp:  time.Unix(1296688602, 0),
				Bits:       0x1d00ffff,
				Nonce:      414098458,
			}
			repo.lastHeaders = append(repo.lastHeaders, genesisHeader)
			repo.height = 0
			repo.heights[*genesisHeader.BlockHash()] = repo.height
			logger.Verbose(ctx, "Added testnet genesis block : %s", genesisHeader.BlockHash().String())
		}
	}

	return nil
}

// Adds a block header
func (repo *BlockRepository) Add(ctx context.Context, header *wire.BlockHeader) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if len(repo.lastHeaders) == blocksPerKey {
		// Save latest key
		if err := repo.save(ctx); err != nil {
			return errors.Wrap(err, "Failed to save")
		}

		// Start next key
		repo.lastHeaders = make([]wire.BlockHeader, 0, blocksPerKey)
	}

	repo.lastHeaders = append(repo.lastHeaders, *header)
	repo.height++
	repo.heights[*header.BlockHash()] = repo.height
	return nil
}

// Return the block hash for the specified height
func (repo *BlockRepository) LastHeight() int {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.height
}

// Return the block hash for the specified height
func (repo *BlockRepository) LastHash() *bitcoin.Hash32 {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	result := repo.lastHeaders[len(repo.lastHeaders)-1].BlockHash()
	return result
}

func (repo *BlockRepository) Contains(hash *bitcoin.Hash32) bool {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	_, exists := repo.heights[*hash]
	return exists
}

// Returns:
//   int - height of hash if it exists
//   bool - true if the hash exists
func (repo *BlockRepository) Height(hash *bitcoin.Hash32) (int, bool) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	result, exists := repo.heights[*hash]
	return result, exists
}

// Return the block hash for the specified height
func (repo *BlockRepository) Hash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.getHash(ctx, height)
}

// This function is internal and doesn't lock the mutex so it can be internally without double locking.
func (repo *BlockRepository) getHash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	if height > repo.height {
		return nil, errors.New("Hash height beyond tip") // We don't know the hash for that height yet
	}

	if repo.height-height < len(repo.lastHeaders) {
		// This height is in the lastHeaders set
		result := repo.lastHeaders[len(repo.lastHeaders)-1-(repo.height-height)].BlockHash()
		return result, nil
	}

	// Read from storage
	headers, err := repo.read(ctx, height)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to read block file for height %d", height))
	}

	if len(headers) != blocksPerKey {
		// This should only be reached on files that are not the latest, and they should all be full
		return nil, errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", len(headers),
			repo.buildPath(repo.height-blocksPerKey)))
	}

	offset := height % blocksPerKey
	result := headers[offset].BlockHash()
	return result, nil
}

// Return the block time for the specified height
func (repo *BlockRepository) Time(ctx context.Context, height int) (uint32, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.getTime(ctx, height)
}

// This function is internal and doesn't lock the mutex so it can be internally without double locking.
func (repo *BlockRepository) getTime(ctx context.Context, height int) (uint32, error) {
	if height > repo.height {
		return 0, nil // We don't know the hash for that height yet
	}

	if repo.height-height < len(repo.lastHeaders) {
		// This height is in the lastHeaders set
		return uint32(repo.lastHeaders[len(repo.lastHeaders)-1-(repo.height-height)].Timestamp.Unix()), nil
	}

	// Read from storage
	headers, err := repo.read(ctx, height)
	if err != nil {
		return 0, nil
	}

	if len(headers) != blocksPerKey {
		// This should only be reached on files that are not the latest, and they should all be full
		return 0, errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", len(headers),
			repo.buildPath(repo.height-blocksPerKey)))
	}

	offset := height % blocksPerKey
	return uint32(headers[offset].Timestamp.Unix()), nil
}

// Return the block header for the specified height
func (repo *BlockRepository) Header(ctx context.Context, height int) (*wire.BlockHeader, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return repo.getHeader(ctx, height)
}

// This function is internal and doesn't lock the mutex so it can be internally without double locking.
func (repo *BlockRepository) getHeader(ctx context.Context, height int) (*wire.BlockHeader, error) {
	if height > repo.height {
		return nil, errors.New("Hash height beyond tip") // We don't know the hash for that height yet
	}

	if repo.height-height < len(repo.lastHeaders) {
		// This height is in the lastHeaders set
		result := repo.lastHeaders[len(repo.lastHeaders)-1-(repo.height-height)]
		return &result, nil
	}

	// Read from storage
	headers, err := repo.read(ctx, height)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to read block file for height %d", height))
	}

	if len(headers) != blocksPerKey {
		// This should only be reached on files that are not the latest, and they should all be full
		return nil, errors.New(fmt.Sprintf("Invalid block file (count %d) : %s", len(headers),
			repo.buildPath(repo.height-blocksPerKey)))
	}

	offset := height % blocksPerKey
	result := headers[offset]
	return &result, nil
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

	if newCount < blocksPerKey && len(data) > wire.MaxBlockHeaderPayload*newCount {
		data = data[:wire.MaxBlockHeaderPayload*newCount] // Truncate data

		// Re-write file with truncated data
		if err := repo.store.Write(ctx, path, data, nil); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to re-write block file to truncate : %s", path))
		}
	}

	// Cache needs to be reset with last file's data.
	repo.lastHeaders = make([]wire.BlockHeader, 0, blocksPerKey)
	buf := bytes.NewBuffer(data)
	header := wire.BlockHeader{}
	for buf.Len() > 0 {
		err := header.Deserialize(buf)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to parse latest block data during truncate : %s", path))
		}
		repo.lastHeaders = append(repo.lastHeaders, header)
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
	data := make([]byte, 0, wire.MaxBlockHeaderPayload*len(repo.lastHeaders))
	buf := bytes.NewBuffer(data)
	for _, header := range repo.lastHeaders {
		err := header.Serialize(buf)
		if err != nil {
			return errors.Wrap(err, "Failed to write header")
		}
	}

	err := repo.store.Write(ctx, repo.buildPath(repo.height), buf.Bytes(), nil)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to write block file %s", repo.buildPath(repo.height)))
	}

	return nil
}

// Read reads a key.
func (repo *BlockRepository) read(ctx context.Context, height int) ([]wire.BlockHeader, error) {
	data, err := repo.store.Read(ctx, repo.buildPath(height))
	if err != nil {
		return nil, err
	}

	// Parse headers from key
	headers := make([]wire.BlockHeader, 0, blocksPerKey)
	buf := bytes.NewBuffer(data)
	header := wire.BlockHeader{}
	for buf.Len() > 0 {
		err := header.Deserialize(buf)
		if err != nil {
			return headers, err
		}
		headers = append(headers, header)
	}

	return headers, nil
}

func (repo *BlockRepository) buildPath(height int) string {
	return fmt.Sprintf("spynode/blocks/%08x", height/blocksPerKey)
}
