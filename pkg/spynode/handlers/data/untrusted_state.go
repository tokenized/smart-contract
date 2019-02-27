package data

import (
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// State of untrusted node
type UntrustedState struct {
	ConnectedTime      *time.Time // Time of last connection
	VersionReceived    bool       // Version message was received
	ProtocolVersion    uint32     // Bitcoin protocol version
	HandshakeComplete  bool       // Handshake negotiation is complete
	AddressesRequested bool       // Peer addresses have been requested
	MemPoolRequested   bool       // Mempool has bee requested
	HeadersRequested   *time.Time // Time that headers were last requested
	Verified           bool       // The node has been verified to be on the same chain
	ScoreUpdated       bool       // The score has been updated after the node has been verified
	mutex              sync.Mutex
}

func NewUntrustedState() *UntrustedState {
	result := UntrustedState{
		ConnectedTime:      nil,
		VersionReceived:    false,
		ProtocolVersion:    wire.ProtocolVersion,
		HandshakeComplete:  false,
		AddressesRequested: false,
		MemPoolRequested:   false,
		HeadersRequested:   nil,
		Verified:           false,
		ScoreUpdated:       false,
	}
	return &result
}

func (state *UntrustedState) IsReady() bool {
	return state.Verified
}
