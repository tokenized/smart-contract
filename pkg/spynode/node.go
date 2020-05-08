package spynode

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	handlerstorage "github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	SubSystem = "SpyNode" // For logger
)

type TxCount struct {
	tx    *wire.MsgTx
	count int
}

// Node is the main object for spynode.
type Node struct {
	config          data.Config                        // Configuration
	state           *data.State                        // Non-persistent data
	store           storage.Storage                    // Persistent data
	peers           *handlerstorage.PeerRepository     // Peer data
	blocks          *handlerstorage.BlockRepository    // Block data
	blockRefeeder   handlers.BlockRefeeder             // Reprocess older blocks
	txs             *handlerstorage.TxRepository       // Tx data
	reorgs          *handlerstorage.ReorgRepository    // Reorg data
	txTracker       *data.TxTracker                    // Tracks tx requests to ensure all txs are received
	memPool         *data.MemPool                      // Tracks which txs have been received and checked
	handlers        map[string]handlers.CommandHandler // Handlers for messages from trusted node
	connection      net.Conn                           // Connection to trusted node
	outgoing        chan wire.Message                  // Channel for messages to send to trusted node
	listeners       []handlers.Listener                // Receive data and notifications about transactions
	txFilters       []handlers.TxFilter                // Determines if a tx should be seen by listeners
	untrustedNodes  []*UntrustedNode                   // Randomized peer connections to monitor for double spends
	addresses       map[string]time.Time               // Recently used peer addresses
	confTxChannel   handlers.TxChannel                 // Channel for directly handled txs so they don't lock the calling thread
	unconfTxChannel handlers.TxChannel                 // Channel for directly handled txs so they don't lock the calling thread
	txStateChannel  handlers.TxStateChannel            // Channel for tx states so they are in one thread
	broadcastLock   sync.Mutex
	broadcastTxs    []TxCount // Txs to transmit to nodes upon connection
	needsRestart    bool
	hardStop        bool
	stopping        bool
	stopped         bool
	scanning        bool
	attempts        int // Count of re-connect attempts without completing handshake.
	lock            sync.Mutex
	untrustedLock   sync.Mutex
	blockLock       sync.Mutex
}

// NewNode creates a new node.
// See handlers/handlers.go for the listener interface definitions.
func NewNode(config data.Config, store storage.Storage) *Node {
	result := Node{
		config:          config,
		state:           data.NewState(),
		store:           store,
		peers:           handlerstorage.NewPeerRepository(store),
		blocks:          handlerstorage.NewBlockRepository(&config, store),
		txs:             handlerstorage.NewTxRepository(store),
		reorgs:          handlerstorage.NewReorgRepository(store),
		txTracker:       data.NewTxTracker(),
		memPool:         data.NewMemPool(),
		outgoing:        nil,
		listeners:       make([]handlers.Listener, 0),
		txFilters:       make([]handlers.TxFilter, 0),
		untrustedNodes:  make([]*UntrustedNode, 0),
		addresses:       make(map[string]time.Time),
		needsRestart:    false,
		hardStop:        false,
		stopping:        false,
		stopped:         false,
		confTxChannel:   handlers.TxChannel{},
		unconfTxChannel: handlers.TxChannel{},
	}
	return &result
}

func (node *Node) RegisterListener(listener handlers.Listener) {
	node.listeners = append(node.listeners, listener)
}

// AddTxFilter adds a tx filter.
// See handlers/filters.go for specification of a filter.
// If no tx filters, then all txs are sent to listeners.
// If any of the tx filters return true the tx will be sent to listeners.
func (node *Node) AddTxFilter(filter handlers.TxFilter) {
	node.txFilters = append(node.txFilters, filter)
}

// SetupRetry configures the maximum connection retries and delay in milliseconds between each
//   attempt.
func (node *Node) SetupRetry(max, delay int) {
	node.config.MaxRetries = max
	node.config.RetryDelay = delay
}

// load loads the data for the node.
// Must be called after adding filter(s), but before Run()
func (node *Node) load(ctx context.Context) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	if err := node.peers.Load(ctx); err != nil {
		return err
	}

	if err := node.blocks.Load(ctx); err != nil {
		return err
	}
	logger.Info(ctx, "Loaded blocks to height %d", node.blocks.LastHeight())
	startHeight, exists := node.blocks.Height(&node.config.StartHash)
	if exists {
		node.state.SetStartHeight(startHeight)
		logger.Info(ctx, "Start block height %d", startHeight)
	} else {
		logger.Info(ctx, "Start block not found yet")
	}

	if err := node.txs.Load(ctx); err != nil {
		return err
	}

	node.handlers = handlers.NewTrustedCommandHandlers(ctx, node.config, node.state, node.peers,
		node.blocks, &node.blockRefeeder, node.txs, node.reorgs, node.txTracker, node.memPool,
		&node.unconfTxChannel, &node.txStateChannel, node.listeners, node.txFilters)
	return nil
}

// Run runs the node.
// Doesn't stop until there is a failure or Stop() is called.
func (node *Node) Run(ctx context.Context) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)

	defer func() {
		node.lock.Lock()
		node.stopped = true
		node.lock.Unlock()
		logger.Verbose(ctx, "Stopped")
	}()

	var err error = nil
	if err = node.load(ctx); err != nil {
		return err
	}

	initial := true
	for !node.isStopping() {
		if node.attempts != 0 {
			time.Sleep(time.Duration(node.config.RetryDelay) * time.Millisecond)
		}
		if node.attempts > node.config.MaxRetries {
			logger.Error(ctx, "SpyNodeAborted trusted connection to %s", node.config.NodeAddress)
		}
		node.attempts++

		if initial {
			logger.Verbose(ctx, "Connecting to %s", node.config.NodeAddress)
		} else {
			logger.Verbose(ctx, "Re-connecting to %s", node.config.NodeAddress)
		}
		initial = false
		if err = node.connect(ctx); err != nil {
			logger.Error(ctx, "SpyNodeFailed trusted connection to %s : %s",
				node.config.NodeAddress, err.Error())
			continue
		}

		node.outgoing = make(chan wire.Message, 100)
		node.confTxChannel.Open(100)
		node.unconfTxChannel.Open(100)
		node.txStateChannel.Open(1000)

		// Queue version message to start handshake
		version := buildVersionMsg(node.config.UserAgent, int32(node.blocks.LastHeight()))
		node.outgoing <- version

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.monitorIncoming(ctx)
			logger.Verbose(ctx, "Monitor incoming finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.monitorRequestTimeouts(ctx)
			logger.Verbose(ctx, "Monitor request timeouts finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.sendOutgoing(ctx)
			logger.Verbose(ctx, "Send outgoing finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.processBlocks(ctx)
			logger.Verbose(ctx, "Process blocks finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.processConfirmedTxs(ctx)
			logger.Verbose(ctx, "Process confirmed txs finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.processUnconfirmedTxs(ctx)
			logger.Verbose(ctx, "Process unconfirmed txs finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.processTxStates(ctx)
			logger.Verbose(ctx, "Process tx states finished")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			node.checkTxDelays(ctx)
			logger.Verbose(ctx, "Check tx delays finished")
		}()

		if node.config.UntrustedCount == 0 {
			logger.Verbose(ctx, "Monitor untrusted not started")
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				node.monitorUntrustedNodes(ctx)
				logger.Verbose(ctx, "Monitor untrusted finished")
			}()
		}

		// Block until goroutines finish as a result of Stop()
		wg.Wait()

		// Save block repository
		logger.Verbose(ctx, "Saving")
		node.blocks.Save(ctx)
		node.txs.Save(ctx)
		node.peers.Save(ctx)

		if !node.needsRestart || node.hardStop {
			break
		}

		logger.Verbose(ctx, "Restarting")
		node.needsRestart = false
		node.lock.Lock()
		node.stopping = false
		node.lock.Unlock()
		node.state.Reset()
	}

	return err
}

func (node *Node) isStopped() bool {
	node.lock.Lock()
	defer node.lock.Unlock()

	return node.stopped
}

func (node *Node) isStopping() bool {
	node.lock.Lock()
	defer node.lock.Unlock()

	return node.stopping
}

// Stop closes the connection and causes Run() to return.
func (node *Node) Stop(ctx context.Context) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	node.lock.Lock()
	done := node.stopped
	node.lock.Unlock()
	if done {
		return nil
	}

	node.hardStop = true
	err := node.requestStop(ctx)
	count := 0
	for !node.isStopped() {
		time.Sleep(100 * time.Millisecond)
		if count > 30 { // 3 seconds
			logger.Info(ctx, "Waiting for spynode to stop")
			count = 0
		}
		count++
	}
	return err
}

func (node *Node) requestStop(ctx context.Context) error {
	logger.Verbose(ctx, "Requesting stop")
	node.lock.Lock()
	defer node.lock.Unlock()

	if node.stopping || node.stopped {
		return nil
	}
	logger.Info(ctx, "Stopping")
	node.stopping = true
	if node.outgoing != nil {
		close(node.outgoing)
		node.outgoing = nil
	}
	node.confTxChannel.Close()
	node.unconfTxChannel.Close()
	node.txStateChannel.Close()
	if node.connection != nil {
		node.connection.Close()
	}
	return nil
}

func (node *Node) OutgoingCount() int {
	node.untrustedLock.Lock()
	defer node.untrustedLock.Unlock()

	result := 0
	for _, untrusted := range node.untrustedNodes {
		if untrusted.IsReady() {
			result++
		}
	}
	return result
}

// BroadcastTx broadcasts a tx to the network.
func (node *Node) BroadcastTx(ctx context.Context, tx *wire.MsgTx) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Info(ctx, "Broadcasting tx : %s", tx.TxHash())

	if node.isStopping() { // TODO Resolve issue when node is restarting
		return errors.New("Node inactive")
	}

	// Wait for ready
	for i := 0; i < 100; i++ {
		if node.isStopping() { // TODO Resolve issue when node is restarting
			return errors.New("Node inactive")
		}
		if node.state.IsReady() {
			break
		}

		time.Sleep(250 * time.Millisecond)
	}

	count := 1

	// Send to trusted node
	if !node.queueOutgoing(tx) {
		return errors.New("Node inactive")
	}

	// Send to untrusted nodes
	node.untrustedLock.Lock()
	for _, untrusted := range node.untrustedNodes {
		if untrusted.IsReady() {
			if err := untrusted.BroadcastTxs(ctx, []*wire.MsgTx{tx}); err != nil {
				logger.Warn(ctx, "Failed to broadcast tx to untrusted : %s", err)
			} else {
				count++
			}
		}
	}
	node.untrustedLock.Unlock()

	if count < node.config.ShotgunCount {
		node.broadcastLock.Lock()
		node.broadcastTxs = append(node.broadcastTxs, TxCount{tx: tx, count: count})
		node.broadcastLock.Unlock()
	}
	return nil
}

func (node *Node) BroadcastIsComplete(ctx context.Context) bool {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	if node.isStopping() {
		return true
	}

	node.broadcastLock.Lock()
	count := len(node.broadcastTxs)
	node.broadcastLock.Unlock()

	logger.Info(ctx, "%d broadcast txs remaining", count)
	return count == 0
}

func (node *Node) IsReady(ctx context.Context) bool {
	return node.state.IsReady()
}

// BroadcastTxUntrustedOnly broadcasts a tx to the network.
func (node *Node) BroadcastTxUntrustedOnly(ctx context.Context, tx *wire.MsgTx) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Info(ctx, "Broadcasting tx to untrusted only : %s", tx.TxHash())

	if node.isStopping() { // TODO Resolve issue when node is restarting
		return errors.New("Node inactive")
	}

	// Send to untrusted nodes
	node.untrustedLock.Lock()
	defer node.untrustedLock.Unlock()

	for _, untrusted := range node.untrustedNodes {
		if untrusted.IsReady() {
			untrusted.BroadcastTxs(ctx, []*wire.MsgTx{tx})
		}
	}
	return nil
}

// Scan opens a lot of connections at once to try to find peers.
func (node *Node) Scan(ctx context.Context, connections int) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)

	if err := node.load(ctx); err != nil {
		return err
	}

	if err := node.scan(ctx, connections, 1); err != nil {
		return err
	}

	return node.peers.Save(ctx)
}

// AddPeer adds a peer to the database with a specific score.
func (node *Node) AddPeer(ctx context.Context, address string, score int) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	if err := node.load(ctx); err != nil {
		return err
	}
	if _, err := node.peers.Add(ctx, address); err != nil {
		return err
	}

	if !node.peers.UpdateScore(ctx, address, int32(score)) {
		return errors.New("Failed to update score")
	}

	return node.peers.Save(ctx)
}

// sendOutgoing waits for and sends outgoing messages
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) sendOutgoing(ctx context.Context) {
	// Wait for outgoing messages on channel
	for msg := range node.outgoing {
		if node.isStopping() {
			break
		}

		tx, ok := msg.(*wire.MsgTx)
		if ok {
			logger.Verbose(ctx, "Sending Tx : %s", tx.TxHash().String())
		}

		if err := sendAsync(ctx, node.connection, msg, wire.BitcoinNet(node.config.Net)); err != nil {
			logger.Error(ctx, "SpyNodeFailed to send %s message : %s", msg.Command(), err)
			node.restart(ctx)
			break
		}
	}
}

// handleMessage Processes an incoming message
func (node *Node) handleMessage(ctx context.Context, msg wire.Message) error {
	if node.isStopping() {
		return nil
	}

	handler, ok := node.handlers[msg.Command()]
	if !ok {
		// no handler for this command
		return nil
	}

	responses, err := handler.Handle(ctx, msg)
	if err != nil {
		logger.Warn(ctx, "Failed to handle [%s] message : %s", msg.Command(), err)
		return nil
	}

	// Queue messages to be sent in response
	for _, response := range responses {
		if !node.queueOutgoing(response) {
			break
		}
	}

	return nil
}

// CleanupBlock is called when a block is being processed.
// Implements handlers.BlockProcessor interface
// It is responsible for any cleanup as a result of a block.
func (node *Node) CleanupBlock(ctx context.Context, block *wire.MsgBlock) error {
	logger.Debug(ctx, "Cleaning up after block : %s", block.BlockHash())
	txids, err := block.TxHashes()
	if err != nil {
		return err
	}

	node.txTracker.RemoveList(ctx, txids)

	node.untrustedLock.Lock()
	defer node.untrustedLock.Unlock()

	for _, untrusted := range node.untrustedNodes {
		untrusted.CleanupBlock(ctx, txids)
	}

	return nil
}

func (node *Node) connect(ctx context.Context) error {
	conn, err := net.Dial("tcp", node.config.NodeAddress)
	if err != nil {
		return err
	}

	node.connection = conn
	node.state.MarkConnected()
	node.peers.UpdateTime(ctx, node.config.NodeAddress)
	return nil
}

// monitorIncoming processes incoming messages.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) monitorIncoming(ctx context.Context) {
	for !node.isStopping() {
		if err := node.check(ctx); err != nil {
			logger.Error(ctx, "SpyNodeAborted check : %s", err.Error())
			node.requestStop(ctx)
			break
		}

		// read new messages, blocking
		if node.isStopping() {
			break
		}
		msg, _, err := wire.ReadMessage(node.connection, wire.ProtocolVersion,
			wire.BitcoinNet(node.config.Net))
		if err != nil {
			wireError, ok := err.(*wire.MessageError)
			if ok {
				if wireError.Type == wire.MessageErrorUnknownCommand {
					logger.Verbose(ctx, wireError.Error())
					continue
				} else {
					logger.Error(ctx, "SpyNodeFailed read message (wireError) : %s", wireError)
					node.restart(ctx)
					break
				}

			} else {
				logger.Error(ctx, "SpyNodeFailed to read message : %s", err)
				node.restart(ctx)
				break
			}
		}

		if err := node.handleMessage(ctx, msg); err != nil {
			logger.Error(ctx, "SpyNodeAborted to handle [%s] message : %s", msg.Command(),
				err.Error())
			node.requestStop(ctx)
			break
		}
		if msg.Command() == "reject" {
			reject, ok := msg.(*wire.MsgReject)
			if ok {
				logger.Warn(ctx, "(%s) Reject message : %s - %s", node.config.NodeAddress,
					reject.Reason, reject.Hash.String())
			}
		}
	}
}

func (node *Node) restart(ctx context.Context) {
	if node.isStopping() {
		return
	}
	node.needsRestart = true
	node.requestStop(ctx)
}

func (node *Node) queueOutgoing(msg wire.Message) bool {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.stopping {
		return false
	}
	node.outgoing <- msg
	return true
}

// check checks the state of spynode and performs state related actions.
func (node *Node) check(ctx context.Context) error {
	if !node.state.VersionReceived() {
		return nil // Still performing handshake
	}

	if !node.state.HandshakeComplete() {
		// Send header request to kick off sync
		headerRequest, err := buildHeaderRequest(ctx, node.state.ProtocolVersion(), node.blocks, 0, 50)
		if err != nil {
			return err
		}

		if node.queueOutgoing(headerRequest) {
			logger.Debug(ctx, "Requesting headers")
			node.state.MarkHeadersRequested()
			node.state.SetHandshakeComplete()
		}
	}

	// Check sync
	if node.state.IsReady() {
		node.attempts = 0

		if !node.state.SentSendHeaders() {
			// Send sendheaders message to get headers instead of block inventories.
			sendheaders := wire.NewMsgSendHeaders()
			if node.queueOutgoing(sendheaders) {
				node.state.SetSentSendHeaders()
			}
		}

		if !node.state.AddressesRequested() {
			addresses := wire.NewMsgGetAddr()
			if node.queueOutgoing(addresses) {
				node.state.SetAddressesRequested()
			}
		}

		if !node.state.MemPoolRequested() {
			// Send mempool request
			// This tells the peer to send inventory of all tx in their mempool.
			mempool := wire.NewMsgMemPool()
			if node.queueOutgoing(mempool) {
				node.state.SetMemPoolRequested()
			}
		} else {
			if !node.state.WasInSync() {
				node.reorgs.ClearActive(ctx)
				node.state.SetWasInSync()
			}

			if !node.state.NotifiedSync() {
				// TODO Add method to wait for mempool to sync
				for _, listener := range node.listeners {
					listener.HandleInSync(ctx)
				}
				node.state.SetNotifiedSync()
			}
		}

		responses, err := node.txTracker.Check(ctx, node.memPool)
		if err != nil {
			return err
		}
		// Queue messages to be sent in response
		for _, response := range responses {
			if !node.queueOutgoing(response) {
				break
			}
		}
	} else if node.state.HeadersRequested() == nil && node.state.TotalBlockRequestCount() < 5 {
		// Request more headers
		headerRequest, err := buildHeaderRequest(ctx, node.state.ProtocolVersion(), node.blocks, 1, 50)
		if err != nil {
			return err
		}

		if node.queueOutgoing(headerRequest) {
			logger.Debug(ctx, "Requesting headers after : %s", headerRequest.BlockLocatorHashes[0])
			node.state.MarkHeadersRequested()
		}
	}

	return nil
}

// monitorRequestTimeouts monitors for request timeouts.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) monitorRequestTimeouts(ctx context.Context) {
	for !node.isStopping() {
		node.sleepUntilStop(50) // Only check every 5 seconds

		if err := node.state.CheckTimeouts(); err != nil {
			logger.Error(ctx, "SpyNodeFailed timeouts : %s", err)
			node.restart(ctx)
			break
		}
	}
}

// checkTxDelays monitors txs for when they have passed the safe tx delay without seeing a
//   conflicting tx.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) checkTxDelays(ctx context.Context) {
	logger.Info(ctx, "Safe tx delay : %d ms", node.config.SafeTxDelay)
	for !node.isStopping() {
		time.Sleep(100 * time.Millisecond)

		if !node.state.IsReady() {
			continue
		}

		// Get newly safe txs
		cutoffTime := time.Now().Add(time.Millisecond * -time.Duration(node.config.SafeTxDelay))
		txids, err := node.txs.GetNewSafe(ctx, node.memPool, cutoffTime)
		if err != nil {
			logger.Error(ctx, "SpyNodeFailed GetNewSafe : %s", err)
			node.restart(ctx)
			break
		}

		for _, txid := range txids {
			logger.Debug(ctx, "Tx is now safe : %s", txid.String())
			node.txStateChannel.Add(handlers.TxState{
				handlers.ListenerMsgTxStateSafe,
				txid,
			})
		}
	}
}

// Scan opens a lot of connections at once to try to find peers.
func (node *Node) scan(ctx context.Context, connections, uncheckedCount int) error {
	if node.scanning {
		return nil
	}
	node.scanning = true

	ctx = logger.ContextWithLogTrace(ctx, "scan")

	peers, err := node.peers.GetUnchecked(ctx)
	if err != nil {
		return err
	}
	logger.Verbose(ctx, "Found %d peers with no score", len(peers))
	if len(peers) < uncheckedCount {
		return nil // Not enough unchecked peers to run a scan
	}

	logger.Verbose(ctx, "Scanning %d peers", connections)

	count := 0
	nodes := make([]*UntrustedNode, 0, connections)
	wg := sync.WaitGroup{}
	var address string
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))

	for !node.isStopping() && count < connections && len(peers) > 0 {
		// Pick peer randomly
		random := seed.Intn(len(peers))
		address = peers[random].Address

		// Remove this address and try again
		peers = append(peers[:random], peers[random+1:]...)

		// Attempt connection
		newNode := NewUntrustedNode(address, node.config.Copy(), node.state, node.store, node.peers,
			node.blocks, node.txs, node.memPool, &node.unconfTxChannel, node.listeners,
			node.txFilters, true)
		nodes = append(nodes, newNode)
		wg.Add(1)
		go func() {
			defer wg.Done()
			newNode.Run(ctx)
		}()
		count++
	}

	node.sleepUntilStop(100) // Wait for handshake

	for _, node := range nodes {
		node.Stop(ctx)
	}

	logger.Verbose(ctx, "Waiting for %d scanning nodes to stop", len(nodes))
	wg.Wait()
	node.scanning = false
	logger.Verbose(ctx, "Finished scanning")
	return nil
}

// monitorUntrustedNodes monitors untrusted nodes.
// Attempt to keep the specified number running.
// Watch for when they become inactive and replace them.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) monitorUntrustedNodes(ctx context.Context) {
	wg := sync.WaitGroup{}
	for !node.isStopping() {
		if !node.state.IsReady() {
			node.sleepUntilStop(5)
			continue
		}

		node.broadcastLock.Lock()
		broadcastCount := len(node.broadcastTxs)
		node.broadcastLock.Unlock()

		if broadcastCount == 0 {
			node.scan(ctx, 1000, 1)
		}
		if node.isStopping() {
			break
		}

		node.untrustedLock.Lock()
		if !node.state.IsReady() {
			node.untrustedLock.Unlock()
			node.sleepUntilStop(5)
			continue
		}

		// Check for inactive
		for {
			removed := false
			for i, untrusted := range node.untrustedNodes {
				if !untrusted.IsActive() {
					// Remove
					node.untrustedNodes = append(node.untrustedNodes[:i], node.untrustedNodes[i+1:]...)
					removed = true
					break
				}
			}

			if !removed {
				break
			}
		}

		count := len(node.untrustedNodes)
		verifiedCount := 0
		for _, untrusted := range node.untrustedNodes {
			if untrusted.untrustedState.IsReady() {
				verifiedCount++
			}
		}

		desiredCount := node.config.UntrustedCount

		node.untrustedLock.Unlock()

		var txs []*wire.MsgTx
		sentCount := 0
		node.broadcastLock.Lock()
		if len(node.broadcastTxs) > 0 {
			desiredCount = node.config.ShotgunCount
			txs = make([]*wire.MsgTx, 0, len(node.broadcastTxs))
			for _, btx := range node.broadcastTxs {
				txs = append(txs, btx.tx)
			}
		}
		node.broadcastLock.Unlock()

		if verifiedCount < desiredCount {
			logger.Debug(ctx, "Untrusted connections : %d", verifiedCount)
		}

		if count < desiredCount/2 {
			// Try for peers with a good score
			for !node.isStopping() && count < desiredCount/2 {
				if node.addUntrustedNode(ctx, &wg, 5, txs) {
					count++
					sentCount++
				} else {
					break
				}
			}
		}

		// Try for peers with a score above zero
		for !node.isStopping() && count < desiredCount {
			if node.addUntrustedNode(ctx, &wg, 1, txs) {
				count++
				sentCount++
			} else {
				break
			}
		}

		if node.isStopping() {
			break
		}

		if sentCount > 0 {
			for _, tx := range txs {
				node.broadcastLock.Lock()
				for i, btx := range node.broadcastTxs {
					if tx == btx.tx {
						node.broadcastTxs[i].count += sentCount
						if node.broadcastTxs[i].count > node.config.ShotgunCount {
							// tx has been sent to enough nodes. remove it
							node.broadcastTxs = append(node.broadcastTxs[:1],
								node.broadcastTxs[i+1:]...)
						}
						break
					}
				}
				node.broadcastLock.Unlock()
			}
		}

		node.sleepUntilStop(20) // Only check every 2 seconds
	}

	// Stop all
	node.untrustedLock.Lock()
	for _, untrusted := range node.untrustedNodes {
		untrusted.Stop(ctx)
	}
	node.untrustedLock.Unlock()

	logger.Verbose(ctx, "Waiting for %d untrusted nodes to finish", len(node.untrustedNodes))
	wg.Wait()
}

// addUntrustedNode adds a new untrusted node.
// Returns true if a new node connection was attempted
func (node *Node) addUntrustedNode(ctx context.Context, wg *sync.WaitGroup, minScore int32,
	txs []*wire.MsgTx) bool {

	// Get new address
	// Check we aren't already connected and haven't used it recently
	peers, err := node.peers.Get(ctx, minScore)
	if err != nil {
		return false
	}
	logger.Debug(ctx, "Found %d peers with score %d", len(peers), minScore)

	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	var address string
	for {
		if node.isStopping() || len(peers) == 0 {
			return false
		}

		if len(peers) == 1 {
			if node.checkAddress(ctx, peers[0].Address) {
				address = peers[0].Address
				break
			} else {
				return false
			}
		}

		// Pick one randomly
		random := seed.Intn(len(peers))
		if node.checkAddress(ctx, peers[random].Address) {
			address = peers[random].Address
			break
		}

		// Remove this address and try again
		peers = append(peers[:random], peers[random+1:]...)
	}

	// Attempt connection
	newNode := NewUntrustedNode(address, node.config.Copy(), node.state, node.store, node.peers,
		node.blocks, node.txs, node.memPool, &node.unconfTxChannel, node.listeners, node.txFilters,
		false)
	if txs != nil {
		newNode.BroadcastTxs(ctx, txs)
	}
	node.untrustedLock.Lock()
	node.untrustedNodes = append(node.untrustedNodes, newNode)
	wg.Add(1)
	node.untrustedLock.Unlock()
	go func() {
		defer wg.Done()
		newNode.Run(ctx)
	}()
	return true
}

// checkAddress checks if an address was recently used.
func (node *Node) checkAddress(ctx context.Context, address string) bool {
	lastUsed, exists := node.addresses[address]
	if exists {
		if time.Now().Sub(lastUsed).Minutes() > 10 {
			// Address hasn't been used for a while
			node.addresses[address] = time.Now()
			return true
		}

		// Address was used recently
		return false
	}

	// Add address
	node.addresses[address] = time.Now()
	return true
}

func (node *Node) sleepUntilStop(deciseconds int) {
	for i := 0; i < deciseconds; i++ {
		if node.isStopping() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (node *Node) RefeedBlocksFromHeight(ctx context.Context, height int) error {
	hash, err := node.Hash(ctx, height)
	if err != nil {
		return errors.Wrap(err, "get hash")
	}

	node.blockRefeeder.SetHeight(height, *hash)
	return nil
}

// ------------------------------------------------------------------------------------------------
// BitcoinHeaders interface
func (node *Node) LastHeight(ctx context.Context) int {
	return node.blocks.LastHeight()
}

func (node *Node) Hash(ctx context.Context, height int) (*bitcoin.Hash32, error) {
	return node.blocks.Hash(ctx, height)
}

func (node *Node) Time(ctx context.Context, height int) (uint32, error) {
	return node.blocks.Time(ctx, height)
}
