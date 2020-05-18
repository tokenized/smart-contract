package spynode

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	handlerStorage "github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

type UntrustedNode struct {
	address         string
	config          data.Config
	trustedState    *data.State
	untrustedState  *data.UntrustedState
	peers           *handlerStorage.PeerRepository
	blocks          *handlerStorage.BlockRepository
	txs             *handlerStorage.TxRepository
	txTracker       *data.TxTracker
	memPool         *data.MemPool
	txChannel       *handlers.TxChannel
	handlers        map[string]handlers.CommandHandler
	connection      net.Conn
	outgoing        MessageChannel
	listeners       []handlers.Listener
	txFilters       []handlers.TxFilter
	stopping        bool
	active          bool // Set to false when connection is closed
	scanning        bool
	readyAnnounced  bool
	pendingLock     sync.Mutex
	pendingOutgoing []*wire.MsgTx
	lock            sync.Mutex

	// These counts are used to monitor the number of threads active in specific categories.
	// They are used to stop the incoming threads before stopping the processing threads to
	//   prevent the incoming threads from filling channels and getting locked.
	incomingCount   uint32
	processingCount uint32
}

func NewUntrustedNode(address string, config data.Config, state *data.State, store storage.Storage,
	peers *handlerStorage.PeerRepository, blocks *handlerStorage.BlockRepository,
	txs *handlerStorage.TxRepository, memPool *data.MemPool, txChannel *handlers.TxChannel,
	listeners []handlers.Listener, txFilters []handlers.TxFilter, scanning bool) *UntrustedNode {

	result := UntrustedNode{
		address:        address,
		config:         config,
		trustedState:   state,
		untrustedState: data.NewUntrustedState(),
		peers:          peers,
		blocks:         blocks,
		txs:            txs,
		txTracker:      data.NewTxTracker(),
		memPool:        memPool,
		listeners:      listeners,
		txFilters:      txFilters,
		txChannel:      txChannel,
		stopping:       false,
		active:         false,
		scanning:       scanning,
	}

	atomic.StoreUint32(&result.incomingCount, 0)
	atomic.StoreUint32(&result.processingCount, 0)

	return &result
}

// Run the node
// Doesn't stop until there is a failure or Stop() is called.
func (node *UntrustedNode) Run(ctx context.Context) error {
	node.lock.Lock()
	if node.stopping {
		node.lock.Unlock()
		return nil
	}

	node.handlers = handlers.NewUntrustedCommandHandlers(ctx, node.trustedState,
		node.untrustedState, node.peers, node.blocks, node.txs, node.txTracker, node.memPool,
		node.txChannel, node.listeners, node.txFilters, node.address)

	if err := node.connect(); err != nil {
		node.lock.Unlock()
		node.peers.UpdateScore(ctx, node.address, -1)
		logger.Debug(ctx, "(%s) Connection failed : %s", node.address, err.Error())
		return err
	}

	logger.Debug(ctx, "(%s) Starting", node.address)
	node.peers.UpdateTime(ctx, node.address)
	node.outgoing.Open(100)
	node.active = true
	node.lock.Unlock()

	// Queue version message to start handshake
	version := buildVersionMsg(node.config.UserAgent, int32(node.blocks.LastHeight()))
	node.outgoing.Add(version)

	go func() {
		atomic.AddUint32(&node.incomingCount, 1)                // increment
		defer atomic.AddUint32(&node.incomingCount, ^uint32(0)) // decrement
		node.monitorIncoming(ctx)
		logger.Debug(ctx, "(%s) Untrusted monitor incoming finished", node.address)
	}()

	go func() {
		atomic.AddUint32(&node.incomingCount, 1)                // increment
		defer atomic.AddUint32(&node.incomingCount, ^uint32(0)) // decrement
		node.monitorRequestTimeouts(ctx)
		logger.Debug(ctx, "(%s) Untrusted monitor request timeouts finished", node.address)
	}()

	go func() {
		atomic.AddUint32(&node.processingCount, 1)                // increment
		defer atomic.AddUint32(&node.processingCount, ^uint32(0)) // decrement
		node.sendOutgoing(ctx)
		logger.Debug(ctx, "(%s) Untrusted send outgoing finished", node.address)
	}()

	// Block until goroutines finish as a result of Stop()

	// Phased shutdown
	for !node.isStopping() {
		time.Sleep(100 * time.Millisecond)
	}

	logger.Debug(ctx, "(%s) Stopping", node.address)

	node.lock.Lock()
	if node.connection != nil {
		if err := node.connection.Close(); err != nil {
			logger.Warn(ctx, "(%s) Failed to close : %s", node.address, err)
		}
		node.connection = nil
	}
	node.lock.Unlock()

	// Wait for incoming threads to stop.
	// We have to be sure that we stop writing to channels before we stop reading from channels or
	//   we can get stuck in a lock trying to write to a full channel.
	waitCount := 0
	for {
		time.Sleep(100 * time.Millisecond)
		incomingCount := atomic.LoadUint32(&node.incomingCount)
		if incomingCount == 0 {
			break
		}
		if waitCount > 30 { // 3 seconds
			logger.Info(ctx, "(%s) Waiting for incoming to stop : %d", node.address, incomingCount)
			waitCount = 0
		}
		waitCount++
	}

	node.outgoing.Close()

	// Wait for processing threads to stop.
	waitCount = 0
	for {
		time.Sleep(100 * time.Millisecond)
		processingCount := atomic.LoadUint32(&node.processingCount)
		if processingCount == 0 {
			break
		}
		if waitCount > 30 { // 3 seconds
			logger.Info(ctx, "(%s) Waiting for processing to stop : %d", node.address,
				processingCount)
			waitCount = 0
		}
		waitCount++
	}

	node.lock.Lock()
	node.active = false
	node.lock.Unlock()
	logger.Debug(ctx, "(%s) Stopped", node.address)
	return nil
}

func (node *UntrustedNode) IsActive() bool {
	node.lock.Lock()
	defer node.lock.Unlock()

	return node.active
}

func (node *UntrustedNode) isStopping() bool {
	node.lock.Lock()
	defer node.lock.Unlock()

	return node.stopping
}

func (node *UntrustedNode) Stop(ctx context.Context) error {
	node.lock.Lock()
	if node.stopping {
		logger.Debug(ctx, "(%s) Already stopping", node.address)
		node.lock.Unlock()
		return nil
	}

	// Setting stopping and closing the connection should stop the monitorIncoming thread.
	logger.Debug(ctx, "(%s) Requesting stop", node.address)
	node.stopping = true
	node.lock.Unlock()
	return nil
}

func (node *UntrustedNode) IsReady() bool {
	return node.untrustedState.IsReady()
}

// Broadcast a tx to the peer
func (node *UntrustedNode) BroadcastTxs(ctx context.Context, txs []*wire.MsgTx) error {
	if node.untrustedState.IsReady() {
		for _, tx := range txs {
			if err := node.outgoing.Add(tx); err != nil {
				return err
			}
		}
	}

	node.pendingLock.Lock()
	node.pendingOutgoing = append(node.pendingOutgoing, txs...)
	node.pendingLock.Unlock()
	return nil
}

// CleanupBlock is called when a block is being processed.
// It is responsible for any cleanup as a result of a block.
func (node *UntrustedNode) CleanupBlock(ctx context.Context, txids []*bitcoin.Hash32) error {
	node.txTracker.RemoveList(ctx, txids)
	return nil
}

func (node *UntrustedNode) connect() error {
	conn, err := net.DialTimeout("tcp", node.address, 15*time.Second)
	if err != nil {
		return err
	}

	node.connection = conn
	node.untrustedState.MarkConnected()
	return nil
}

// monitorIncoming monitors incoming messages.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *UntrustedNode) monitorIncoming(ctx context.Context) {
	for !node.isStopping() {
		if err := node.check(ctx); err != nil {
			logger.Debug(ctx, "(%s) Check failed : %s", node.address, err.Error())
			node.Stop(ctx)
			break
		}

		node.lock.Lock()
		connection := node.connection
		node.lock.Unlock()

		if connection == nil {
			break
		}

		// read new messages, blocking
		_, msg, _, err := wire.ReadMessageN(connection, wire.ProtocolVersion,
			wire.BitcoinNet(node.config.Net))
		if err != nil {
			wireError, ok := errors.Cause(err).(*wire.MessageError)
			if ok {
				if wireError.Type == wire.MessageErrorUnknownCommand {
					logger.Debug(ctx, "(%s) %s", node.address, wireError)
					continue
				} else {
					logger.Debug(ctx, "(%s) %s", node.address, wireError)
					node.Stop(ctx)
					break
				}

			} else {
				logger.Debug(ctx, "(%s) Failed to read message : %s", node.address, err.Error())
				node.Stop(ctx)
				break
			}
		}

		if err := node.handleMessage(ctx, msg); err != nil {
			node.peers.UpdateScore(ctx, node.address, -1)
			logger.Debug(ctx, "(%s) Failed to handle [%s] message : %s", node.address,
				msg.Command(), err.Error())
			node.Stop(ctx)
			break
		}
		if msg.Command() == "reject" {
			reject, ok := msg.(*wire.MsgReject)
			if ok {
				logger.Debug(ctx, "(%s) Reject message : %s - %s", node.address, reject.Reason,
					reject.Hash.String())
			}
		}
	}
}

// Check state
func (node *UntrustedNode) check(ctx context.Context) error {
	if !node.untrustedState.VersionReceived() {
		return nil // Still performing handshake
	}

	if !node.untrustedState.HandshakeComplete() {
		// Send header request to verify chain
		headerRequest, err := buildHeaderRequest(ctx, node.untrustedState.ProtocolVersion(),
			node.blocks, handlers.UntrustedHeaderDelta, 10)
		if err != nil {
			return errors.Wrap(err, "build header request")
		}
		if node.outgoing.Add(headerRequest) == nil {
			node.untrustedState.MarkHeadersRequested()
			node.untrustedState.SetHandshakeComplete()
		}
	}

	// Check sync
	if !node.untrustedState.IsReady() {
		return nil
	}

	if !node.readyAnnounced {
		logger.Debug(ctx, "(%s) Ready", node.address)
		node.readyAnnounced = true
	}

	if !node.untrustedState.ScoreUpdated() {
		node.peers.UpdateScore(ctx, node.address, 5)
		node.untrustedState.SetScoreUpdated()
	}

	if !node.untrustedState.AddressesRequested() {
		addresses := wire.NewMsgGetAddr()
		if node.outgoing.Add(addresses) == nil {
			node.untrustedState.SetAddressesRequested()
		}
	}

	if node.scanning {
		logger.Info(ctx, "(%s) Found peer", node.address)
		node.Stop(ctx)
		return nil
	}

	node.pendingLock.Lock()
	if len(node.pendingOutgoing) > 0 {
		for _, tx := range node.pendingOutgoing {
			if node.outgoing.Add(tx) != nil {
				break
			}
		}
	}
	node.pendingOutgoing = nil
	node.pendingLock.Unlock()

	if !node.untrustedState.MemPoolRequested() {
		// Send mempool request
		// This tells the peer to send inventory of all tx in their mempool.
		mempool := wire.NewMsgMemPool()
		if node.outgoing.Add(mempool) == nil {
			node.untrustedState.SetMemPoolRequested()
		}
	}

	if err := node.txTracker.Check(ctx, node.memPool, node); err != nil {
		return errors.Wrap(err, "tx tracker check")
	}

	return nil
}

// TransmitMessage interface
func (node *UntrustedNode) TransmitMessage(msg wire.Message) bool {
	if node.isStopping() {
		return false
	}

	if err := node.outgoing.Add(msg); err != nil {
		return false
	}

	return true
}

// Monitor for request timeouts
func (node *UntrustedNode) monitorRequestTimeouts(ctx context.Context) {
	for !node.isStopping() {
		node.sleepUntilStop(10) // Only check every 10 seconds
		if node.isStopping() {
			break
		}

		if err := node.untrustedState.CheckTimeouts(); err != nil {
			logger.Debug(ctx, "(%s) Timed out : %s", node.address, err.Error())
			node.peers.UpdateScore(ctx, node.address, -1)
			node.Stop(ctx)
			break
		}
	}
}

// sendOutgoing waits for and sends outgoing messages
//
// This is a blocking function that will run forever, so it should be run in a goroutine.
func (node *UntrustedNode) sendOutgoing(ctx context.Context) error {

	for msg := range node.outgoing.Channel {
		node.lock.Lock()
		connection := node.connection
		node.lock.Unlock()

		if connection == nil {
			logger.Debug(ctx, "(%s) Dropping %s message", node.address, msg.Command())
			continue // Keep clearing the channel
		}

		tx, ok := msg.(*wire.MsgTx)
		if ok {
			logger.Verbose(ctx, "(%s) Sending Tx : %s", node.address, tx.TxHash().String())
		}

		if err := sendAsync(ctx, connection, msg, wire.BitcoinNet(node.config.Net)); err != nil {
			logger.Warn(ctx, "(%s) Failed sending command %s : %s", node.address, msg.Command(), err)
			// don't break out of the loop because we need to continue emptying the channel.
			// otherwise any processing adding to the channel can get locked on a write.
			node.Stop(ctx)
		}
	}

	return nil
}

// handleMessage Processes an incoming message
func (node *UntrustedNode) handleMessage(ctx context.Context, msg wire.Message) error {
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
		return errors.Wrap(err, "handle message")
	}

	// Queue messages to be sent in response
	for _, response := range responses {
		if err := node.outgoing.Add(response); err != nil {
			return errors.Wrap(err, "add outgoing")
		}
	}

	return nil
}

func (node *UntrustedNode) sleepUntilStop(seconds int) {
	for i := 0; i < seconds; i++ {
		if node.isStopping() {
			break
		}
		time.Sleep(time.Second)
	}
}
