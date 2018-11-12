package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type confiscationHandler struct{}

func newConfiscationHandler() confiscationHandler {
	return confiscationHandler{}
}

func (h confiscationHandler) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
