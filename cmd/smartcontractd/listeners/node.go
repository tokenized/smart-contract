package listeners

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Server struct {
	RpcNode          *rpcWithCache
	SpyNode          *spynode.Node
	Handler          protomux.Handler
	contractPKH      []byte // Used to determine which txs will be needed again
	pendingResponses []*wire.MsgTx
	blockHeight      int // track current block height for confirm messages
	inSync           bool
}

func NewServer(rpcNode *rpcnode.RPCNode, spyNode *spynode.Node, handler protomux.Handler, contractPKH []byte) *Server {
	result := Server{
		RpcNode:          newRPCWithCache(rpcNode),
		SpyNode:          spyNode,
		Handler:          handler,
		contractPKH:      contractPKH,
		pendingResponses: make([]*wire.MsgTx, 0),
		blockHeight:      0,
		inSync:           false,
	}

	return &result
}

func (server *Server) Run(ctx context.Context) error {

	// Set responder
	server.Handler.SetResponder(server.respondTx)

	// Register listeners
	server.SpyNode.RegisterListener(server)

	if err := server.SpyNode.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (server *Server) Stop(ctx context.Context) error {
	// TODO: This action should unregister listeners

	if err := server.SpyNode.Stop(ctx); err != nil {
		return err
	}

	return nil
}

func (server *Server) sendTx(ctx context.Context, tx *wire.MsgTx) error {
	server.RpcNode.RPCNode.SendTX(ctx, tx)
	if err := server.SpyNode.BroadcastTx(ctx, tx); err != nil {
		return err
	}
	if err := server.SpyNode.HandleTx(ctx, tx); err != nil {
		return err
	}
	return nil
}

// respondTx is an internal method used as a the responder
// The method signatures are the same but we keep repeat for clarify
func (server *Server) respondTx(ctx context.Context, tx *wire.MsgTx) error {
	if server.inSync {
		return server.sendTx(ctx, tx)
	}

	// Append to pending so it can be monitored
	logger.Log(ctx, logger.Info, "Saving pending response tx : %s", tx.TxHash().String())
	server.pendingResponses = append(server.pendingResponses, tx)
	return nil
}

// Remove any pending that are conflicting with this tx.
// Contract responses use the tx output from the request to the contract as a tx input in the response tx.
// So if that contract request output is spent by another tx, then the contract has already responded.
func (server *Server) removeConflictingPending(ctx context.Context, itx *inspector.Transaction) error {
	for i, pendingTx := range server.pendingResponses {
		for _, pendingInput := range pendingTx.TxIn {
			for _, input := range itx.Inputs {
				if pendingInput.PreviousOutPoint.Hash == input.UTXO.Hash &&
					pendingInput.PreviousOutPoint.Index == input.UTXO.Index {
					logger.Log(ctx, logger.Info, "Pending response tx removed : %s", pendingTx.TxHash().String())
					server.pendingResponses = append(server.pendingResponses[:i], server.pendingResponses[i+1:]...)
					return nil
				}
			}
		}
	}

	return nil
}
