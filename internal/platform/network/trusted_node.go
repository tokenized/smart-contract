package network

import (
	"github.com/tokenized/smart-contract/internal/platform/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
)

type TrustedNode struct {
	RpcNode  *rpcnode.RPCNode
	PeerNode spvnode.Node
}
