package network

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

/**
 * Network Kit
 *
 * What is my purpose?
 * - You manage a trusted node
 * - You manage peer nodes
 * - You tell me about double spends
 * - You tell me about transaction propagation
 */
type Network struct {
	TrustedNode TrustedNode
	// PeerNodes     []spvnode.Node
}

func NewNetwork(rc rpcnode.Config, pn spvnode.Node) (*Network, error) {
	rn, err := rpcnode.NewNode(rc)
	if err != nil {
		return nil, err
	}

	tn := TrustedNode{
		RpcNode:  rn,
		PeerNode: pn,
	}

	n := &Network{
		TrustedNode: tn,
	}

	return n, nil
}

func (n Network) RegisterTxListener(listener Listener) {
	n.TrustedNode.PeerNode.RegisterListener(spvnode.ListenerTX, listener)
}

func (n Network) RegisterBlockListener(listener Listener) {
	n.TrustedNode.PeerNode.RegisterListener(spvnode.ListenerBlock, listener)
}

func (n Network) Start() error {
	return n.TrustedNode.PeerNode.Start()
}

//
// RPC Node proxies
//

func (n Network) GetTX(ctx context.Context, id *chainhash.Hash) (*wire.MsgTx, error) {
	return n.TrustedNode.RpcNode.GetTX(ctx, id)
}

func (n Network) SendTX(ctx context.Context, tx *wire.MsgTx) (*chainhash.Hash, error) {
	return n.TrustedNode.RpcNode.SendTX(ctx, tx)
}

func (n Network) ListTransactions(ctx context.Context, address btcutil.Address) ([]btcjson.ListTransactionsResult, error) {
	return n.TrustedNode.RpcNode.ListTransactions(ctx, address)
}

func (n Network) WatchAddress(ctx context.Context, address btcutil.Address) error {
	return n.TrustedNode.RpcNode.WatchAddress(ctx, address)
}
