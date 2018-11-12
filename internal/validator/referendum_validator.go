package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type referendumValidator struct{}

func newReferendumValidator() referendumValidator {
	return referendumValidator{}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h referendumValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	// Contract and Message
	c := vd.contract
	// m := vd.m.(*protocol.Referendum)

	hash := itx.MsgTx.TxHash()

	// add the vote to the votes on the proposal
	key := hash.String()
	if _, ok := c.Votes[key]; ok {
		// a vote already exists, cannot clobber the old one
		return protocol.RejectionCodeVoteExists
	}

	// TODO reject if not from an Issuer (should already be handled by the
	// Service. Service may be the right place to ensure correct actor type?
	// ...

	// FIXME fill in auth flags, permit User to act.
	return protocol.RejectionCodeOK
}
