package spynode

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	handlerstorage "github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	MainNetBch wire.BitcoinNet = 0xe8f3e1e3
	TestNetBch wire.BitcoinNet = 0xf4f3e5f4
	RegTestBch wire.BitcoinNet = 0xfabfb5da

	SubSystem = "SpyNode" // For logger
)

// Node is the main object for spynode.
type Node struct {
	config         data.Config                        // Configuration
	state          *data.State                        // Non-persistent data
	store          storage.Storage                    // Persistent data
	peers          *handlerstorage.PeerRepository     // Peer data
	blocks         *handlerstorage.BlockRepository    // Block data
	txs            *handlerstorage.TxRepository       // Tx data
	reorgs         *handlerstorage.ReorgRepository    // Reorg data
	txTracker      *data.TxTracker                    // Tracks tx requests to ensure all txs are received
	memPool        *data.MemPool                      // Tracks which txs have been received and checked
	handlers       map[string]handlers.CommandHandler // Handlers for messages from trusted node
	connection     net.Conn                           // Connection to trusted node
	outgoing       chan wire.Message                  // Channel for messages to send to trusted node
	listeners      []handlers.Listener                // Receive data and notifications about transactions
	txFilters      []handlers.TxFilter                // Determines if a tx should be seen by listeners
	untrustedNodes []*UntrustedNode                   // Randomized peer connections to monitor for double spends
	addresses      map[string]time.Time               // Recently used peer addresses
	txChannel      chan *wire.MsgTx                   // Channel for directly handled txs so they don't lock the calling thread
	broadcastTx    *wire.MsgTx                        // Tx to transmit to nodes upon connection
	needsRestart   bool
	hardStop       bool
	stopping       bool
	stopped        bool
	lock           sync.Mutex
	untrustedLock  sync.Mutex
}

// NewNode creates a new node.
// See handlers/handlers.go for the listener interface definitions.
func NewNode(config data.Config, store storage.Storage) *Node {
	result := Node{
		config:         config,
		state:          data.NewState(),
		store:          store,
		peers:          handlerstorage.NewPeerRepository(store),
		blocks:         handlerstorage.NewBlockRepository(store),
		txs:            handlerstorage.NewTxRepository(store),
		reorgs:         handlerstorage.NewReorgRepository(store),
		txTracker:      data.NewTxTracker(),
		memPool:        data.NewMemPool(),
		outgoing:       nil,
		listeners:      make([]handlers.Listener, 0),
		txFilters:      make([]handlers.TxFilter, 0),
		untrustedNodes: make([]*UntrustedNode, 0),
		addresses:      make(map[string]time.Time),
		needsRestart:   false,
		hardStop:       false,
		stopping:       false,
		stopped:        false,
		txChannel:      nil,
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

// load loads the data for the node.
// Must be called after adding filter(s), but before Run()
func (node *Node) load(ctx context.Context) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	if err := node.peers.Load(ctx); err != nil {
		return err
	}
	logger.Verbose(ctx, "Loaded %d peers", node.peers.Count())

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
		node.blocks, node.txs, node.reorgs, node.txTracker, node.memPool, node.listeners,
		node.txFilters, node)
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
	for {
		if initial {
			logger.Verbose(ctx, "Connecting to %s", node.config.NodeAddress)
		} else {
			logger.Verbose(ctx, "Re-connecting to %s", node.config.NodeAddress)
			node.state.LogRestart()
		}
		if err = node.connect(ctx); err != nil {
			logger.Verbose(ctx, "Trusted connection failed to %s : %s", node.config.NodeAddress, err.Error())
			break
		}
		initial = false

		node.outgoing = make(chan wire.Message, 100)
		node.txChannel = make(chan *wire.MsgTx, 100)

		// Queue version message to start handshake
		version := buildVersionMsg(node.config.UserAgent, int32(node.blocks.LastHeight()))
		node.outgoing <- version

		wg := sync.WaitGroup{}
		wg.Add(6)

		go func() {
			defer wg.Done()
			node.monitorIncoming(ctx)
			logger.Debug(ctx, "Monitor incoming finished")
		}()

		go func() {
			defer wg.Done()
			node.monitorRequestTimeouts(ctx)
			logger.Debug(ctx, "Monitor request timeouts finished")
		}()

		go func() {
			defer wg.Done()
			node.sendOutgoing(ctx)
			logger.Debug(ctx, "Send outgoing finished")
		}()

		go func() {
			defer wg.Done()
			node.processTxs(ctx)
			logger.Debug(ctx, "Process txs finished")
		}()

		go func() {
			defer wg.Done()
			node.checkTxDelays(ctx)
			logger.Debug(ctx, "Check tx delays finished")
		}()

		if node.config.UntrustedCount == 0 {
			wg.Done()
			logger.Debug(ctx, "Monitor untrusted not started")
		} else {
			go func() {
				defer wg.Done()
				node.monitorUntrustedNodes(ctx)
				logger.Debug(ctx, "Monitor untrusted finished")
			}()
		}

		// Block until goroutines finish as a result of Stop()
		wg.Wait()

		// Save block repository
		logger.Verbose(ctx, "Saving")
		node.blocks.Save(ctx)
		node.txs.Save(ctx)

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
	node.hardStop = true
	err := node.requestStop(ctx)
	for !node.isStopped() {
		time.Sleep(100 * time.Millisecond)
	}
	return err
}

func (node *Node) requestStop(ctx context.Context) error {
	node.lock.Lock()
	defer node.lock.Unlock()

	if node.stopping || node.stopped {
		return nil
	}
	node.stopping = true
	close(node.outgoing)
	close(node.txChannel)
	return node.connection.Close()
}

// BroadcastTx broadcasts a tx to the network.
func (node *Node) BroadcastTx(ctx context.Context, tx *wire.MsgTx) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Info(ctx, "Broadcasting tx : %s", tx.TxHash())

	if node.isStopping() { // TODO Resolve issue when node is restarting
		return errors.New("Node inactive")
	}

	// Send to trusted node
	if !node.queueOutgoing(tx) {
		return errors.New("Node inactive")
	}

	// Send to untrusted nodes
	node.untrustedLock.Lock()
	for _, untrusted := range node.untrustedNodes {
		untrusted.BroadcastTx(ctx, tx)
	}
	node.untrustedLock.Unlock()
	return nil
}

// Scan opens a lot of connetions at once to try to find peers.
func (node *Node) Scan(ctx context.Context, connections int) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Info(ctx, "Scanning for peers")

	if err := node.load(ctx); err != nil {
		return err
	}

	node.config.Scanning = true
	wg := sync.WaitGroup{}
	count := 0

	peers, err := node.peers.GetUnchecked(ctx)
	if err != nil {
		return err
	}
	logger.Verbose(ctx, "Found %d peers with no score", len(peers))

	var address string
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))

	for !node.isStopping() && count < connections && len(peers) > 0 {
		// Pick peer randomly
		random := seed.Intn(len(peers))
		address = peers[random].Address

		// Remove this address and try again
		peers = append(peers[:random], peers[random+1:]...)

		// Attempt connection
		newNode := NewUntrustedNode(address, node.config, node.store, node.peers, node.blocks, node.txs, node.memPool, node.listeners, node.txFilters)
		node.untrustedLock.Lock()
		node.untrustedNodes = append(node.untrustedNodes, newNode)
		node.untrustedLock.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			newNode.Run(ctx)
		}()
		count++
	}

	node.sleepUntilStop(30) // Wait for handshake

	// Stop all
	node.untrustedLock.Lock()
	nodeCount := len(node.untrustedNodes)
	for _, untrusted := range node.untrustedNodes {
		untrusted.Stop(ctx)
	}
	node.untrustedLock.Unlock()

	logger.Verbose(ctx, "Waiting for %d untrusted nodes to finish", nodeCount)
	wg.Wait()
	node.peers.Save(ctx)
	return nil
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

// ShotgunTransmitTx broadcasts a tx to as many nodes as it can.
// Don't call Run when using this function. Just create a node and call this.
func (node *Node) ShotgunTransmitTx(ctx context.Context, tx *wire.MsgTx, sendCount int) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Info(ctx, "Shotgunning tx : %s", tx.TxHash())

	if err := node.load(ctx); err != nil {
		return err
	}

	node.config.ShotgunTx = tx
	wg := sync.WaitGroup{}
	count := 0

	// Try for peers with a good score
	for !node.isStopping() && count < sendCount {
		if node.addUntrustedNode(ctx, &wg, 5) {
			count++
		} else {
			break
		}
	}

	// Try for peers with a non negative score
	for !node.isStopping() && count < sendCount {
		if node.addUntrustedNode(ctx, &wg, 0) {
			count++
		} else {
			break
		}
	}

	node.sleepUntilStop(30) // Wait for handshake

	// Stop all
	node.untrustedLock.Lock()
	nodeCount := len(node.untrustedNodes)
	for _, untrusted := range node.untrustedNodes {
		untrusted.Stop(ctx)
	}
	node.untrustedLock.Unlock()

	logger.Verbose(ctx, "Waiting for %d untrusted nodes to finish", nodeCount)
	wg.Wait()
	node.peers.Save(ctx)
	return nil
}

// HandleTx processes a tx through spynode as if it came from the network.
// Used to feed "response" txs directly back through spynode.
func (node *Node) HandleTx(ctx context.Context, tx *wire.MsgTx) error {
	node.lock.Lock()
	defer node.lock.Unlock()

	if node.stopping { // TODO Resolve issue when node is restarting
		return errors.New("Node inactive")
	}

	node.txChannel <- tx

	return nil
}

// ProcessTxs pulls txs from the tx channel and processes them.
func (node *Node) processTxs(ctx context.Context) error {
	ctx = context.WithValue(ctx, handlers.DirectTxKey, true)
	for !node.isStopping() {
		for len(node.txChannel) > 0 {
			tx, ok := <-node.txChannel
			if !ok {
				break
			}
			logger.Info(ctx, "Directly handling tx : %s", tx.TxHash())
			if err := node.handleMessage(ctx, tx); err != nil {
				if tx.Command() == "reject" {
					logger.Warn(ctx, "Reject message : %s", err.Error())
					continue
				}
				logger.Info(ctx, "Failed to directly handle tx : %s", err.Error())
			}
		}

		if node.isStopping() {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// sendOutgoing waits for and sends outgoing messages
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) sendOutgoing(ctx context.Context) error {
	for !node.isStopping() {
		// Wait for outgoing message on channel
		msg, ok := <-node.outgoing

		if !ok || node.isStopping() {
			break
		}

		if err := sendAsync(ctx, node.connection, msg, wire.BitcoinNet(node.config.ChainParams.Net)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to send %s", msg.Command()))
		}
	}

	return nil
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
		return err
	}

	// Queue messages to be sent in response
	for _, response := range responses {
		if !node.queueOutgoing(response) {
			break
		}
	}

	return nil
}

// ProcessBlock is called when a block is being processed.
// Implements handlers.BlockProcessor interface
// It is responsible for any cleanup as a result of a block.
func (node *Node) ProcessBlock(ctx context.Context, block *wire.MsgBlock) error {
	logger.Debug(ctx, "Cleaning up after block : %s", block.BlockHash())
	txids, err := block.TxHashes()
	if err != nil {
		return err
	}

	node.txTracker.Remove(ctx, txids)

	node.untrustedLock.Lock()
	defer node.untrustedLock.Unlock()

	for _, untrusted := range node.untrustedNodes {
		untrusted.ProcessBlock(ctx, txids)
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
	return nil
}

// monitorIncoming processes incoming messages.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *Node) monitorIncoming(ctx context.Context) {
	for !node.isStopping() {
		if err := node.check(ctx); err != nil {
			logger.Warn(ctx, "Check failed : %s", err.Error())
			node.requestStop(ctx)
			break
		}

		// read new messages, blocking
		if node.isStopping() {
			break
		}
		msg, _, err := wire.ReadMessage(node.connection, wire.ProtocolVersion, wire.BitcoinNet(node.config.ChainParams.Net))
		if err == io.EOF {
			// Happens when the connection is closed
			logger.Verbose(ctx, "Connection closed")
			node.restart(ctx)
			break
		}
		if err != nil {
			// Happens when the connection is closed
			logger.Warn(ctx, "Failed to read message : %s", err.Error())
			continue
		}

		if err := node.handleMessage(ctx, msg); err != nil {
			if msg.Command() == "reject" {
				logger.Warn(ctx, "Reject message : %s", err.Error())
				continue
			}
			logger.Warn(ctx, "Failed to handle (%s) message : %s", msg.Command(), err.Error())
			node.requestStop(ctx)
			break
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

		if !node.queueOutgoing(headerRequest) {
			logger.Debug(ctx, "Requesting headers")
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
		node.sleepUntilStop(10) // Only check every 10 seconds

		if err := node.state.CheckTimeouts(); err != nil {
			logger.Warn(ctx, err.Error())
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
func (node *Node) checkTxDelays(ctx context.Context) error {
	for !node.isStopping() {
		time.Sleep(200 * time.Millisecond)

		if !node.state.IsReady() {
			continue
		}

		// Get newly safe txs
		cutoffTime := time.Now().Add(time.Millisecond * -time.Duration(node.config.SafeTxDelay))
		txids, err := node.txs.GetNewSafe(ctx, cutoffTime)
		if err != nil {
			logger.Warn(ctx, err.Error())
			node.restart(ctx)
			break
		}

		for _, txid := range txids {
			for _, listener := range node.listeners {
				listener.HandleTxState(ctx, handlers.ListenerMsgTxStateSafe, txid)
			}
		}
	}
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
			if untrusted.state.IsReady() {
				verifiedCount++
			}
		}

		node.untrustedLock.Unlock()

		if verifiedCount < node.config.UntrustedCount {
			logger.Debug(ctx, "Untrusted connections : %d", verifiedCount)
		}

		if count < node.config.UntrustedCount/2 {
			// Try for peers with a good score
			for !node.isStopping() && count < node.config.UntrustedCount/2 {
				if node.addUntrustedNode(ctx, &wg, 5) {
					count++
				} else {
					break
				}
			}
		}

		// Try for peers with a non-negative score
		for !node.isStopping() && count < node.config.UntrustedCount {
			if node.addUntrustedNode(ctx, &wg, 0) {
				count++
			} else {
				break
			}
		}

		node.peers.Save(ctx)

		if node.isStopping() {
			break
		}

		node.sleepUntilStop(5) // Only check every 5 seconds
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
func (node *Node) addUntrustedNode(ctx context.Context, wg *sync.WaitGroup, minScore int32) bool {
	// Get new address
	// Check we aren't already connected and haven't used it recently
	peers, err := node.peers.Get(ctx, minScore)
	if err != nil {
		return false
	}
	logger.Verbose(ctx, "Found %d peers with score %d", len(peers), minScore)

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
	newNode := NewUntrustedNode(address, node.config, node.store, node.peers, node.blocks, node.txs, node.memPool, node.listeners, node.txFilters)
	node.untrustedLock.Lock()
	node.untrustedNodes = append(node.untrustedNodes, newNode)
	node.untrustedLock.Unlock()
	wg.Add(1)
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

func (node *Node) sleepUntilStop(seconds int) {
	for i := 0; i < seconds; i++ {
		if node.isStopping() {
			break
		}
		time.Sleep(time.Second)
	}
}

// ------------------------------------------------------------------------------------------------
// BitcoinHeaders interface
func (node *Node) LastHeight(ctx context.Context) int {
	return node.blocks.LastHeight()
}

func (node *Node) Hash(ctx context.Context, height int) (*chainhash.Hash, error) {
	return node.blocks.Hash(ctx, height)
}

func (node *Node) Time(ctx context.Context, height int) (uint32, error) {
	return node.blocks.Time(ctx, height)
}
