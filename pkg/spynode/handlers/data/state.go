package data

import (
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// State of node
type State struct {
	connectedTime      *time.Time        // Time of last connection
	versionReceived    bool              // Version message was received
	protocolVersion    uint32            // Bitcoin protocol version
	handshakeComplete  bool              // Handshake negotiation is complete
	sentSendHeaders    bool              // The sendheaders message has been sent
	wasInSync          bool              // Flag used to determine if the in sync flag was recently set
	isInSync           bool              // We have all the blocks our peer does and we are just monitoring new data
	notifiedSync       bool              // Sync message has been sent to listeners
	addressesRequested bool              // Peer addresses have been requested
	memPoolRequested   bool              // Mempool has bee requested
	headersRequested   *time.Time        // Time that headers were last requested
	startHeight        int               // Height of start block (to start pulling full blocks)
	blocksRequested    []*requestedBlock // Blocks that have been requested
	blocksToRequest    []chainhash.Hash  // Blocks that need to be requested
	pendingSync        bool              // The peer has notified us of all blocks. Now we just have to process to catch up.
	restartCount       int               // Number of restarts since last clear
	lock               sync.Mutex
}

func NewState() *State {
	result := State{
		connectedTime:      nil,
		versionReceived:    false,
		protocolVersion:    wire.ProtocolVersion,
		handshakeComplete:  false,
		sentSendHeaders:    false,
		wasInSync:          false,
		isInSync:           false,
		notifiedSync:       false,
		addressesRequested: false,
		memPoolRequested:   false,
		headersRequested:   nil,
		startHeight:        -1,
		blocksRequested:    make([]*requestedBlock, 0, maxRequestedBlocks),
		blocksToRequest:    make([]chainhash.Hash, 0, 2000),
		pendingSync:        false,
		restartCount:       0,
	}
	return &result
}

func (state *State) Reset() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.connectedTime = nil
	state.versionReceived = false
	state.protocolVersion = wire.ProtocolVersion
	state.handshakeComplete = false
	state.sentSendHeaders = false
	state.wasInSync = false
	state.isInSync = false
	state.memPoolRequested = false
	state.headersRequested = nil
	state.blocksRequested = state.blocksRequested[:0]
	state.blocksToRequest = state.blocksToRequest[:0]
	state.pendingSync = false
}

func (state *State) ProtocolVersion() uint32 {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.protocolVersion
}

func (state *State) MarkConnected() {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()
	state.connectedTime = &now
}

func (state *State) AddressesRequested() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.addressesRequested
}

func (state *State) SetAddressesRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.addressesRequested = true
}

func (state *State) MemPoolRequested() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.memPoolRequested
}

func (state *State) SetMemPoolRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.memPoolRequested = true
}

func (state *State) SentSendHeaders() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.sentSendHeaders
}

func (state *State) SetSentSendHeaders() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.sentSendHeaders = true
}

func (state *State) IsPendingSync() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.pendingSync
}

func (state *State) SetPendingSync() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.pendingSync = true
}

func (state *State) IsReady() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.isInSync
}

func (state *State) SetInSync() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.isInSync = true
}

func (state *State) ClearInSync() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.wasInSync = false
	state.isInSync = false
}

func (state *State) WasInSync() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.wasInSync
}

func (state *State) SetWasInSync() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.wasInSync = true
}

func (state *State) NotifiedSync() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.notifiedSync
}

func (state *State) SetNotifiedSync() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.notifiedSync = true
}

func (state *State) VersionReceived() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.versionReceived
}

func (state *State) SetVersionReceived() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.versionReceived = true
}

func (state *State) HandshakeComplete() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.handshakeComplete
}

func (state *State) SetHandshakeComplete() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.handshakeComplete = true
}

func (state *State) StartHeight() int {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.startHeight
}

func (state *State) SetStartHeight(startHeight int) {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.startHeight = startHeight
}

func (state *State) LogRestart() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.restartCount++
}

func (state *State) HeadersRequested() *time.Time {
	state.lock.Lock()
	defer state.lock.Unlock()

	if state.headersRequested == nil {
		return nil
	}
	result := *state.headersRequested
	return &result
}

func (state *State) MarkHeadersRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()
	state.headersRequested = &now
}

func (state *State) ClearHeadersRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.headersRequested = nil
}
