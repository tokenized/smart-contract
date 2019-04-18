package asset

import (
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	IssuerPKH protocol.PublicKeyHash `json:"issuer_pkh,omitempty"`

	Timestamp protocol.Timestamp `json:"timestamp,omitempty"`

	AssetType                   string    `json:"asset_type,omitempty"`
	AssetAuthFlags              []byte    `json:"asset_auth_flags,omitempty"`
	TransfersPermitted          bool      `json:"transfers_permitted,omitempty"`
	TradeRestrictions           [][3]byte `json:"trade_restrictions,omitempty"`
	EnforcementOrdersPermitted  bool      `json:"enforcement_orders_permitted,omitempty"`
	VotingRights                bool      `json:"voting_rights,omitempty"`
	VoteMultiplier              uint8     `json:"vote_multiplier,omitempty"`
	IssuerProposal              bool      `json:"issuer_proposal,omitempty"`
	HolderProposal              bool      `json:"holder_proposal,omitempty"`
	AssetModificationGovernance uint8     `json:"asset_modification_governance,omitempty"`
	TokenQty                    uint64    `json:"token_qty,omitempty"`
	ContractFeeCurrency         string    `json:"contract_fee_currency,omitempty"`
	ContractFeeVar              float32   `json:"contract_fee_var,omitempty"`
	ContractFeeFixed            float32   `json:"contract_fee_fixed,omitempty"`
	AssetPayload                []byte    `json:"asset_payload,omitempty"`
}

// UpdateAsset defines what information may be provided to modify an existing
// Asset. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAsset struct {
	Revision  *uint32             `json:"revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"timestamp,omitempty"`

	AssetType                   *string             `json:"asset_type,omitempty"`
	AssetAuthFlags              *[]byte             `json:"asset_auth_flags,omitempty"`
	TransfersPermitted          *bool               `json:"transfers_permitted,omitempty"`
	TradeRestrictions           *[][3]byte          `json:"trade_restrictions,omitempty"`
	EnforcementOrdersPermitted  *bool               `json:"enforcement_orders_permitted,omitempty"`
	VotingRights                *bool               `json:"voting_rights,omitempty"`
	VoteMultiplier              *uint8              `json:"vote_multiplier,omitempty"`
	IssuerProposal              *bool               `json:"issuer_proposal,omitempty"`
	HolderProposal              *bool               `json:"holder_proposal,omitempty"`
	AssetModificationGovernance *uint8              `json:"asset_modification_governance,omitempty"`
	TokenQty                    *uint64             `json:"token_qty,omitempty"`
	AssetPayload                *[]byte             `json:"asset_payload,omitempty"`
	FreezePeriod                *protocol.Timestamp `json:"freeze_period,omitempty"`

	NewBalances          map[protocol.PublicKeyHash]uint64              `json:"new_balances,omitempty"`
	NewHoldingStatuses   map[protocol.PublicKeyHash]state.HoldingStatus `json:"new_holding_statuses,omitempty"`
	ClearHoldingStatuses map[protocol.PublicKeyHash]protocol.TxId       `json:"clear_holding_statuses,omitempty"`
}
