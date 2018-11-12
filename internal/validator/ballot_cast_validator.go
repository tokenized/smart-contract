package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type ballotCastValidator struct{}

func newBallotCastValidator() ballotCastValidator {
	return ballotCastValidator{}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h ballotCastValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.BallotCast)

	// Is this a valid and active vote?
	key := string(m.VoteTxnID)
	vote, ok := c.Votes[key]
	if !ok {
		return protocol.RejectionCodeVoteNotFound
	}

	// Can this person vote
	sender := itx.InputAddrs[0]
	ballot := contract.NewBallotFromBallotCast(sender, m)

	if code := c.CanVote(vote, ballot); code != protocol.RejectionCodeOK {
		return code
	}

	// TODO reject if the asset was received after the vote
	// If settlement date > vote date = reject

	return protocol.RejectionCodeOK
}
