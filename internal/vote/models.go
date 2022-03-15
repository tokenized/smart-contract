package vote

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewVote defines what information may be provided to create a Vote.
type NewVote struct {
	Type               uint32                   `json:"Type,omitempty"`
	VoteSystem         uint32                   `json:"VoteSystem,omitempty"`
	ContractWideVote   bool                     `json:"ContractWideVote,omitempty"`
	InstrumentType     string                   `json:"InstrumentType,omitempty"`
	InstrumentCode     *bitcoin.Hash20          `json:"InstrumentCode,omitempty"`
	ProposedAmendments []actions.AmendmentField `json:"ProposedAmendments,omitempty"`

	VoteTxId     bitcoin.Hash32     `json:"VoteTxId,omitempty"`
	ProposalTxId bitcoin.Hash32     `json:"ProposalTxId,omitempty"`
	TokenQty     uint64             `json:"TokenQty,omitempty"`
	Expires      protocol.Timestamp `json:"Expires,omitempty"`
	Timestamp    protocol.Timestamp `json:"Timestamp,omitempty"`

	Ballots map[bitcoin.Hash20]state.Ballot `json:"-"`
}

// UpdateVote defines what information may be provided to modify an existing Vote.
type UpdateVote struct {
	CompletedAt *protocol.Timestamp `json:"CompletedAt,omitempty"`
	AppliedTxId *bitcoin.Hash32     `json:"AppliedTxId,omitempty"`
	OptionTally *[]uint64           `json:"OptionTally,omitempty"`
	Result      *string             `json:"Result,omitempty"`
	NewBallot   *state.Ballot       `json:"NewBallot,omitempty"`
}
