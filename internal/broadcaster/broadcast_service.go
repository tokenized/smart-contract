package broadcaster

/**
 * Broadcaster Service
 *
 * What is my purpose?
 * - You broadcast responses
 */

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type BroadcastService struct {
	Network network.NetworkInterface
}

func NewBroadcastService(network network.NetworkInterface) BroadcastService {
	return BroadcastService{
		Network: network,
	}
}

func (s BroadcastService) Announce(ctx context.Context,
	tx *wire.MsgTx) (*chainhash.Hash, error) {
	hash, err := s.Network.SendTX(ctx, tx)
	if err != nil {
		return nil, err
	}

	return hash, nil
}
