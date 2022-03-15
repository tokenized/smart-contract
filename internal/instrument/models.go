package instrument

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewInstrument defines what we require when creating a Instrument record.
type NewInstrument struct {
	AdminAddress bitcoin.RawAddress `json:"AdminAddress,omitempty"`

	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	InstrumentType                   string   `json:"InstrumentType,omitempty"`
	InstrumentIndex                  uint64   `json:"InstrumentIndex,omitempty"`
	InstrumentPermissions            []byte   `json:"InstrumentPermissions,omitempty"`
	TradeRestrictions                []string `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted       bool     `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                     bool     `json:"VotingRights,omitempty"`
	VoteMultiplier                   uint32   `json:"VoteMultiplier,omitempty"`
	AdministrationProposal           bool     `json:"AdministrationProposal,omitempty"`
	HolderProposal                   bool     `json:"HolderProposal,omitempty"`
	InstrumentModificationGovernance uint32   `json:"InstrumentModificationGovernance,omitempty"`
	AuthorizedTokenQty               uint64   `json:"AuthorizedTokenQty,omitempty"`
	InstrumentPayload                []byte   `json:"InstrumentPayload,omitempty"`
}

// UpdateInstrument defines what information may be provided to modify an existing
// Instrument. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateInstrument struct {
	Revision  *uint32             `json:"Revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"Timestamp,omitempty"`

	InstrumentPermissions            *[]byte             `json:"InstrumentPermissions,omitempty"`
	TradeRestrictions                *[]string           `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted       *bool               `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                     *bool               `json:"VotingRights,omitempty"`
	VoteMultiplier                   *uint32             `json:"VoteMultiplier,omitempty"`
	AdministrationProposal           *bool               `json:"AdministrationProposal,omitempty"`
	HolderProposal                   *bool               `json:"HolderProposal,omitempty"`
	InstrumentModificationGovernance *uint32             `json:"InstrumentModificationGovernance,omitempty"`
	AuthorizedTokenQty               *uint64             `json:"AuthorizedTokenQty,omitempty"`
	InstrumentPayload                *[]byte             `json:"InstrumentPayload,omitempty"`
	FreezePeriod                     *protocol.Timestamp `json:"FreezePeriod,omitempty"`
}
