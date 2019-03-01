package data

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	// Max concurrent block requests
	maxRequestedBlocks = 10
)

type RequestedBlock struct {
	Hash chainhash.Hash
	Time time.Time // Time request was sent
}

func (state *State) BlockIsRequested(hash *chainhash.Hash) bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	for _, item := range state.blocksRequested {
		if item.Hash == *hash {
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

// Returns -1 if block can't be added to current requests. (too requests many already)
// Returns offset from current block height of block if it was added to current requests.
func (state *State) AddBlockRequest(hash *chainhash.Hash) int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) >= maxRequestedBlocks {
		state.blocksToRequest = append(state.blocksToRequest, *hash)
		return -1
	}

	newRequest := RequestedBlock{
		Hash: *hash,
		Time: time.Now(),
	}

	state.blocksRequested = append(state.blocksRequested, newRequest)
	return len(state.blocksRequested)
}

func (state *State) GetNextBlockToRequest(hash *chainhash.Hash) (*chainhash.Hash, int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksToRequest) == 0 || len(state.blocksRequested) >= maxRequestedBlocks {
		return nil, -1
	}

	newRequest := RequestedBlock{
		Hash: state.blocksToRequest[0],
		Time: time.Now(),
	}

	state.blocksToRequest = state.blocksToRequest[1:] // Remove first item
	state.blocksRequested = append(state.blocksRequested, newRequest)
	return &newRequest.Hash, len(state.blocksRequested)
}

func (state *State) RemoveBlockRequest(hash *chainhash.Hash) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) == 0 {
		return
	}

	if state.blocksRequested[0].Hash == *hash {
		state.blocksRequested = state.blocksRequested[1:] // Remove first item
	}
}

func (state *State) BlocksRequestedCount() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksRequested)
}

func (state *State) BlockRequestsEmpty() bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return len(state.blocksToRequest) == 0 && len(state.blocksRequested) == 0
}

func (state *State) LastHash() *chainhash.Hash {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if len(state.blocksRequested) > 0 {
		return &state.blocksRequested[len(state.blocksRequested)-1].Hash
	}

	if len(state.blocksToRequest) > 0 {
		return &state.blocksToRequest[len(state.blocksToRequest)-1]
	}

	return nil
}
