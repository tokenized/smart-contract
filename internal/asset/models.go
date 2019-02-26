package asset

import "github.com/tokenized/smart-contract/internal/platform/state"

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	IssuerAddress      string `json:"string"`
	ID                 string `json:"id"`
	Type               string `json:"type"`
	VotingSystem       string `json:"voting_system"`
	VoteMultiplier     uint8  `json:"vote_multiplier"`
	Qty                uint64 `json:"qty"`
	AuthorizationFlags []byte `json:"authorization_flags"`
}

// UpdateAsset defines what information may be provided to modify an existing
// Asset. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAsset struct {
	Type               *string                         `json:"type"`
	Revision           *uint16                         `json:"revision"`
	VotingSystem       *string                         `json:"voting_system"`
	VoteMultiplier     *uint8                          `json:"vote_multiplier"`
	Qty                *uint64                         `json:"qty"`
	AuthorizationFlags []byte                          `json:"authorization_flags"`
	NewBalances        map[string]uint64               `json:"new_balances"`
	NewHoldingStatus   map[string]*state.HoldingStatus `json:"new_holding_status"`
}
