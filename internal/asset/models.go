package asset

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/platform/state"
)

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	IssuerAddress string `json:"issuer_address,omitempty"`

	AssetType                   string  `json:"asset_type,omitempty"`
	AssetAuthFlags              [8]byte `json:"asset_auth_flags,omitempty"`
	TransfersPermitted          bool    `json:"transfers_permitted,omitempty"`
	TradeRestrictions           string  `json:"trade_restrictions,omitempty"`
	EnforcementOrdersPermitted  bool    `json:"enforcement_orders_permitted,omitempty"`
	VoteMultiplier              uint8   `json:"vote_multiplier,omitempty"`
	ReferendumProposal          bool    `json:"referendum_proposal,omitempty"`
	InitiativeProposal          bool    `json:"initiative_proposal,omitempty"`
	AssetModificationGovernance bool    `json:"asset_modification_governance,omitempty"`
	TokenQty                    uint64  `json:"token_qty,omitempty"`
	ContractFeeCurrency         string  `json:"contract_fee_currency,omitempty"`
	ContractFeeVar              float32 `json:"contract_fee_var,omitempty"`
	ContractFeeFixed            float32 `json:"contract_fee_fixed,omitempty"`
	AssetPayload                []byte  `json:"asset_payload,omitempty"`
}

// UpdateAsset defines what information may be provided to modify an existing
// Asset. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAsset struct {
	Revision *uint64 `json:"revision,omitempty"`

	AssetType                   *string  `json:"asset_type,omitempty"`
	AssetAuthFlags              *[8]byte `json:"asset_auth_flags,omitempty"`
	TransfersPermitted          *bool    `json:"transfers_permitted,omitempty"`
	TradeRestrictions           *string  `json:"trade_restrictions,omitempty"`
	EnforcementOrdersPermitted  *bool    `json:"enforcement_orders_permitted,omitempty"`
	VoteMultiplier              *uint8   `json:"vote_multiplier,omitempty"`
	ReferendumProposal          *bool    `json:"referendum_proposal,omitempty"`
	InitiativeProposal          *bool    `json:"initiative_proposal,omitempty"`
	AssetModificationGovernance *bool    `json:"asset_modification_governance,omitempty"`
	TokenQty                    *uint64  `json:"token_qty,omitempty"`
	ContractFeeCurrency         *string  `json:"contract_fee_currency,omitempty"`
	ContractFeeVar              *float32 `json:"contract_fee_var,omitempty"`
	ContractFeeFixed            *float32 `json:"contract_fee_fixed,omitempty"`
	AssetPayload                *[]byte  `json:"asset_payload,omitempty"`

	NewBalances          map[string]uint64               `json:"new_balances,omitempty"`
	NewHoldingStatuses   map[string]*state.HoldingStatus `json:"new_holding_statuses,omitempty"`
	ClearHoldingStatuses map[string]*chainhash.Hash      `json:"clear_holding_statuses,omitempty"`
}
