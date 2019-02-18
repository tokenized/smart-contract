package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

type voteHandler struct{}

func newVoteHandler() voteHandler {
	return voteHandler{}
}

func (h voteHandler) process(ctx context.Context, itx *inspector.Transaction, c *contract.Contract) error {

	return nil
}
