package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type settlementHandler struct{}

func newSettlementHandler() settlementHandler {
	return settlementHandler{}
}

func (h settlementHandler) process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	return nil
}
