package vote

import (
	"github.com/tokenized/specification/dist/golang/protocol"
)

// ValidateProposal returns true if the Proposal is valid.
func ValidateProposal(msg *protocol.Proposal, now protocol.Timestamp) bool {
	if msg.Specific && len(msg.ProposedAmendments) == 0 {
		return false
	}

	if len(msg.VoteOptions) == 0 {
		return false
	}

	if msg.VoteMax == 0 {
		return false
	}

	if msg.VoteCutOffTimestamp.Nano() < now.Nano() {
		return false
	}

	return true
}
