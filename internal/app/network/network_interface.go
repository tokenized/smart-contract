package network

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

type NetworkInterface interface {
	Start() error
	RegisterTxListener(Listener)
	RegisterBlockListener(Listener)
	GetTX(context.Context, *chainhash.Hash) (*wire.MsgTx, error)
	SendTX(context.Context, *wire.MsgTx) (*chainhash.Hash, error)
	ListTransactions(context.Context, btcutil.Address) ([]btcjson.ListTransactionsResult, error)
}
