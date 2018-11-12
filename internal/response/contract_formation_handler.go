package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type contractFormationHandler struct{}

func newContractFormationHandler() contractFormationHandler {
	return contractFormationHandler{}
}

func (h contractFormationHandler) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
