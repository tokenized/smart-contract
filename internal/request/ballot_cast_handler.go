package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type ballotCastHandler struct{}

func newBallotCastHandler() ballotCastHandler {
	return ballotCastHandler{}
}

func (h ballotCastHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	ballotCast, ok := r.m.(*protocol.BallotCast)
	if !ok {
		return nil, errors.New("Not *protocol.BallotCast")
	}

	// Contract
	c := r.contract

	// Is this a valid and active vote?
	key := string(ballotCast.VoteTxnID)
	vote, ok := c.Votes[key]
	if !ok {
		return nil, errors.New("Vote not found")
	}

	ballot := contract.NewBallotFromBallotCast(r.senders[0], ballotCast)

	// there is no response from the ballot cast, until the vote cut off
	// time has been reached.
	//
	// TODO this means that we need a "cron-like" task to fire off the
	// Result, or that the creator of the Vote should close it by sending
	// a message after cut off time.

	// add the ballot to the vote
	vote.Ballots = append(vote.Ballots, ballot)

	// put the vote back on the Contract
	// FIXME this can clobber the Vote if concurrent access occurred.
	c.Votes[key] = vote

	resp := contractResponse{
		Contract: c,
		Message:  nil,
	}

	return &resp, nil
}
