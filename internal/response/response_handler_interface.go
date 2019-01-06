package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
)

type responseHandlerInterface interface {
	process(context.Context, *inspector.Transaction, *contract.Contract) error
}
