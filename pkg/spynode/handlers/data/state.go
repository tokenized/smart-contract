package data

import (
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// State of node
type State struct {
	ConnectedTime      *time.Time       // Time of last connection
	VersionReceived    bool             // Version message was received
	ProtocolVersion    uint32           // Bitcoin protocol version
	HandshakeComplete  bool             // Handshake negotiation is complete
	SentSendHeaders    bool             // The sendheaders message has been sent
	IsInSync           bool             // We have all the blocks our peer does and we are just monitoring new data
	NotifiedSync       bool             // Sync message has been sent to listeners
	AddressesRequested bool             // Peer addresses have been requested
	MemPoolRequested   bool             // Mempool has bee requested
	HeadersRequested   *time.Time       // Time that headers were last requested
	StartHeight        int              // Height of start block (to start pulling full blocks)
	blocksRequested    []RequestedBlock // Blocks that have been requested
	blocksToRequest    []chainhash.Hash // Blocks that need to be requested
	PendingSync        bool             // The peer has notified us of all blocks. Now we just have to process to catch up.
	restartCount       int              // Number of restarts since last clear
	mutex              sync.Mutex
}

func NewState() *State {
	result := State{
		ConnectedTime:      nil,
		VersionReceived:    false,
		ProtocolVersion:    wire.ProtocolVersion,
		HandshakeComplete:  false,
		SentSendHeaders:    false,
		IsInSync:           false,
		NotifiedSync:       false,
		AddressesRequested: false,
		MemPoolRequested:   false,
		HeadersRequested:   nil,
		StartHeight:        -1,
		blocksRequested:    make([]RequestedBlock, 0, maxRequestedBlocks),
		blocksToRequest:    make([]chainhash.Hash, 0, 2000),
		PendingSync:        false,
		restartCount:       0,
	}
	return &result
}

func (state *State) Reset() {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.ConnectedTime = nil
	state.VersionReceived = false
	state.ProtocolVersion = wire.ProtocolVersion
	state.HandshakeComplete = false
	state.SentSendHeaders = false
	state.IsInSync = false
	state.MemPoolRequested = false
	state.HeadersRequested = nil
	state.blocksRequested = state.blocksRequested[:0]
	state.blocksToRequest = state.blocksToRequest[:0]
	state.PendingSync = false
}

func (state *State) IsReady() bool {
	return state.IsInSync
}
