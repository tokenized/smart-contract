package data

import (
	"errors"
	"fmt"
	"time"
)

const (
	// Request timeouts in seconds
	handshakeTimeout = 60
	headerTimeout    = 120
	blockTimeout     = 600

	// Maximum number of restarts allowed in a minute before stopping
	maxRestarts = 5 // TODO This needs to stop the node
)

func (state *State) LogRestart() {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.restartCount++
}

func (state *State) CheckTimeouts() error {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	now := time.Now()

	connected := state.ConnectedTime // Copy because updates are outside of mutex
	if !state.HandshakeComplete && connected != nil && now.Sub(*connected).Seconds() > handshakeTimeout {
		return errors.New(fmt.Sprintf("Handshake took longer than %d seconds", handshakeTimeout))
	}

	headersRequested := state.HeadersRequested // Copy because updates are outside of mutex
	if headersRequested != nil && now.Sub(*headersRequested).Seconds() > headerTimeout {
		return errors.New(fmt.Sprintf("Headers request took longer than %d seconds", headerTimeout))
	}

	for _, blockRequest := range state.blocksRequested {
		if now.Sub(blockRequest.Time).Seconds() > blockTimeout {
			return errors.New(fmt.Sprintf("Block request took longer than %d seconds : %s",
				blockTimeout, blockRequest.Hash.String()))
		}
	}

	if state.restartCount > 0 && now.Sub(*connected).Seconds() > 60 {
		// Clear restart count
		state.restartCount = 0
	}

	if state.restartCount > maxRestarts {
		return errors.New(fmt.Sprintf("Restarted %d seconds", headerTimeout))
	}

	return nil
}

func (state *UntrustedState) CheckTimeouts() error {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	now := time.Now()

	connected := state.ConnectedTime // Copy because updates are outside of mutex
	if !state.HandshakeComplete && connected != nil && now.Sub(*connected).Seconds() > handshakeTimeout {
		return errors.New(fmt.Sprintf("Handshake took longer than %d seconds", handshakeTimeout))
	}

	headersRequested := state.HeadersRequested // Copy because updates are outside of mutex
	if headersRequested != nil && now.Sub(*headersRequested).Seconds() > headerTimeout {
		return errors.New(fmt.Sprintf("Headers request took longer than %d seconds", headerTimeout))
	}

	return nil
}
