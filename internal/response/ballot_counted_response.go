package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type ballotCountedResponse struct{}

func newBallotCountedResponse() ballotCountedResponse {
	return ballotCountedResponse{}
}

func (h ballotCountedResponse) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
