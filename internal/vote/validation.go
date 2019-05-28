package vote

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// ValidateProposal returns true if the Proposal is valid.
func ValidateProposal(msg *protocol.Proposal, now protocol.Timestamp) error {
	if msg.Specific && len(msg.ProposedAmendments) == 0 {
		return errors.New("Specific with no amendments")
	}

	if len(msg.VoteOptions) == 0 {
		return errors.New("No vote options")
	}

	if msg.VoteMax == 0 {
		return errors.New("Zero vote max")
	}

	if msg.VoteCutOffTimestamp.Nano() < now.Nano() {
		return fmt.Errorf("Vote Expired : %d < %d", msg.VoteCutOffTimestamp.Nano(), now.Nano())
	}

	return nil
}
