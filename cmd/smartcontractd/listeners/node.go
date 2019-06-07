package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
)

type Server struct {
	wallet           wallet.WalletInterface
	Config           *node.Config
	MasterDB         *db.DB
	RpcNode          inspector.NodeInterface
	SpyNode          *spynode.Node
	Headers          node.BitcoinHeaders
	Scheduler        *scheduler.Scheduler
	Tracer           *Tracer
	utxos            *utxos.UTXOs
	lock             sync.Mutex
	Handler          protomux.Handler
	contractPKHs     [][]byte // Used to determine which txs will be needed again
	pendingResponses []*wire.MsgTx
	revertedTxs      []*chainhash.Hash
	blockHeight      int // track current block height for confirm messages
	inSync           bool

	pendingTxs  map[chainhash.Hash]*IncomingTxData
	readyTxs    []*chainhash.Hash // Saves order of tx approval in case preprocessing doesn't finish before approval.
	pendingLock sync.Mutex

	incomingTxs   IncomingTxChannel
	processingTxs ProcessingTxChannel

	TxSentCount        int
	AlternateResponder protomux.ResponderFunc
}

func NewServer(wallet wallet.WalletInterface, handler protomux.Handler, config *node.Config, masterDB *db.DB,
	rpcNode inspector.NodeInterface, spyNode *spynode.Node, headers node.BitcoinHeaders, sch *scheduler.Scheduler,
	tracer *Tracer, utxos *utxos.UTXOs) *Server {
	result := Server{
		wallet:           wallet,
		Config:           config,
		MasterDB:         masterDB,
		RpcNode:          rpcNode,
		SpyNode:          spyNode,
		Headers:          headers,
		Scheduler:        sch,
		Tracer:           tracer,
		Handler:          handler,
		utxos:            utxos,
		pendingTxs:       make(map[chainhash.Hash]*IncomingTxData),
		pendingResponses: make([]*wire.MsgTx, 0),
		blockHeight:      0,
		inSync:           false,
	}

	result.incomingTxs.Open(100)
	result.processingTxs.Open(100)

	keys := wallet.ListAll()
	result.contractPKHs = make([][]byte, 0, len(keys))
	for _, key := range keys {
		result.contractPKHs = append(result.contractPKHs, key.Address.ScriptAddress())
	}

	return &result
}

func (server *Server) Run(ctx context.Context) error {
	// Set responder
	server.Handler.SetResponder(server.respondTx)
	server.Handler.SetReprocessor(server.reprocessTx)

	// Register listeners
	if server.SpyNode != nil {
		server.SpyNode.RegisterListener(server)
	}

	if err := server.Tracer.Load(ctx, server.MasterDB); err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	if server.SpyNode != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.SpyNode.Run(ctx); err != nil {
				node.LogError(ctx, "Spynode failed : %s", err)
			}
			node.LogVerbose(ctx, "Spynode thread stopping Scheduler")
			server.Scheduler.Stop(ctx)
			server.incomingTxs.Close()
			server.processingTxs.Close()
			node.LogVerbose(ctx, "Spynode finished")
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Scheduler.Run(ctx); err != nil {
			node.LogError(ctx, "Scheduler failed : %s", err)
		}
		if server.SpyNode != nil {
			node.LogVerbose(ctx, "Scheduler thread stopping Spynode")
			server.SpyNode.Stop(ctx)
		}
		server.incomingTxs.Close()
		server.processingTxs.Close()
		node.LogVerbose(ctx, "Scheduler finished")
	}()

	rks := server.wallet.ListAll()
	contractPKHs := make([]*protocol.PublicKeyHash, 0, len(rks))
	for _, cpkh := range server.contractPKHs {
		pkh := protocol.PublicKeyHashFromBytes(cpkh)
		contractPKHs = append(contractPKHs, pkh)
	}
	for i := 0; i < server.Config.PreprocessThreads; i++ {
		node.Log(ctx, "Starting pre-process thread %d", i)
		// Start preprocess thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.ProcessIncomingTxs(ctx, server.MasterDB, contractPKHs, server.Headers); err != nil {
				node.LogError(ctx, "Pre-process failed : %s", err)
			}
			node.LogVerbose(ctx, "Pre-process thread finished")
		}()
	}

	// Start process thread
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ProcessTxs(ctx); err != nil {
			node.LogError(ctx, "Process failed : %s", err)
		}
		node.LogVerbose(ctx, "Process thread finished")
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()

	if err := holdings.WriteCache(ctx, server.MasterDB); err != nil {
		return err
	}

	return server.Tracer.Save(ctx, server.MasterDB)
}

func (server *Server) Stop(ctx context.Context) error {
	var spynodeErr error
	if server.SpyNode != nil {
		spynodeErr = server.SpyNode.Stop(ctx)
	}
	schedulerErr := server.Scheduler.Stop(ctx)
	server.incomingTxs.Close()
	server.processingTxs.Close()

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

func (server *Server) SetInSync() {
	server.inSync = true
}

func (server *Server) SetAlternateResponder(responder protomux.ResponderFunc) {
	server.AlternateResponder = responder
}

func (server *Server) sendTx(ctx context.Context, tx *wire.MsgTx) error {
	server.TxSentCount++
	if server.AlternateResponder != nil {
		server.AlternateResponder(ctx, tx)
	}
	if server.SpyNode != nil {
		if err := server.SpyNode.BroadcastTx(ctx, tx); err != nil {
			return err
		}
		if err := server.SpyNode.HandleTx(ctx, tx); err != nil {
			return err
		}
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

func (server *Server) reprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.processingTxs.Add(ProcessingTx{Itx: itx, Event: "REPROCESS"})
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

func (server *Server) cancelTx(ctx context.Context, itx *inspector.Transaction) error {
	server.lock.Lock()
	defer server.lock.Unlock()

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
