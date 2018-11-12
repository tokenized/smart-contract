package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type assetCreationHandler struct{}

func newAssetCreationHandler() assetCreationHandler {
	return assetCreationHandler{}
}

func (h assetCreationHandler) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
