package vote

import (
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
)

// ValidateProposal returns true if the Proposal is valid.
func ValidateProposal(msg *protocol.Proposal) bool {
	if msg.Specific && len(msg.ProposedAmendments) == 0 {
		return false
	}

	if len(msg.VoteOptions) == 0 {
		return false
	}

	if msg.VoteMax == 0 {
		return false
	}

	if msg.VoteCutOffTimestamp.Nano() < uint64(time.Now().UnixNano()) {
		return false
	}

	return true
}
