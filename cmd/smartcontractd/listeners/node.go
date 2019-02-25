package listeners

import (
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
)

type Server struct {
	RpcNode *rpcnode.RPCNode
	SpvNode *spvnode.Node
	Handler protomux.Handler
}

func (s *Server) Start() error {

	// Register listeners
	txListener := &TXListener{Handler: s.Handler}
	s.SpvNode.RegisterListener(spvnode.ListenerTX, txListener)

	if err := s.SpvNode.Start(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Close() error {
	// TODO: This action should unregister listeners

	if err := s.SpvNode.Close(); err != nil {
		return err
	}

	return nil
}
