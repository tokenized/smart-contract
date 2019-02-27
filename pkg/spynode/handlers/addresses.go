package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// AddressHandler exists to handle the Addresses command.
type AddressHandler struct {
	peers *storage.PeerRepository
}

// NewAddressHandler returns a new VersionHandler with the given Config.
func NewAddressHandler(peers *storage.PeerRepository) *AddressHandler {
	result := AddressHandler{peers: peers}
	return &result
}

// Processes addresses message
func (handler *AddressHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgAddr)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgAddr")
	}

	modified := false
	for _, address := range msg.AddrList {
		added, _ := handler.peers.Add(ctx, fmt.Sprintf("[%s]:%d", address.IP.To16().String(), address.Port))
		if added {
			modified = true
		}
	}

	if modified {
		handler.peers.Save(ctx)
	}

	return nil, nil
}
