package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/internal/contract"
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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
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
	pendingResponses  []*wire.MsgTx
	revertedTxs       []*chainhash.Hash
	blockHeight       int // track current block height for confirm messages
	inSync            bool

	pendingTxs  map[chainhash.Hash]*IncomingTxData
	readyTxs    []*chainhash.Hash // Saves order of tx approval in case preprocessing doesn't finish before approval.
	pendingLock sync.Mutex

	incomingTxs   IncomingTxChannel
	processingTxs ProcessingTxChannel

	holdingsChannel *holdings.CacheChannel

	TxSentCount        int
	AlternateResponder protomux.ResponderFunc
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
		txFilter:         txFilter,
		pendingTxs:       make(map[chainhash.Hash]*IncomingTxData),
		pendingResponses: make([]*wire.MsgTx, 0),
		blockHeight:      0,
		inSync:           false,
		holdingsChannel:  holdingsChannel,
	}

	keys := wallet.ListAll()
	result.contractAddresses = make([]bitcoin.RawAddress, 0, len(keys))
	for _, key := range keys {
		address, err := bitcoin.NewAddressPKH(bitcoin.Hash160(key.Key.PublicKey().Bytes()),
			wire.BitcoinNet(config.ChainParams.Net))
		if err != nil {
			return nil
		}
		result.contractAddresses = append(result.contractAddresses, address)
	}

	return &result
}

func (server *Server) Run(ctx context.Context) error {
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
		return err
	}

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

	if err := server.wallet.Save(ctx, server.MasterDB, wire.BitcoinNet(server.Config.ChainParams.Net)); err != nil {
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

// respondTx is an internal method used as the responder
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
	server.utxos.Remove(itx.MsgTx, server.contractAddresses)
	return server.Handler.Trigger(ctx, "STOLE", itx)
}

func (server *Server) revertTx(ctx context.Context, itx *inspector.Transaction) error {
	server.Tracer.RevertTx(ctx, &itx.Hash)
	server.utxos.Remove(itx.MsgTx, server.contractAddresses)
	return server.Handler.Trigger(ctx, "LOST", itx)
}

func (server *Server) ReprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.Handler.Trigger(ctx, "REPROCESS", itx)
}

// AddContractKey adds a new contract key to those being monitored.
func (server *Server) AddContractKey(ctx context.Context, k bitcoin.Key) error {
	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	address, err := bitcoin.NewAddressPKH(bitcoin.Hash160(k.PublicKey().Bytes()), k.Network())
	if err != nil {
		return err
	}
	newKey := wallet.Key{
		Address: address,
		Key:     k,
	}

	node.Log(ctx, "Adding key : %x", bitcoin.Hash160(k.PublicKey().Bytes()))
	server.wallet.Add(&newKey)
	if err := server.wallet.Save(ctx, server.MasterDB, wire.BitcoinNet(server.Config.ChainParams.Net)); err != nil {
		return err
	}
	server.contractAddresses = append(server.contractAddresses, address)

	server.txFilter.AddPubKey(ctx, k.PublicKey().Bytes())
	return nil
}

// RemoveContractKeyIfUnused removes a contract key from those being monitored if it hasn't been used yet.
func (server *Server) RemoveContractKeyIfUnused(ctx context.Context, k bitcoin.Key) error {
	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	pkh := bitcoin.Hash160(k.PublicKey().Bytes())
	address, err := bitcoin.NewAddressPKH(pkh, k.Network())
	if err != nil {
		return err
	}
	newKey := wallet.Key{
		Address: address,
		Key:     k,
	}

	// Check if contract exists
	_, err = contract.Retrieve(ctx, server.MasterDB, address)
	if err != contract.ErrNotFound {
		return nil
	}

	stringAddress := bitcoin.NewAddressFromRawAddress(address, wire.BitcoinNet(server.Config.ChainParams.Net))
	node.Log(ctx, "Removing key : %s", stringAddress.String())
	server.wallet.Remove(&newKey)
	if err := server.wallet.Save(ctx, server.MasterDB, wire.BitcoinNet(server.Config.ChainParams.Net)); err != nil {
		return err
	}

	for i, caddress := range server.contractAddresses {
		if address.Equal(caddress) {
			server.contractAddresses = append(server.contractAddresses[:i], server.contractAddresses[i+1:]...)
			break
		}
	}

	server.txFilter.RemovePubKey(ctx, k.PublicKey().Bytes())
	return nil
}
