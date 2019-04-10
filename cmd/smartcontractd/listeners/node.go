package listeners

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Server struct {
	wallet           wallet.WalletInterface
	Config           *node.Config
	MasterDB         *db.DB
	RpcNode          *rpcnode.RPCNode
	SpyNode          *spynode.Node
	Scheduler        *scheduler.Scheduler
	TxCache          *TxCache
	Tracer           *Tracer
	Handler          protomux.Handler
	contractPKH      []byte // Used to determine which txs will be needed again
	pendingRequests  []*inspector.Transaction
	unsafeRequests   []*inspector.Transaction
	pendingResponses []*wire.MsgTx
	blockHeight      int // track current block height for confirm messages
	inSync           bool
}

func NewServer(wallet wallet.WalletInterface, handler protomux.Handler, config *node.Config, masterDB *db.DB,
	rpcNode *rpcnode.RPCNode, spyNode *spynode.Node, sch *scheduler.Scheduler, txCache *TxCache,
	tracer *Tracer, contractPKH []byte) *Server {
	result := Server{
		wallet:           wallet,
		Config:           config,
		MasterDB:         masterDB,
		RpcNode:          rpcNode,
		SpyNode:          spyNode,
		Scheduler:        sch,
		TxCache:          txCache,
		Tracer:           tracer,
		Handler:          handler,
		contractPKH:      contractPKH,
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

	if err := server.TxCache.Load(ctx, server.MasterDB); err != nil {
		return err
	}

	if err := server.Tracer.Load(ctx, server.MasterDB); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := server.SpyNode.Run(ctx); err != nil {
			logger.Error(ctx, "Spynode failed : %s", err)
			server.Scheduler.Stop(ctx)
		}
		logger.Debug(ctx, "Spynode finished")
	}()

	go func() {
		defer wg.Done()
		if err := server.Scheduler.Run(ctx); err != nil {
			logger.Error(ctx, "Scheduler failed : %s", err)
			server.SpyNode.Stop(ctx)
		}
		logger.Debug(ctx, "Scheduler finished")
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()

	if err := server.Tracer.Save(ctx, server.MasterDB); err != nil {
		return err
	}

	return server.TxCache.Save(ctx, server.MasterDB)
}

func (server *Server) Stop(ctx context.Context) error {
	// TODO: This action should unregister listeners
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
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		logger.Verbose(ctx, "Failed to marshal tx : %s", err)
	} else {
		logger.Verbose(ctx, "Broadcast Tx :\n%s", string(data))
	}
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
	logger.Info(ctx, "Saving pending response tx : %s", tx.TxHash().String())
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
					logger.Info(ctx, "Canceling pending response tx : %s", pendingTx.TxHash().String())
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
		logger.Warn(ctx, "Failed to remove conflicting pending : %s", err.Error())
		return err
	}

	// Save tx to cache so it can be used to process the response
	if bytes.Equal(itx.Outputs[0].Address.ScriptAddress(), server.contractPKH) {
		if err := server.RpcNode.SaveTX(ctx, itx.MsgTx); err != nil {
			return err
		}
	}

	return server.Handler.Trigger(ctx, protomux.SEE, itx)
}

func (server *Server) ReprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.Handler.Trigger(ctx, protomux.REPROCESS, itx)
}
