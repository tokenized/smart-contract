package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type voteHandler struct{}

func newVoteHandler() voteHandler {
	return voteHandler{}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h voteHandler) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
