package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	walletKey = "wallet" // storage path for wallet
)

type Server struct {
	wallet            wallet.WalletInterface
	Config            *node.Config
	MasterDB          *db.DB
	RpcNode           inspector.NodeInterface
	SpyNode           *spynode.Node
	Headers           node.BitcoinHeaders
	Scheduler         *scheduler.Scheduler
	Tracer            *filters.Tracer
	utxos             *utxos.UTXOs
	lock              sync.Mutex
	Handler           protomux.Handler
	contractAddresses []bitcoin.RawAddress // Used to determine which txs will be needed again
	walletLock        sync.RWMutex
	txFilter          *filters.TxFilter
	pendingRequests   []pendingRequest
	pendingResponses  inspector.TransactionList
	revertedTxs       []*bitcoin.Hash32
	blockHeight       int // track current block height for confirm messages
	inSync            bool

	pendingTxs  map[bitcoin.Hash32]*IncomingTxData
	readyTxs    []*bitcoin.Hash32 // Saves order of tx approval in case preprocessing doesn't finish before approval.
	pendingLock sync.Mutex

	incomingTxs   IncomingTxChannel
	processingTxs ProcessingTxChannel

	holdingsChannel *holdings.CacheChannel

	TxSentCount        int
	AlternateResponder protomux.ResponderFunc
}

type pendingRequest struct {
	Itx           *inspector.Transaction
	ContractIndex int // Index of output that goes to contract address
}

func NewServer(
	wallet wallet.WalletInterface,
	handler protomux.Handler,
	config *node.Config,
	masterDB *db.DB,
	rpcNode inspector.NodeInterface,
	spyNode *spynode.Node,
	headers node.BitcoinHeaders,
	sch *scheduler.Scheduler,
	tracer *filters.Tracer,
	utxos *utxos.UTXOs,
	txFilter *filters.TxFilter,
	holdingsChannel *holdings.CacheChannel,
) *Server {
	result := &Server{
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
		txFilter:         txFilter,
		pendingTxs:       make(map[bitcoin.Hash32]*IncomingTxData),
		pendingRequests:  make([]pendingRequest, 0),
		pendingResponses: make(inspector.TransactionList, 0),
		blockHeight:      0,
		inSync:           false,
		holdingsChannel:  holdingsChannel,
	}

	return result
}

func (server *Server) Load(ctx context.Context) error {
	// Set responder
	server.Handler.SetResponder(server.respondTx)
	server.Handler.SetReprocessor(server.reprocessTx)

	server.incomingTxs.Open(100)
	server.processingTxs.Open(100)
	server.holdingsChannel.Open(5000)

	// Register listeners
	if server.SpyNode != nil {
		server.SpyNode.RegisterListener(server)
	}

	if err := server.Tracer.Load(ctx, server.MasterDB); err != nil {
		return errors.Wrap(err, "load trader")
	}

	return nil
}

func (server *Server) Run(ctx context.Context) error {

	wg := sync.WaitGroup{}

	if server.SpyNode != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.SpyNode.Run(ctx); err != nil {
				node.LogError(ctx, "Spynode failed : %s", err)
				node.LogVerbose(ctx, "Spynode thread stopping Scheduler")
				server.Scheduler.Stop(ctx)
				server.incomingTxs.Close()
				server.processingTxs.Close()
				server.holdingsChannel.Close()
			}
			node.LogVerbose(ctx, "Spynode finished")
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Scheduler.Run(ctx); err != nil {
			node.LogError(ctx, "Scheduler failed : %s", err)
			if server.SpyNode != nil {
				node.LogVerbose(ctx, "Scheduler thread stopping Spynode")
				server.SpyNode.Stop(ctx)
			}
			server.incomingTxs.Close()
			server.processingTxs.Close()
			server.holdingsChannel.Close()
		}
		node.LogVerbose(ctx, "Scheduler finished")
	}()

	for i := 0; i < server.Config.PreprocessThreads; i++ {
		node.Log(ctx, "Starting pre-process thread %d", i)
		// Start preprocess thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.ProcessIncomingTxs(ctx, server.MasterDB, server.Headers); err != nil {
				node.LogError(ctx, "Pre-process failed : %s", err)
				server.Scheduler.Stop(ctx)
				if server.SpyNode != nil {
					node.LogVerbose(ctx, "Process incoming thread stopping Spynode")
					server.SpyNode.Stop(ctx)
				}
				server.incomingTxs.Close()
				server.processingTxs.Close()
				server.holdingsChannel.Close()
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
			server.Scheduler.Stop(ctx)
			if server.SpyNode != nil {
				node.LogVerbose(ctx, "Process thread stopping Spynode")
				server.SpyNode.Stop(ctx)
			}
			server.incomingTxs.Close()
			server.processingTxs.Close()
			server.holdingsChannel.Close()
		}
		node.LogVerbose(ctx, "Process thread finished")
	}()

	// Start holdings cache writer thread
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := holdings.ProcessCacheItems(ctx, server.MasterDB, server.holdingsChannel); err != nil {
			node.LogError(ctx, "Process holdings cache failed : %s", err)
			server.Scheduler.Stop(ctx)
			if server.SpyNode != nil {
				node.LogVerbose(ctx, "Process cache thread stopping Spynode")
				server.SpyNode.Stop(ctx)
			}
			server.incomingTxs.Close()
			server.processingTxs.Close()
		}
		node.LogVerbose(ctx, "Process holdings cache thread finished")
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()

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
	server.holdingsChannel.Close()

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
	server.lock.Lock()
	defer server.lock.Unlock()

	server.inSync = true
}

func (server *Server) IsInSync() bool {
	server.lock.Lock()
	defer server.lock.Unlock()

	return server.inSync
}

func (server *Server) SetAlternateResponder(responder protomux.ResponderFunc) {
	server.lock.Lock()
	defer server.lock.Unlock()

	server.AlternateResponder = responder
}

func (server *Server) sendTx(ctx context.Context, tx *wire.MsgTx) error {
	server.TxSentCount++

	if server.SpyNode != nil {
		if err := server.SpyNode.BroadcastTx(ctx, tx); err != nil {
			return err
		}
	}

	if server.AlternateResponder != nil {
		server.AlternateResponder(ctx, tx)
	}

	return nil
}

// respondTx is an internal method used as the responder
func (server *Server) respondTx(ctx context.Context, tx *wire.MsgTx) error {
	server.lock.Lock()
	defer server.lock.Unlock()

	// Add to spynode and mark as safe so it will be processed now
	if server.SpyNode != nil {
		if err := server.SpyNode.HandleTx(ctx, tx); err != nil {
			return err
		}
	}

	// Broadcast to network
	if err := server.sendTx(ctx, tx); err != nil {
		return err
	}

	if server.AlternateResponder != nil {
		server.AlternateResponder(ctx, tx)
	}

	return nil
}

func (server *Server) reprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.processingTxs.Add(ProcessingTx{Itx: itx, Event: "END"})
}

// Remove any pending that are conflicting with this tx.
// Contract responses use the tx output from the request to the contract as a tx input in the response tx.
// So if that contract request output is spent by another tx, then the contract has already responded.
func (server *Server) removeConflictingPending(ctx context.Context, itx *inspector.Transaction) error {
	for i, pendingTx := range server.pendingRequests {
		if pendingTx.ContractIndex < len(itx.MsgTx.TxIn) &&
			itx.MsgTx.TxIn[pendingTx.ContractIndex].PreviousOutPoint.Hash.Equal(pendingTx.Itx.Hash) {
			node.Log(ctx, "Canceling pending request tx : %s", pendingTx.Itx.Hash.String())
			server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)
			return nil
		}
	}

	return nil
}

func (server *Server) cancelTx(ctx context.Context, itx *inspector.Transaction) error {
	server.lock.Lock()
	defer server.lock.Unlock()

	server.Tracer.RevertTx(ctx, itx.Hash)
	server.utxos.Remove(itx.MsgTx, server.contractAddresses)
	return server.Handler.Trigger(ctx, "STOLE", itx)
}

func (server *Server) revertTx(ctx context.Context, itx *inspector.Transaction) error {
	server.Tracer.RevertTx(ctx, itx.Hash)
	server.utxos.Remove(itx.MsgTx, server.contractAddresses)
	return server.Handler.Trigger(ctx, "LOST", itx)
}

func (server *Server) ReprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.Handler.Trigger(ctx, "END", itx)
}
