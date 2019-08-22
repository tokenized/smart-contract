package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	UntrustedHeaderDelta = 6
)

// HeadersHandler exists to handle the headers command.
type UntrustedHeadersHandler struct {
	state   *data.UntrustedState
	peers   *storage.PeerRepository
	address string
	blocks  *storage.BlockRepository
}

// NewUntrustedHeadersHandler returns a new UntrustedHeadersHandler with the given Config.
func NewUntrustedHeadersHandler(state *data.UntrustedState, peers *storage.PeerRepository, address string, blockRepo *storage.BlockRepository) *UntrustedHeadersHandler {
	result := UntrustedHeadersHandler{
		state:   state,
		peers:   peers,
		address: address,
		blocks:  blockRepo,
	}
	return &result
}

// Implements the Handler interface.
// Headers are in order from lowest block height, to highest.
// Untrusted nodes request headers starting several blocks back to help verify chain.
func (handler *UntrustedHeadersHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	message, ok := m.(*wire.MsgHeaders)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	if handler.state.IsReady() {
		return nil, nil
	}

	if len(message.Headers) == 0 {
		// Untrusted nodes should never get zero headers unless something is wrong.
		return nil, errors.New("Returned zero headers")
	}

	// Verify the first header is within 1 - UntrustedHeaderDelta of the top
	hash := message.Headers[0].BlockHash()
	height, exists := handler.blocks.Height(hash)
	if !exists {
		// This can happen if this node is ahead of our trusted node, but still can't be trusted.
		return nil, errors.New("Returned unknown header")
	}

	if height < handler.blocks.LastHeight()-UntrustedHeaderDelta-1 {
		handler.peers.UpdateScore(ctx, handler.address, -1)
		return nil, errors.New(fmt.Sprintf("Returned header at low height : %d", height))
	}

	// Verify headers are linked
	// Note: POW check might be nice here
	previousHash := hash
	for _, header := range message.Headers[1:] {
		if !header.PrevBlock.Equal(previousHash) {
			return nil, errors.New("Returned unlinked headers")
		}

		previousHash = header.BlockHash()
	}

	handler.state.ClearHeadersRequested()
	handler.state.SetVerified()
	return nil, nil
}
