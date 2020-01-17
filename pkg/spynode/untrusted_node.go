package spynode

import (
	"context"
	"fmt"
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

type UntrustedNode struct {
	address         string
	config          data.Config
	state           *data.UntrustedState
	peers           *handlerstorage.PeerRepository
	blocks          *handlerstorage.BlockRepository
	txs             *handlerstorage.TxRepository
	txTracker       *data.TxTracker
	memPool         *data.MemPool
	txChannel       *handlers.TxChannel
	handlers        map[string]handlers.CommandHandler
	connection      net.Conn
	sendLock        sync.Mutex // Lock for sending on connection
	outgoing        messageChannel
	listeners       []handlers.Listener
	txFilters       []handlers.TxFilter
	stopping        bool
	active          bool // Set to false when connection is closed
	scanning        bool
	readyAnnounced  bool
	pendingLock     sync.Mutex
	pendingOutgoing []*wire.MsgTx
	lock            sync.Mutex
}

func NewUntrustedNode(address string, config data.Config, store storage.Storage, peers *handlerstorage.PeerRepository,
	blocks *handlerstorage.BlockRepository, txs *handlerstorage.TxRepository, memPool *data.MemPool,
	txChannel *handlers.TxChannel, listeners []handlers.Listener, txFilters []handlers.TxFilter, scanning bool) *UntrustedNode {

	result := UntrustedNode{
		address:   address,
		config:    config,
		state:     data.NewUntrustedState(),
		peers:     peers,
		blocks:    blocks,
		txs:       txs,
		txTracker: data.NewTxTracker(),
		memPool:   memPool,
		outgoing:  messageChannel{},
		listeners: listeners,
		txFilters: txFilters,
		txChannel: txChannel,
		stopping:  false,
		active:    false,
		scanning:  scanning,
	}
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

	node.handlers = handlers.NewUntrustedCommandHandlers(ctx, node.state, node.peers, node.blocks, node.txs,
		node.txTracker, node.memPool, node.txChannel, node.listeners, node.txFilters, node.address)

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

	defer func() {
		node.lock.Lock()
		node.active = false
		logger.Debug(ctx, "(%s) Stopped", node.address)
		node.lock.Unlock()
	}()

	// Queue version message to start handshake
	version := buildVersionMsg(node.config.UserAgent, int32(node.blocks.LastHeight()))
	node.outgoing.Add(version)

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		node.monitorIncoming(ctx)
		logger.Debug(ctx, "(%s) Untrusted monitor incoming finished", node.address)
	}()

	go func() {
		defer wg.Done()
		node.monitorRequestTimeouts(ctx)
		logger.Debug(ctx, "(%s) Untrusted monitor request timeouts finished", node.address)
	}()

	go func() {
		defer wg.Done()
		node.sendOutgoing(ctx)
		logger.Debug(ctx, "(%s) Untrusted send outgoing finished", node.address)
	}()

	// Block until goroutines finish as a result of Stop()
	wg.Wait()
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
	defer node.lock.Unlock()

	if node.stopping {
		return nil
	}

	logger.Debug(ctx, "(%s) Stopping", node.address)
	node.stopping = true
	node.outgoing.Close()
	node.sendLock.Lock()
	if node.connection != nil {
		if err := node.connection.Close(); err != nil {
			logger.Warn(ctx, "(%s) Failed to close : %s", node.address, err)
		}
		node.connection = nil
	}
	node.sendLock.Unlock()
	return nil
}

func (node *UntrustedNode) IsReady() bool {
	return node.state.IsReady()
}

// Broadcast a tx to the peer
func (node *UntrustedNode) BroadcastTxs(ctx context.Context, txs []*wire.MsgTx) error {
	if node.state.IsReady() {
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

// ProcessBlock is called when a block is being processed.
// It is responsible for any cleanup as a result of a block.
func (node *UntrustedNode) ProcessBlock(ctx context.Context, txids []*bitcoin.Hash32) error {
	node.txTracker.RemoveList(ctx, txids)
	return nil
}

func (node *UntrustedNode) connect() error {
	conn, err := net.DialTimeout("tcp", node.address, 15*time.Second)
	if err != nil {
		return err
	}

	node.connection = conn
	node.state.MarkConnected()
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

		if node.isStopping() {
			break
		}

		// read new messages, blocking
		msg, _, err := wire.ReadMessage(node.connection, wire.ProtocolVersion, wire.BitcoinNet(node.config.Net))
		if err != nil {
			wireError, ok := err.(*wire.MessageError)
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
			logger.Debug(ctx, "(%s) Failed to handle [%s] message : %s", node.address, msg.Command(), err.Error())
			node.Stop(ctx)
			break
		}
		if msg.Command() == "reject" {
			reject, ok := msg.(*wire.MsgReject)
			if ok {
				logger.Warn(ctx, "(%s) Reject message from : %s - %s", node.address, reject.Reason, reject.Hash.String())
			}
		}
	}
}

// Check state
func (node *UntrustedNode) check(ctx context.Context) error {
	if !node.state.VersionReceived() {
		return nil // Still performing handshake
	}

	if !node.state.HandshakeComplete() {
		// Send header request to verify chain
		headerRequest, err := buildHeaderRequest(ctx, node.state.ProtocolVersion(), node.blocks, handlers.UntrustedHeaderDelta, 10)
		if err != nil {
			return err
		}
		if node.outgoing.Add(headerRequest) == nil {
			node.state.MarkHeadersRequested()
			node.state.SetHandshakeComplete()
		}
	}

	// Check sync
	if !node.state.IsReady() {
		return nil
	}

	if !node.readyAnnounced {
		logger.Debug(ctx, "(%s) Ready", node.address)
		node.readyAnnounced = true
	}

	if !node.state.ScoreUpdated() {
		node.peers.UpdateScore(ctx, node.address, 5)
		node.state.SetScoreUpdated()
	}

	if !node.state.AddressesRequested() {
		addresses := wire.NewMsgGetAddr()
		if node.outgoing.Add(addresses) == nil {
			node.state.SetAddressesRequested()
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
			node.outgoing.Add(tx)
		}
	}
	node.pendingOutgoing = nil
	node.pendingLock.Unlock()

	if !node.state.MemPoolRequested() {
		// Send mempool request
		// This tells the peer to send inventory of all tx in their mempool.
		mempool := wire.NewMsgMemPool()
		if node.outgoing.Add(mempool) == nil {
			node.state.SetMemPoolRequested()
		}
	}

	responses, err := node.txTracker.Check(ctx, node.memPool)
	if err != nil {
		return err
	}
	// Queue messages to be sent in response
	for _, response := range responses {
		node.outgoing.Add(response)
	}

	return nil
}

// Monitor for request timeouts
func (node *UntrustedNode) monitorRequestTimeouts(ctx context.Context) {
	for !node.isStopping() {
		node.sleepUntilStop(10) // Only check every 10 seconds
		if node.isStopping() {
			break
		}

		if err := node.state.CheckTimeouts(); err != nil {
			logger.Debug(ctx, "(%s) Timed out : %s", node.address, err.Error())
			node.peers.UpdateScore(ctx, node.address, -1)
			node.Stop(ctx)
			break
		}
	}
}

// sendOutgoing waits for and sends outgoing messages
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (node *UntrustedNode) sendOutgoing(ctx context.Context) error {
	for msg := range node.outgoing.Channel {
		node.sendLock.Lock()
		if node.connection == nil {
			node.sendLock.Unlock()
			break
		}

		tx, ok := msg.(*wire.MsgTx)
		if ok {
			logger.Verbose(ctx, "(%s) Sending Tx : %s", node.address, tx.TxHash().String())
		}

		if err := sendAsync(ctx, node.connection, msg, wire.BitcoinNet(node.config.Net)); err != nil {
			node.sendLock.Unlock()
			return errors.Wrap(err, fmt.Sprintf("Failed to send %s", msg.Command()))
		}
		node.sendLock.Unlock()
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
		logger.Warn(ctx, "(%s) Failed to handle [%s] message : %s", node.address, msg.Command(), err)
		return nil
	}

	// Queue messages to be sent in response
	for _, response := range responses {
		node.outgoing.Add(response)
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

type messageChannel struct {
	Channel chan wire.Message
	lock    sync.Mutex
	open    bool
}

func (c *messageChannel) Add(msg wire.Message) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- msg
	return nil
}

func (c *messageChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan wire.Message, count)
	c.open = true
	return nil
}

func (c *messageChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
