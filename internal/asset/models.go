package asset

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	AdministrationAddress bitcoin.RawAddress `json:"AdministrationAddress,omitempty"`

	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	AssetType                   string   `json:"AssetType,omitempty"`
	AssetIndex                  uint64   `json:"AssetIndex,omitempty"`
	AssetPermissions            []byte   `json:"AssetPermissions,omitempty"`
	TransfersPermitted          bool     `json:"TransfersPermitted,omitempty"`
	TradeRestrictions           []string `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted  bool     `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                bool     `json:"VotingRights,omitempty"`
	VoteMultiplier              uint32   `json:"VoteMultiplier,omitempty"`
	AdministrationProposal      bool     `json:"AdministrationProposal,omitempty"`
	HolderProposal              bool     `json:"HolderProposal,omitempty"`
	AssetModificationGovernance uint32   `json:"AssetModificationGovernance,omitempty"`
	TokenQty                    uint64   `json:"TokenQty,omitempty"`
	AssetPayload                []byte   `json:"AssetPayload,omitempty"`
}

// UpdateAsset defines what information may be provided to modify an existing
// Asset. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAsset struct {
	Revision  *uint32             `json:"Revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"Timestamp,omitempty"`

	AssetPermissions            *[]byte             `json:"AssetPermissions,omitempty"`
	TransfersPermitted          *bool               `json:"TransfersPermitted,omitempty"`
	TradeRestrictions           *[]string           `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted  *bool               `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                *bool               `json:"VotingRights,omitempty"`
	VoteMultiplier              *uint32             `json:"VoteMultiplier,omitempty"`
	AdministrationProposal      *bool               `json:"AdministrationProposal,omitempty"`
	HolderProposal              *bool               `json:"HolderProposal,omitempty"`
	AssetModificationGovernance *uint32             `json:"AssetModificationGovernance,omitempty"`
	TokenQty                    *uint64             `json:"TokenQty,omitempty"`
	AssetPayload                *[]byte             `json:"AssetPayload,omitempty"`
	FreezePeriod                *protocol.Timestamp `json:"FreezePeriod,omitempty"`
}
