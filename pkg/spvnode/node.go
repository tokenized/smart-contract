package spvnode

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/spvnode/logger"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"go.uber.org/multierr"
)

const (
	MainNetBch wire.BitcoinNet = 0xe8f3e1e3
	TestNetBch wire.BitcoinNet = 0xf4f3e5f4
	RegTestBch wire.BitcoinNet = 0xfabfb5da

	ListenerTX    = "TX"
	ListenerBlock = "block"

	firstBCHBlock = 478559
)

type Node struct {
	Config       Config
	Handlers     map[string]CommandHandler
	conn         net.Conn
	messages     chan wire.Message
	BlockService *BlockService
	Listeners    map[string]Listener
}

func NewNode(config Config, store storage.Storage) Node {
	stateRepo := NewStateRepository(store)
	blockRepo := NewBlockRepository(store)
	blockService := NewBlockService(blockRepo, stateRepo)

	n := Node{
		Config:       config,
		messages:     make(chan wire.Message),
		BlockService: &blockService,
		Listeners:    map[string]Listener{},
	}

	return n
}

func (n *Node) Start() error {
	ctx := logger.NewContext()
	log := logger.NewLoggerFromContext(ctx).Sugar()

	n.Handlers = newCommandHandlers(n.Config, n.BlockService, n.Listeners)

	state, err := n.BlockService.LoadState(ctx)
	if err != nil {
		return err
	}

	if state.LastSeen.Height > 0 {
		// This is not the first run.
		//
		// On first run we don't broadcast to listeners to avoid publishing
		// the entire blockchain.
		n.BlockService.synced = true
	}

	log.Infof("Loaded initial state : %+v", *state)

	// load any blocks we have into the cache.
	if err := n.BlockService.LoadBlocks(ctx); err != nil {
		return err
	}

	log.Infof("Loaded %v blocks", len(n.BlockService.Blocks))

	if err := n.connect(); err != nil {
		return err
	}

	// we will probably never really exit.
	defer n.close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		// receive messages from the peer in a goroutine
		n.readPeer()
	}()

	go func() {
		defer wg.Done()

		// receive outbound messages on a channel
		n.readChannel()
	}()

	// kick off the connection handshaking process by sending a version
	// message.
	if err := n.handshake(); err != nil {
		return nil
	}

	// block until goroutines finish, which is currently never
	wg.Wait()

	return nil
}

func (n *Node) connect() error {
	n.close()

	conn, err := net.Dial("tcp", n.Config.NodeAddress)
	if err != nil {
		return err
	}

	n.conn = conn

	return nil
}

func (n *Node) close() {
	if n.conn == nil {
		return
	}

	// close the connection, ignoring any errors
	_ = n.conn.Close()

	n.conn = nil
}

// readPeer reads new messages from the Peer.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (n Node) readPeer() {
	for {
		ctx := logger.NewContext()

		// read new messages, blocking
		m, _, err := wire.ReadMessage(n.conn, wire.ProtocolVersion, MainNetBch)
		if err != nil {
			log := logger.NewLoggerFromContext(ctx)
			log.Error(err.Error())

			// wait before reconnecting
			time.Sleep(time.Second * 30)
			continue
		}

		if err := n.handle(ctx, m); err != nil {
			log := logger.NewLoggerFromContext(ctx).Sugar()
			log.Errorf("msg = %+v : %v", m, err.Error())
		}
	}
}

// handle processes an inbound message.
func (n Node) handle(ctx context.Context,
	m wire.Message) error {

	h, ok := n.Handlers[m.Command()]
	if !ok {
		// no handler for this command
		return nil
	}

	out, err := h.Handle(ctx, m)
	if err != nil {
		return err
	}

	if out == nil {
		if _, ok := m.(*wire.MsgHeaders); ok {
			n.BlockService.synced = true
		}

		return nil
	}

	errors := []error{}

	for _, m := range out {
		if err := n.Queue(ctx, m); err != nil {
			log := logger.NewLoggerFromContext(ctx).Sugar()
			log.Error(err)
			errors = append(errors, err)
		}
	}

	return multierr.Combine(errors...)
}

func (n *Node) RegisterListener(name string, listener Listener) {
	n.Listeners[name] = listener
}

// handshake starts the handshake process.
//
// Sending a version message to the peer will fire off is enough as the
// process is event driven, and this node responds appropriately to the
// responses.
//
// The version handshake look like this.
//
// L -> R: Send version message with the local peer's version
// R -> L: Send version message back
// R -> L: Send verack message
// R:      Sets version to the minimum of the 2 versions
// L -> R: Send verack message after receiving version message from R
// L:      Sets version to the minimum of the 2 versions
func (n Node) handshake() error {
	ctx := logger.NewContext()

	// my local. This doesn't matter, we don't accept inboound connections.
	local := wire.NewNetAddressIPPort(net.IPv4(127, 0, 0, 1), 9333, 0)

	// build the address of the remote
	remote := wire.NewNetAddressIPPort(net.IPv4(127, 0, 0, 1), 8333, 0)

	lastSeen := n.BlockService.State.LastSeen
	msg := wire.NewMsgVersion(remote, local, n.nonce(), lastSeen.Height)
	msg.UserAgent = n.buildUserAgent()
	msg.Services = 0x01

	return n.Queue(ctx, msg)
}

// Queue puts the message on a queue for async delivery.
func (n Node) Queue(ctx context.Context, msg wire.Message) error {
	go func() {
		n.messages <- msg
	}()

	return nil
}

// readChannel receives messages from the channel.
//
// This is a blocking function that will run forever, so it should be run
// in a goroutine.
func (n Node) readChannel() {
	for {
		ctx := logger.NewContext()
		log := logger.NewLoggerFromContext(ctx).Sugar()

		// read from the channel
		m, ok := <-n.messages

		if !ok {
			log.Errorf("Failed reading from channel")
			continue
		}

		if err := n.sendAsync(ctx, m); err != nil {
			log.Errorf("Failed to send %v : %v", m.Command(), err)
		}
	}
}

// sendAsync writes a message to a peer.
func (n Node) sendAsync(ctx context.Context, m wire.Message) error {
	var buf bytes.Buffer

	// build the message to send
	_, err := wire.WriteMessageN(&buf, m, wire.ProtocolVersion, MainNetBch)
	if err != nil {
		return err
	}

	b := buf.Bytes()

	// send the message to the remote
	_, err = n.conn.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (n Node) buildUserAgent() string {
	return fmt.Sprintf("%v", n.Config.UserAgent)
}

func (n Node) nonce() uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)

	return binary.LittleEndian.Uint64(buf)
}
