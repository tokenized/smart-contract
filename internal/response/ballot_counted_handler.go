package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type ballotCountedHandler struct{}

func newBallotCountedHandler() ballotCountedHandler {
	return ballotCountedHandler{}
}

func (h ballotCountedHandler) process(ctx context.Context,
	itx *inspector.Transaction, c *contract.Contract) error {

	return nil
}
