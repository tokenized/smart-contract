package txbuilder

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type NetInterface interface {
	GetTX(context.Context, *chainhash.Hash) (*wire.MsgTx, error)
}
