package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type assetCreationResponse struct{}

func newAssetCreationResponse() assetCreationResponse {
	return assetCreationResponse{}
}

func (h assetCreationResponse) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
