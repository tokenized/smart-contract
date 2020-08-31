package blocks

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/storage"
	"github.com/tokenized/pkg/wire"

	"github.com/pkg/errors"
)

const (
	// blocksPerKey is the number of block hashes stored in each file.
	blocksPerKey = 1000

	// blockHeaderSize is the size in bytes of a block header.
	blockHeaderSize = 80
)

// Blocks uses spynode storage to fetch block headers and hashes.
type Blocks struct {
	store storage.Storage
}

// NewBlocks creates a new Blocks.
func NewBlocks(st storage.Storage) *Blocks {
	return &Blocks{
		store: st,
	}
}

// Hash returns the hash of the block header at the specified block height.
func (b *Blocks) Hash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	header, err := b.Header(ctx, height)
	if err != nil {
		return nil, err
	}

	return header.BlockHash(), nil
}

// Header returns the block header at the specified block height.
func (b *Blocks) Header(ctx context.Context, height int) (*wire.BlockHeader, error) {
	// Read from storage
	path := fmt.Sprintf("spynode/blocks/%08x", height/blocksPerKey)
	data, err := b.store.Read(ctx, path)
	if err != nil {
		return nil, err
	}

	offset := height % blocksPerKey
	byteOffset := offset * blockHeaderSize

	if byteOffset+blockHeaderSize > len(data) {
		return nil, errors.New("Not available")
	}

	// Parse header from data
	buf := bytes.NewBuffer(data[byteOffset:])
	var result wire.BlockHeader
	if err := result.Deserialize(buf); err != nil {
		return nil, errors.Wrap(err, "deserialize header")
	}

	return &result, nil
}
