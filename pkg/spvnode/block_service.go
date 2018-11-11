package spvnode

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/spvnode/logger"
)

const (
	// maxBlocks is the number of recent blocks to keep in the data store.
	//
	// At approximately 1 block every 10 minutes, 10000 blocks is roughly
	// a week worth of blocks.
	maxBlocks = 10000
)

type BlockService struct {
	BlockRepostory  BlockRepository
	StateRepository StateRepository
	Blocks          map[chainhash.Hash]Block
	State           *State
	synced          bool
}

func NewBlockService(br BlockRepository, sr StateRepository) BlockService {
	return BlockService{
		BlockRepostory:  br,
		StateRepository: sr,
		Blocks:          map[chainhash.Hash]Block{},
	}
}

func (b *BlockService) LoadBlocks(ctx context.Context) error {
	blocks, err := b.BlockRepostory.All(ctx)
	if err != nil {
		return err
	}

	for _, block := range blocks {
		h, err := chainhash.NewHashFromStr(block.Hash)
		if err != nil {
			return err
		}

		b.Blocks[*h] = block
	}

	return nil
}

// HasBlock returns true if the block is currently in the cache, false
// otherwise.
func (b BlockService) HasBlock(ctx context.Context, hash chainhash.Hash) bool {
	_, ok := b.Blocks[hash]

	return ok
}

func (b BlockService) Read(ctx context.Context,
	hash chainhash.Hash) (*Block, error) {

	block, ok := b.Blocks[hash]
	if ok {
		return &block, nil
	}

	bl, err := b.BlockRepostory.Read(ctx, hash.String())
	if err != nil {
		return nil, err
	}

	return bl, nil
}

func (b *BlockService) Write(ctx context.Context, block Block) error {
	if err := b.BlockRepostory.Write(ctx, block); err != nil {
		return err
	}

	h, _ := chainhash.NewHashFromStr(block.Hash)
	b.Blocks[*h] = block

	return nil
}

func (b BlockService) LastSeen(ctx context.Context,
	block Block) (*Block, error) {

	if b.State != nil && block.Height <= b.State.LastSeen.Height {
		// the state we have is already higher than this state, ignore it
		return &b.State.LastSeen, nil
	}

	// update last seen
	state := State{
		LastSeen: block,
	}

	b.State = &state

	log := logger.NewLoggerFromContext(ctx).Sugar()

	blk := state.LastSeen
	log.Infof("New state last_seen hash=%v height=%v", blk.Hash, blk.Height)

	if err := b.StateRepository.Write(ctx, state); err != nil {
		return nil, err
	}

	return &block, nil
}

func (b BlockService) Remove(ctx context.Context, block Block) error {
	if err := b.BlockRepostory.Remove(ctx, block); err != nil {
		return err
	}

	h, _ := chainhash.NewHashFromStr(block.Hash)
	delete(b.Blocks, *h)

	return nil
}

func (b BlockService) prune(ctx context.Context, max int32) error {
	if len(b.Blocks) < maxBlocks {
		return nil
	}

	// delete any blocks with a height less than this
	minHeight := max - maxBlocks

	for k, block := range b.Blocks {
		if block.Height < minHeight {

			// delete from the store.
			if err := b.BlockRepostory.Remove(ctx, block); err != nil {
				return err
			}

			// maybe link the store and the hash.
			delete(b.Blocks, k)
		}
	}

	return nil
}

func (b *BlockService) LoadState(ctx context.Context) (*State, error) {
	state, err := b.StateRepository.Read(ctx, "")

	if err != nil {
		if err != ErrStateNotFound {
			return nil, err
		}
	}

	if err == ErrStateNotFound {
		// state wasn't found, make sure we can write it before starting
		state = &State{}

		if err = b.StateRepository.Write(ctx, *state); err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	b.State = state

	return state, nil
}
