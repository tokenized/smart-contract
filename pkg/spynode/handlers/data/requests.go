package data

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	// Max concurrent block requests
	maxRequestedBlocks = 10
)

type requestedBlock struct {
	hash  chainhash.Hash
	time  time.Time // Time request was sent
	block *wire.MsgBlock
}

func (state *State) BlockIsRequested(hash *chainhash.Hash) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for _, item := range state.blocksRequested {
		if item.hash == *hash {
			return true
		}
	}
	return false
}

func (state *State) BlockIsToBeRequested(hash *chainhash.Hash) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for _, item := range state.blocksToRequest {
		if item == *hash {
			return true
		}
	}
	return false
}

// AddBlockRequest adds a block request to the queue.
// Returns true if the request should be made now.
// Returns false if the request is queued for later.
func (state *State) AddBlockRequest(hash *chainhash.Hash) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) >= maxRequestedBlocks {
		state.blocksToRequest = append(state.blocksToRequest, *hash)
		return false
	}

	newRequest := requestedBlock{
		hash:  *hash,
		time:  time.Now(),
		block: nil,
	}

	state.blocksRequested = append(state.blocksRequested, &newRequest)
	return true
}

// AddBlock adds the block message to the queued block request for later processing.
func (state *State) AddBlock(hash *chainhash.Hash, block *wire.MsgBlock) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for _, request := range state.blocksRequested {
		if request.hash == *hash {
			request.block = block
			return true
		}
	}

	return false
}

// AddNewBlock adds a new request with a block that was not requested.
func (state *State) AddNewBlock(hash *chainhash.Hash, block *wire.MsgBlock) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) > 0 || len(state.blocksRequested) > 0 {
		return false
	}

	newRequest := requestedBlock{
		hash:  *hash,
		time:  time.Now(),
		block: block,
	}

	state.blocksRequested = append(state.blocksRequested, &newRequest)
	return true
}

func (state *State) NextBlock() *wire.MsgBlock {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) == 0 || state.blocksRequested[0].block == nil {
		return nil
	}

	result := state.blocksRequested[0].block
	state.blocksRequested = state.blocksRequested[1:] // Remove first item
	return result
}

func (state *State) GetNextBlockToRequest() (*chainhash.Hash, int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksToRequest) == 0 || len(state.blocksRequested) >= maxRequestedBlocks {
		return nil, -1
	}

	newRequest := requestedBlock{
		hash:  state.blocksToRequest[0],
		time:  time.Now(),
		block: nil,
	}

	state.blocksToRequest = state.blocksToRequest[1:] // Remove first item
	state.blocksRequested = append(state.blocksRequested, &newRequest)
	return &newRequest.hash, len(state.blocksRequested)
}

func (state *State) RemoveBlockRequest(hash *chainhash.Hash) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) == 0 {
		return
	}

	if state.blocksRequested[0].hash == *hash {
		state.blocksRequested = state.blocksRequested[1:] // Remove first item
	}
}

func (state *State) BlocksRequestedCount() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksRequested)
}

func (state *State) BlocksToRequestCount() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksToRequest)
}

func (state *State) TotalBlockRequestCount() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksRequested) + len(state.blocksToRequest)
}

func (state *State) BlockRequestsEmpty() bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksToRequest) == 0 && len(state.blocksRequested) == 0
}

func (state *State) LastHash() *chainhash.Hash {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksToRequest) > 0 {
		return &state.blocksToRequest[len(state.blocksToRequest)-1]
	}

	if len(state.blocksRequested) > 0 {
		return &state.blocksRequested[len(state.blocksRequested)-1].hash
	}

	return nil
}
