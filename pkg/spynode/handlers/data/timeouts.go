package data

import (
	"errors"
	"fmt"
	"time"
)

const (
	// Request timeouts in seconds
	handshakeTimeout = 30
	headerTimeout    = 60
	blockTimeout     = 600
)

func (state *State) CheckTimeouts() error {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()

	if !state.handshakeComplete && state.connectedTime != nil && now.Sub(*state.connectedTime).Seconds() > handshakeTimeout {
		return errors.New(fmt.Sprintf("Handshake took longer than %d seconds", handshakeTimeout))
	}

	if state.headersRequested != nil && now.Sub(*state.headersRequested).Seconds() > headerTimeout {
		return errors.New(fmt.Sprintf("Headers request took longer than %d seconds", headerTimeout))
	}

	for _, blockRequest := range state.blocksRequested {
		if now.Sub(blockRequest.time).Seconds() > blockTimeout {
			return errors.New(fmt.Sprintf("Block request took longer than %d seconds : %s",
				blockTimeout, blockRequest.hash.String()))
		}
	}

	return nil
}

func (state *UntrustedState) CheckTimeouts() error {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()

	if !state.handshakeComplete && state.connectedTime != nil && now.Sub(*state.connectedTime).Seconds() > handshakeTimeout {
		return errors.New(fmt.Sprintf("Handshake took longer than %d seconds", handshakeTimeout))
	}

	if state.headersRequested != nil && now.Sub(*state.headersRequested).Seconds() > headerTimeout {
		return errors.New(fmt.Sprintf("Headers request took longer than %d seconds", headerTimeout))
	}

	return nil
}
