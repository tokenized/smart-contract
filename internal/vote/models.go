package vote

import (
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// NewVote defines what information may be provided to create a Vote.
type NewVote struct {
	VoteTxId     protocol.TxId      `json:"vote_tx_id_hash,omit_empty"`
	ProposalTxId protocol.TxId      `json:"proposal_tx_id_hash,omit_empty"`
	TokenQty     uint64             `json:"token_qty,omit_empty"`
	Expires      protocol.Timestamp `json:"expires,omit_empty"`
	Timestamp    protocol.Timestamp `json:"timestamp,omit_empty"`
}

// UpdateVote struct { defines what information may be provided to modify an
// existing Vote.
type UpdateVote struct {
	CompletedAt *protocol.Timestamp `json:"completed_at,omit_empty"`
	OptionTally *[]uint64           `json:"option_tally,omit_empty"`
	Result      *string             `json:"result,omit_empty"`
	NewBallot   *state.Ballot       `json:"new_ballot,omit_empty"`
}
