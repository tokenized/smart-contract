package listeners

import (
	"bytes"
	"context"
	"sync"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

type Server struct {
	wallet           wallet.WalletInterface
	Config           *node.Config
	MasterDB         *db.DB
	RpcNode          *rpcnode.RPCNode
	SpyNode          *spynode.Node
	Scheduler        *scheduler.Scheduler
	Tracer           *Tracer
	utxos            *utxos.UTXOs
	Handler          protomux.Handler
	contractPKHs     [][]byte // Used to determine which txs will be needed again
	pendingRequests  []*inspector.Transaction
	unsafeRequests   []*inspector.Transaction
	pendingResponses []*wire.MsgTx
	revertedTxs      []*chainhash.Hash
	blockHeight      int // track current block height for confirm messages
	inSync           bool
}

func NewServer(wallet wallet.WalletInterface, handler protomux.Handler, config *node.Config, masterDB *db.DB,
	rpcNode *rpcnode.RPCNode, spyNode *spynode.Node, sch *scheduler.Scheduler,
	tracer *Tracer, contractPKHs [][]byte, utxos *utxos.UTXOs) *Server {
	result := Server{
		wallet:           wallet,
		Config:           config,
		MasterDB:         masterDB,
		RpcNode:          rpcNode,
		SpyNode:          spyNode,
		Scheduler:        sch,
		Tracer:           tracer,
		Handler:          handler,
		contractPKHs:     contractPKHs,
		utxos:            utxos,
		pendingRequests:  make([]*inspector.Transaction, 0),
		unsafeRequests:   make([]*inspector.Transaction, 0),
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

	if err := server.Tracer.Load(ctx, server.MasterDB); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := server.SpyNode.Run(ctx); err != nil {
			node.LogError(ctx, "Spynode failed : %s", err)
			server.Scheduler.Stop(ctx)
		}
		node.LogVerbose(ctx, "Spynode finished")
	}()

	go func() {
		defer wg.Done()
		if err := server.Scheduler.Run(ctx); err != nil {
			node.LogError(ctx, "Scheduler failed : %s", err)
			server.SpyNode.Stop(ctx)
		}
		node.LogVerbose(ctx, "Scheduler finished")
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()

	return server.Tracer.Save(ctx, server.MasterDB)
}

func (server *Server) Stop(ctx context.Context) error {
	spynodeErr := server.SpyNode.Stop(ctx)
	schedulerErr := server.Scheduler.Stop(ctx)

	if spynodeErr != nil && schedulerErr != nil {
		return errors.Wrap(errors.Wrap(spynodeErr, schedulerErr.Error()), "SpyNode and Scheduler failed")
	}
	if spynodeErr != nil {
		return errors.Wrap(spynodeErr, "Spynode failed to stop")
	}
	if schedulerErr != nil {
		return errors.Wrap(schedulerErr, "Scheduler failed to stop")
	}
	return nil
}

func (server *Server) sendTx(ctx context.Context, tx *wire.MsgTx) error {
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
	node.Log(ctx, "Saving pending response tx : %s", tx.TxHash().String())
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
					node.Log(ctx, "Canceling pending response tx : %s", pendingTx.TxHash().String())
					server.pendingResponses = append(server.pendingResponses[:i], server.pendingResponses[i+1:]...)
					return nil
				}
			}
		}
	}

	return nil
}

func (server *Server) processTx(ctx context.Context, itx *inspector.Transaction) error {
	if err := server.removeConflictingPending(ctx, itx); err != nil {
		node.LogWarn(ctx, "Failed to remove conflicting pending : %s", err.Error())
		return err
	}

	// Save tx to cache so it can be used to process the response
	for _, output := range itx.Outputs {
		for _, pkh := range server.contractPKHs {
			if bytes.Equal(output.Address.ScriptAddress(), pkh) {
				if err := server.RpcNode.SaveTX(ctx, itx.MsgTx); err != nil {
					return err
				}
			}
		}
	}

	server.Tracer.AddTx(ctx, itx.MsgTx)
	server.utxos.Add(itx.MsgTx, server.contractPKHs)
	return server.Handler.Trigger(ctx, "SEE", itx)
}

func (server *Server) cancelTx(ctx context.Context, itx *inspector.Transaction) error {
	server.Tracer.RevertTx(ctx, &itx.Hash)
	server.utxos.Remove(itx.MsgTx, server.contractPKHs)
	return server.Handler.Trigger(ctx, "STOLE", itx)
}

func (server *Server) revertTx(ctx context.Context, itx *inspector.Transaction) error {
	server.Tracer.RevertTx(ctx, &itx.Hash)
	server.utxos.Remove(itx.MsgTx, server.contractPKHs)
	return server.Handler.Trigger(ctx, "LOST", itx)
}

func (server *Server) ReprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.Handler.Trigger(ctx, "REPROCESS", itx)
}
