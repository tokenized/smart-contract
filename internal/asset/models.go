package asset

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/platform/state"
)

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	IssuerAddress string

	AssetType                   string
	AssetAuthFlags              []byte
	TransfersPermitted          bool
	TradeRestrictions           []byte
	EnforcementOrdersPermitted  bool
	VoteMultiplier              uint8
	ReferendumProposal          bool
	InitiativeProposal          bool
	AssetModificationGovernance bool
	TokenQty                    uint64
	ContractFeeCurrency         []byte
	ContractFeeVar              float32
	ContractFeeFixed            float32
	AssetPayload                []byte
}

// UpdateAsset defines what information may be provided to modify an existing
// Asset. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAsset struct {
	Revision *uint64

	AssetType                   *string
	AssetAuthFlags              *[]byte
	TransfersPermitted          *bool
	TradeRestrictions           *[]byte
	EnforcementOrdersPermitted  *bool
	VoteMultiplier              *uint8
	ReferendumProposal          *bool
	InitiativeProposal          *bool
	AssetModificationGovernance *bool
	TokenQty                    *uint64
	ContractFeeCurrency         *[]byte
	ContractFeeVar              *float32
	ContractFeeFixed            *float32
	AssetPayload                *[]byte

	NewBalances          map[string]uint64
	NewHoldingStatuses   map[string]*state.HoldingStatus
	ClearHoldingStatuses map[string]*chainhash.Hash
}
