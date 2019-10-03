package vote

import (
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewVote defines what information may be provided to create a Vote.
type NewVote struct {
	Type          uint32                   `json:"Type,omitempty"`
	VoteSystem         uint32                   `json:"VoteSystem,omitempty"`
	ContractWideVote   bool                     `json:"ContractWideVote,omitempty"`
	AssetType          string                   `json:"AssetType,omitempty"`
	AssetCode          protocol.AssetCode       `json:"AssetCode,omitempty"`
	ProposedAmendments []actions.AmendmentField `json:"ProposedAmendments,omitempty"`

	VoteTxId     protocol.TxId      `json:"VoteTxId,omitempty"`
	ProposalTxId protocol.TxId      `json:"ProposalTxId,omitempty"`
	TokenQty     uint64             `json:"TokenQty,omitempty"`
	Expires      protocol.Timestamp `json:"Expires,omitempty"`
	Timestamp    protocol.Timestamp `json:"Timestamp,omitempty"`
}

// UpdateVote struct { defines what information may be provided to modify an
// existing Vote.
type UpdateVote struct {
	CompletedAt *protocol.Timestamp `json:"CompletedAt,omitempty"`
	AppliedTxId *protocol.TxId      `json:"AppliedTxId,omitempty"`
	OptionTally *[]uint64           `json:"OptionTally,omitempty"`
	Result      *string             `json:"Result,omitempty"`
	NewBallot   *state.Ballot       `json:"NewBallot,omitempty"`
}
