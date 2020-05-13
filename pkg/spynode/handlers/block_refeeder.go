package handlers

import (
	"sync"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type BlockRefeeder struct {
	nextHeight int
	nextHash   bitcoin.Hash32
	requested  bool
	block      *wire.MsgParseBlock
	lock       sync.Mutex
}

func (br *BlockRefeeder) SetBlock(hash bitcoin.Hash32, block *wire.MsgParseBlock) bool {
	br.lock.Lock()
	defer br.lock.Unlock()

	if hash.Equal(&br.nextHash) {
		br.block = block
		return true
	}

	return false
}

func (br *BlockRefeeder) GetBlock() (*wire.MsgParseBlock, int, bool) {
	br.lock.Lock()
	defer br.lock.Unlock()

	return br.block, br.nextHeight, br.nextHeight != 0
}

// GetBlockToRequest returns a block to request if there is one.
// Returns a block hash if one should be requested and false if the block feeder is clear.
// If no block hash is returned, but true is returned, then no other blocks should be requested.
func (br *BlockRefeeder) GetBlockToRequest() *bitcoin.Hash32 {
	br.lock.Lock()
	defer br.lock.Unlock()

	if br.nextHeight == 0 {
		return nil // block refeeder is inactive
	}

	if br.requested {
		return nil // block is already requested. wait for it
	}

	br.requested = true
	return &br.nextHash // request this block
}

func (br *BlockRefeeder) IsNextBlock(hash bitcoin.Hash32) bool {
	br.lock.Lock()
	defer br.lock.Unlock()

	return hash.Equal(&br.nextHash)
}

func (br *BlockRefeeder) NextHeight() int {
	br.lock.Lock()
	defer br.lock.Unlock()

	return br.nextHeight
}

func (br *BlockRefeeder) SetHeight(height int, hash bitcoin.Hash32) {
	br.lock.Lock()
	defer br.lock.Unlock()

	if br.nextHeight == 0 || br.nextHeight > height {
		br.nextHeight = height
		br.nextHash = hash
		br.requested = false
	}
}

// Increment specifies the next block to be requested. It should be called when the current block
//   has been received.
func (br *BlockRefeeder) Increment(height int, hash bitcoin.Hash32) {
	br.lock.Lock()
	defer br.lock.Unlock()

	if br.nextHeight+1 == height {
		br.nextHeight = height
		br.nextHash = hash
		br.requested = false
	}
}

// Clear turns off the block refeeder. It should be called when the refeeder has caught the main
//   block feeder.
func (br *BlockRefeeder) Clear(height int) {
	br.lock.Lock()
	defer br.lock.Unlock()

	if br.nextHeight == height {
		br.nextHeight = 0
		br.requested = true
	}
}
