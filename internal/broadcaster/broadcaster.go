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

func Announce(ctx context.Context, network network.NetworkInterface, tx *wire.MsgTx) (*chainhash.Hash, error) {
	hash, err := network.SendTX(ctx, tx)
	if err != nil {
		return nil, err
	}

	return hash, nil
}
