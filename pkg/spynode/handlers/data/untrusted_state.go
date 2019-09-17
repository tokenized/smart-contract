package data

import (
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// State of untrusted node
type UntrustedState struct {
	connectedTime      *time.Time // Time of last connection
	versionReceived    bool       // Version message was received
	protocolVersion    uint32     // Bitcoin protocol version
	handshakeComplete  bool       // Handshake negotiation is complete
	addressesRequested bool       // Peer addresses have been requested
	memPoolRequested   bool       // Mempool has bee requested
	headersRequested   *time.Time // Time that headers were last requested
	verified           bool       // The node has been verified to be on the same chain
	scoreUpdated       bool       // The score has been updated after the node has been verified
	lock               sync.Mutex
}

func NewUntrustedState() *UntrustedState {
	result := UntrustedState{
		connectedTime:      nil,
		versionReceived:    false,
		protocolVersion:    wire.ProtocolVersion,
		handshakeComplete:  false,
		addressesRequested: false,
		memPoolRequested:   false,
		headersRequested:   nil,
		verified:           false,
		scoreUpdated:       false,
	}
	return &result
}

func (state *UntrustedState) ProtocolVersion() uint32 {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.protocolVersion
}

func (state *UntrustedState) IsReady() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.verified
}

func (state *UntrustedState) SetVerified() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.verified = true
}

func (state *UntrustedState) ScoreUpdated() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.scoreUpdated
}

func (state *UntrustedState) SetScoreUpdated() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.scoreUpdated = true
}

func (state *UntrustedState) MarkConnected() {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()
	state.connectedTime = &now
}

func (state *UntrustedState) AddressesRequested() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.addressesRequested
}

func (state *UntrustedState) SetAddressesRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.addressesRequested = true
}

func (state *UntrustedState) VersionReceived() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.versionReceived
}

func (state *UntrustedState) SetVersionReceived() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.versionReceived = true
}

func (state *UntrustedState) MemPoolRequested() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.memPoolRequested
}

func (state *UntrustedState) SetMemPoolRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.memPoolRequested = true
}

func (state *UntrustedState) HandshakeComplete() bool {
	state.lock.Lock()
	defer state.lock.Unlock()

	return state.handshakeComplete
}

func (state *UntrustedState) SetHandshakeComplete() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.handshakeComplete = true
}

func (state *UntrustedState) HeadersRequested() *time.Time {
	state.lock.Lock()
	defer state.lock.Unlock()

	if state.headersRequested == nil {
		return nil
	}
	result := *state.headersRequested
	return &result
}

func (state *UntrustedState) MarkHeadersRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	now := time.Now()
	state.headersRequested = &now
}

func (state *UntrustedState) ClearHeadersRequested() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.headersRequested = nil
}

func (state *UntrustedState) MarkVerified() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.verified = true
}
