package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type confiscationResponse struct{}

func newConfiscationResponse() confiscationResponse {
	return confiscationResponse{}
}

func (h confiscationResponse) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
