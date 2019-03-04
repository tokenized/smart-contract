package listeners

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type rpcWithCache struct {
	RPCNode *rpcnode.RPCNode
	txCache map[chainhash.Hash]*wire.MsgTx
}

func newRPCWithCache(rpc *rpcnode.RPCNode) *rpcWithCache {
	result := rpcWithCache{
		RPCNode: rpc,
		txCache: make(map[chainhash.Hash]*wire.MsgTx),
	}

	return &result
}

func (node *rpcWithCache) GetTX(ctx context.Context, txid *chainhash.Hash) (*wire.MsgTx, error) {
	msg, ok := node.txCache[*txid]
	if ok {
		logger.Log(ctx, logger.Verbose, "Using tx from rpc cache : %s\n", txid.String())
		delete(node.txCache, *txid)
		return msg, nil
	}

	logger.Log(ctx, logger.Verbose, "Requesting tx from rpc : %s\n", txid.String())
	return node.RPCNode.GetTX(ctx, txid)
}

func (node *rpcWithCache) AddTX(ctx context.Context, msg *wire.MsgTx) error {
	logger.Log(ctx, logger.Verbose, "Saving tx to rpc cache : %s\n", msg.TxHash().String())
	node.txCache[msg.TxHash()] = msg
	return nil
}
