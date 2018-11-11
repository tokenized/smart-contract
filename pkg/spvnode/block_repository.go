package spvnode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/storage"
)

// ErrBlockNotFound is returns when a requested item is not found.
var ErrBlockNotFound = errors.New("Block not found")

// Block represents a block on the blockchain.
type Block struct {
	Hash      string `json:"hash"`
	PrevBlock string `json:"prev_block"`
	Height    int32  `json:"height"`
}

// BlockRepository is used for managing Block data.
type BlockRepository struct {
	Storage storage.Storage
}

// NewBlockRepository returns a new BlockRepository.
func NewBlockRepository(store storage.Storage) BlockRepository {
	return BlockRepository{
		Storage: store,
	}
}

// All returns all items.
func (r BlockRepository) All(ctx context.Context) ([]Block, error) {
	query := map[string]string{
		"path": "blocks",
	}

	data, err := r.Storage.Search(ctx, query)

	if err != nil {
		return nil, err
	}

	blocks := []Block{}

	for _, b := range data {
		block := Block{}

		if err := json.Unmarshal(b, &block); err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Write stores a Block.
func (r BlockRepository) Write(ctx context.Context, c Block) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := r.buildPath(c.Hash)

	return r.Storage.Write(ctx, key, b, nil)
}

// Read reads a Block.
func (r BlockRepository) Read(ctx context.Context, id string) (*Block, error) {
	key := r.buildPath(id)

	b, err := r.Storage.Read(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = ErrBlockNotFound
		}

		return nil, err
	}

	// we have found a matching key
	c := Block{}

	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// Remove removes a Block from storage.
func (r BlockRepository) Remove(ctx context.Context, b Block) error {
	return r.Storage.Remove(ctx, r.buildPath(b.Hash))
}

func (r BlockRepository) buildPath(id string) string {
	return fmt.Sprintf("blocks/%v", id)
}
