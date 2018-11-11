package network

import (
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/internal/app/rpcnode"
)

type TrustedNode struct {
	RpcNode    *rpcnode.RPCNode
	PeerNode   spvnode.Node
}
