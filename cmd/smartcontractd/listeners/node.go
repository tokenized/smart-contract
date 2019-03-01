package listeners

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Server struct {
	RpcNode *rpcnode.RPCNode
	SpyNode *spynode.Node
	Handler protomux.Handler
}

func (s *Server) Run(ctx context.Context) error {

	// Set responder
	s.Handler.SetResponder(s.respondTx)

	// Register listeners
	listener := Listener{Handler: s.Handler, Node: s.RpcNode}
	s.SpyNode.RegisterListener(&listener)

	if err := s.SpyNode.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	// TODO: This action should unregister listeners

	if err := s.SpyNode.Stop(ctx); err != nil {
		return err
	}

	return nil
}

// respondTx is an internal method used as a the responder
// The method signatures are the same but we keep repeat for clarify
func (s *Server) respondTx(ctx context.Context, tx *wire.MsgTx) error {
	s.RpcNode.SendTX(ctx, tx)
	if err := s.SpyNode.BroadcastTx(ctx, tx); err != nil {
		return err
	}
	if err := s.SpyNode.HandleTx(ctx, tx); err != nil {
		return err
	}
	return nil
}
