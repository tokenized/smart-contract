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

func (h voteHandler) process(ctx context.Context,
	itx *inspector.Transaction, c *contract.Contract) error {

	return nil
}
