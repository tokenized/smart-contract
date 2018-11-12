package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type contractFormationResponse struct{}

func newContractFormationResponse() contractFormationResponse {
	return contractFormationResponse{}
}

func (h contractFormationResponse) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
