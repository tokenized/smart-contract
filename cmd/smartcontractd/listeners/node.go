package listeners

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/spynode/pkg/client"

	"github.com/pkg/errors"
)

const (
	walletKey = "wallet" // storage path for wallet
	serverKey = "server" // storage path for server
)

var (
	// DefaultEndian specifies the order of bytes for encoding integers.
	DefaultEndian = binary.LittleEndian

	ErrDuplicateTx = errors.New("Duplicate Tx")
)

type Server struct {
	wallet            wallet.WalletInterface
	Config            *node.Config
	MasterDB          *db.DB
	SpyNode           client.Client
	Headers           node.BitcoinHeaders
	Scheduler         *scheduler.Scheduler
	Tracer            *filters.Tracer
	utxos             *utxos.UTXOs
	lock              sync.Mutex
	Handler           protomux.Handler
	contractAddresses []bitcoin.RawAddress // Used to determine which txs will be needed again
	walletLock        sync.RWMutex
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
	spyNode client.Client,
	headers node.BitcoinHeaders,
	sch *scheduler.Scheduler,
	tracer *filters.Tracer,
	utxos *utxos.UTXOs,
	holdingsChannel *holdings.CacheChannel,
) *Server {
	result := &Server{
		wallet:           wallet,
		Config:           config,
		MasterDB:         masterDB,
		SpyNode:          spyNode,
		Headers:          headers,
		Scheduler:        sch,
		Tracer:           tracer,
		Handler:          handler,
		utxos:            utxos,
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
	ctx = node.ContextWithLogTrace(ctx, "Load")
	b, err := server.MasterDB.Fetch(ctx, serverKey)
	if err == nil {
		if err := server.Deserialize(ctx, bytes.NewReader(b)); err != nil {
			return errors.Wrap(err, "deserialize server")
		}
	} else if err != db.ErrNotFound {
		return errors.Wrap(err, "fetch server")
	}

	// Set responder
	server.Handler.SetResponder(server.respondTx)
	server.Handler.SetReprocessor(server.reprocessTx)

	server.incomingTxs.Open(100)
	server.processingTxs.Open(100)
	server.holdingsChannel.Open(5000)

	// Register listeners
	if server.SpyNode != nil {
		server.SpyNode.RegisterHandler(server)
		server.SpyNode.SubscribeContracts(ctx)
	}

	if err := server.Tracer.Load(ctx, server.MasterDB); err != nil {
		return errors.Wrap(err, "load tracer")
	}

	return nil
}

func (server *Server) Run(ctx context.Context) error {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Scheduler.Run(ctx); err != nil {
			node.LogError(ctx, "Scheduler failed : %s", err)
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
			if err := server.ProcessIncomingTxs(ctx); err != nil {
				node.LogError(ctx, "Pre-process failed : %s", err)
				server.Scheduler.Stop(ctx)
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
			server.incomingTxs.Close()
			server.processingTxs.Close()
		}
		node.LogVerbose(ctx, "Process holdings cache thread finished")
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()

	return server.Save(ctx)
}

func (server *Server) Save(ctx context.Context) error {

	var buf bytes.Buffer
	if err := server.Serialize(&buf); err != nil {
		return errors.Wrap(err, "serialize server")
	}

	if err := server.MasterDB.Put(ctx, serverKey, buf.Bytes()); err != nil {
		return errors.Wrap(err, "put server")
	}

	if err := server.SaveWallet(ctx); err != nil {
		return errors.Wrap(err, "save wallet")
	}

	if err := server.Tracer.Save(ctx, server.MasterDB); err != nil {
		return errors.Wrap(err, "save tracer")
	}

	if server.SpyNode != nil {
		if err := state.SaveNextMessageID(ctx, server.MasterDB,
			server.SpyNode.NextMessageID()); err != nil {
			return errors.Wrap(err, "next message id")
		}
	}

	return nil
}

func (server *Server) Stop(ctx context.Context) error {
	schedulerErr := server.Scheduler.Stop(ctx)
	server.incomingTxs.Close()
	server.processingTxs.Close()
	server.holdingsChannel.Close()

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
		if err := server.SpyNode.SendTx(ctx, tx); err != nil {
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

	// Broadcast to network
	if err := server.sendTx(ctx, tx); err != nil {
		return err
	}

	return nil
}

func (server *Server) reprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return server.processingTxs.Add(ProcessingTx{Itx: itx, Event: "END"})
}

// removePendingRequests removes any pending requests that this tx is a response to.
// Contract responses use the tx output from the request to the contract as a tx input in the
// response tx. So if that contract request output is spent by another tx, then the contract has
// already responded.
func (server *Server) removePendingRequests(ctx context.Context, itx *inspector.Transaction) error {
	if !itx.IsOutgoingMessageType() {
		return nil
	}

	// Check each input for a link to a pending request.
	for _, input := range itx.MsgTx.TxIn {
		for i, pendingTx := range server.pendingRequests {
			if int(input.PreviousOutPoint.Index) == pendingTx.ContractIndex &&
				input.PreviousOutPoint.Hash.Equal(pendingTx.Itx.Hash) {

				node.Log(ctx, "Canceling pending request tx : %s", pendingTx.Itx.Hash.String())
				server.pendingRequests = append(server.pendingRequests[:i],
					server.pendingRequests[i+1:]...)
				break
			}
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

func (server *Server) Serialize(buf *bytes.Buffer) error {
	// Version
	if err := binary.Write(buf, DefaultEndian, uint8(1)); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := binary.Write(buf, DefaultEndian, uint32(len(server.pendingRequests))); err != nil {
		return errors.Wrap(err, "pending requests size")
	}
	for _, pr := range server.pendingRequests {
		if err := pr.Itx.Write(buf); err != nil {
			return errors.Wrap(err, "serialize pending request itx")
		}

		if err := binary.Write(buf, DefaultEndian, uint32(pr.ContractIndex)); err != nil {
			return errors.Wrap(err, "write pending request index")
		}
	}

	if err := binary.Write(buf, DefaultEndian, uint32(len(server.pendingResponses))); err != nil {
		return errors.Wrap(err, "pending responses size")
	}
	for _, itx := range server.pendingResponses {
		if err := itx.Write(buf); err != nil {
			return errors.Wrap(err, "serialize pending response itx")
		}
	}

	if err := binary.Write(buf, DefaultEndian, uint32(len(server.revertedTxs))); err != nil {
		return errors.Wrap(err, "reverted txs size")
	}
	for _, txid := range server.revertedTxs {
		if err := txid.Serialize(buf); err != nil {
			return errors.Wrap(err, "serialize reverted tx")
		}
	}

	if err := binary.Write(buf, DefaultEndian, uint32(len(server.pendingTxs))); err != nil {
		return errors.Wrap(err, "write pending tx count")
	}

	for hash, tx := range server.pendingTxs {
		if _, err := buf.Write(hash[:]); err != nil {
			return errors.Wrap(err, "write pending tx hash")
		}

		if err := tx.Serialize(buf); err != nil {
			return errors.Wrap(err, "write pending tx")
		}
	}

	if err := binary.Write(buf, DefaultEndian, uint32(len(server.readyTxs))); err != nil {
		return errors.Wrap(err, "write ready tx count")
	}

	for _, hash := range server.readyTxs {
		if _, err := buf.Write(hash[:]); err != nil {
			return errors.Wrap(err, "write ready tx hash")
		}
	}

	return nil
}

func (server *Server) Deserialize(ctx context.Context, buf *bytes.Reader) error {
	// Version
	var version uint8
	if err := binary.Read(buf, DefaultEndian, &version); err != nil {
		return errors.Wrap(err, "version")
	}

	if version != 0 && version != 1 {
		return fmt.Errorf("Unsupported version : %d", version)
	}

	var count uint32
	if err := binary.Read(buf, DefaultEndian, &count); err != nil {
		return errors.Wrap(err, "pending requests size")
	}
	server.pendingRequests = make([]pendingRequest, 0, count)
	for i := uint32(0); i < count; i++ {
		pr := pendingRequest{}
		pr.Itx = &inspector.Transaction{}
		if err := pr.Itx.Read(buf, server.Config.IsTest); err != nil {
			return errors.Wrap(err, "deserialize pending request itx")
		}

		var contractIndex uint32
		if err := binary.Read(buf, DefaultEndian, &contractIndex); err != nil {
			return errors.Wrap(err, "read pending request index")
		}
		pr.ContractIndex = int(contractIndex)
		server.pendingRequests = append(server.pendingRequests, pr)
		node.LogVerbose(ctx, "Loaded pending request : %s", pr.Itx.Hash.String())
	}

	if err := binary.Read(buf, DefaultEndian, &count); err != nil {
		return errors.Wrap(err, "pending responses size")
	}
	server.pendingResponses = make(inspector.TransactionList, 0, count)
	for i := uint32(0); i < count; i++ {
		var itx inspector.Transaction
		if err := itx.Read(buf, server.Config.IsTest); err != nil {
			return errors.Wrap(err, "deserialize pending response itx")
		}
		server.pendingResponses = append(server.pendingResponses, &itx)
		node.LogVerbose(ctx, "Loaded pending response : %s", itx.Hash.String())
	}

	if err := binary.Read(buf, DefaultEndian, &count); err != nil {
		return errors.Wrap(err, "reverted txs size")
	}
	server.revertedTxs = make([]*bitcoin.Hash32, 0, count)
	for i := uint32(0); i < count; i++ {
		var txid bitcoin.Hash32
		if err := txid.Deserialize(buf); err != nil {
			return errors.Wrap(err, "deserialize reverted tx")
		}
		server.revertedTxs = append(server.revertedTxs, &txid)
		node.LogVerbose(ctx, "Loaded reverted tx : %s", txid.String())
	}

	if version >= 1 {
		if err := binary.Read(buf, DefaultEndian, &count); err != nil {
			return errors.Wrap(err, "read pending tx count")
		}

		server.pendingTxs = make(map[bitcoin.Hash32]*IncomingTxData)
		for i := 0; i < int(count); i++ {
			var hash bitcoin.Hash32
			if _, err := buf.Read(hash[:]); err != nil {
				return errors.Wrap(err, "read pending tx hash")
			}

			var tx IncomingTxData
			if err := tx.Deserialize(buf, server.Config.IsTest); err != nil {
				return errors.Wrap(err, "read pending tx")
			}
			server.pendingTxs[hash] = &tx
			node.LogVerbose(ctx, "Loaded pending tx : %s", hash.String())
		}

		if err := binary.Read(buf, DefaultEndian, &count); err != nil {
			return errors.Wrap(err, "read ready tx count")
		}

		server.readyTxs = make([]*bitcoin.Hash32, 0, count)
		for i := 0; i < int(count); i++ {
			var hash bitcoin.Hash32
			if _, err := buf.Read(hash[:]); err != nil {
				return errors.Wrap(err, "read ready tx hash")
			}
			server.readyTxs = append(server.readyTxs, &hash)
			node.LogVerbose(ctx, "Loaded ready txid : %s", hash.String())
		}
	}

	return nil
}
