package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

type reconciliationHandler struct{}

func newReconciliationHandler() reconciliationHandler {
	return reconciliationHandler{}
}

func (h reconciliationHandler) process(ctx context.Context, itx *inspector.Transaction, c *contract.Contract) error {

	return nil
}
