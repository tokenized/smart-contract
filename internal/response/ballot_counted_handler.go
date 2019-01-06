package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
)

type ballotCountedHandler struct{}

func newBallotCountedHandler() ballotCountedHandler {
	return ballotCountedHandler{}
}

func (h ballotCountedHandler) process(ctx context.Context,
	itx *inspector.Transaction, c *contract.Contract) error {

	return nil
}
