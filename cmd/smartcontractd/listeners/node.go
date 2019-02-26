package listeners

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Server struct {
	RpcNode *rpcnode.RPCNode
	SpvNode *spvnode.Node
	Handler protomux.Handler
}

func (s *Server) Start() error {

	// Set responder
	s.Handler.SetResponder(s.respondTx)

	// Register listeners
	txListener := &TXListener{Handler: s.Handler, Node: s.RpcNode}
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

// respondTx is an internal method used as a the responder
// The method signatures are the same but we keep repeat for clarify
func (s *Server) respondTx(ctx context.Context, tx *wire.MsgTx) {
	s.RpcNode.SendTX(ctx, tx)
}
