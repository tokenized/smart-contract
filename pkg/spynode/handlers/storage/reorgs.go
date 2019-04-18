package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	ReorgNotFound = errors.New("Reorg not found")
)

// ReorgRepository is used for managing reorg data.
type ReorgRepository struct {
	store storage.Storage
	mutex sync.Mutex
}

// NewReorgRepository returns a new ReorgRepository.
func NewReorgRepository(store storage.Storage) *ReorgRepository {
	result := ReorgRepository{
		store: store,
	}
	return &result
}

type Reorg struct {
	BlockHeight int
	Blocks      []ReorgBlock
}

// Note: These do not prove the txs were in the block without all txids or a merkle tree.
type ReorgBlock struct {
	Header wire.BlockHeader
	TxIds  []chainhash.Hash
}

// Save saves the active reorg.
func (repo *ReorgRepository) Save(ctx context.Context, reorg *Reorg) error {
	var buf bytes.Buffer
	if err := reorg.Write(&buf); err != nil {
		return errors.Wrap(err, "Failed to save reorg")
	}

	if err := repo.store.Write(ctx, repo.buildActivePath(), buf.Bytes(), nil); err != nil {
		return errors.Wrap(err, "Failed to save reorg")
	}

	return nil
}

// GetActive returns the active reorg or nil if there isn't one.
func (repo *ReorgRepository) GetActive(ctx context.Context) (*Reorg, error) {
	data, err := repo.store.Read(ctx, repo.buildActivePath())
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "Failed to read active reorg")
	}

	buf := bytes.NewBuffer(data)
	result := Reorg{}
	if err := result.Read(buf); err != nil {
		return nil, err
	}

	return &result, nil
}

// ClearActive clears the active reorg when it is completed and archives it.
func (repo *ReorgRepository) ClearActive(ctx context.Context) error {
	data, err := repo.store.Read(ctx, repo.buildActivePath())
	if err != nil {
		return errors.Wrap(err, "Failed to read active reorg")
	}

	buf := bytes.NewBuffer(data)
	active := Reorg{}
	if err := active.Read(buf); err != nil {
		return errors.Wrap(err, "Failed to parse active reorg")
	}

	if err := repo.store.Write(ctx, repo.buildPath(&active), data, nil); err != nil {
		return errors.Wrap(err, "Failed to archive active reorg")
	}

	if err := repo.store.Remove(ctx, repo.buildActivePath()); err != nil {
		return errors.Wrap(err, "Failed to remove active reorg")
	}

	return nil
}

// List all reorgs
func (repo *ReorgRepository) List(ctx context.Context) ([]*Reorg, error) {
	query := make(map[string]string)
	query["path"] = "reorgs"
	data, err := repo.store.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]*Reorg, len(data))
	for i, b := range data {
		buf := bytes.NewBuffer(b)
		if err := result[i].Read(buf); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (repo *ReorgRepository) buildPath(reorg *Reorg) string {
	return fmt.Sprintf("reorgs/%x", reorg.Id())
}

func (repo *ReorgRepository) buildActivePath() string {
	return "reorgs/active"
}

func (reorg *Reorg) Id() []byte {
	digest := sha256.New()

	height := uint32(reorg.BlockHeight)
	binary.Write(digest, binary.BigEndian, &height)

	for i, _ := range reorg.Blocks {
		hash := reorg.Blocks[i].Header.BlockHash()
		digest.Write(hash[:])
	}

	hash := digest.Sum(nil)
	return hash[:]
}

func (reorg *Reorg) Write(buf *bytes.Buffer) error {
	height := uint32(reorg.BlockHeight)
	if err := binary.Write(buf, binary.BigEndian, &height); err != nil {
		return err
	}

	count := uint32(len(reorg.Blocks))
	if err := binary.Write(buf, binary.BigEndian, &count); err != nil {
		return err
	}

	for i, _ := range reorg.Blocks {
		if err := reorg.Blocks[i].Write(buf); err != nil {
			return err
		}
	}

	return nil
}

func (reorg *Reorg) Read(buf *bytes.Buffer) error {
	var height uint32
	if err := binary.Read(buf, binary.BigEndian, &height); err != nil {
		return err
	}
	reorg.BlockHeight = int(height)

	var count uint32
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return err
	}

	reorg.Blocks = make([]ReorgBlock, count)
	for i, _ := range reorg.Blocks {
		if err := reorg.Blocks[i].Read(buf); err != nil {
			return err
		}
	}

	return nil
}

func (block *ReorgBlock) Write(buf *bytes.Buffer) error {
	if err := block.Header.Serialize(buf); err != nil {
		return err
	}

	count := uint32(len(block.TxIds))
	if err := binary.Write(buf, binary.BigEndian, &count); err != nil {
		return err
	}

	for i, _ := range block.TxIds {
		if _, err := buf.Write(block.TxIds[i][:]); err != nil {
			return err
		}
	}

	return nil
}

func (block *ReorgBlock) Read(buf *bytes.Buffer) error {
	err := block.Header.Deserialize(buf)
	if err != nil {
		return err
	}

	var count uint32
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return err
	}

	block.TxIds = make([]chainhash.Hash, count)
	for i, _ := range block.TxIds {
		if _, err := buf.Read(block.TxIds[i][:]); err != nil {
			return err
		}
	}

	return nil
}
