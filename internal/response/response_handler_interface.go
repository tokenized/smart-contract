package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

type responseHandlerInterface interface {
	process(context.Context, *inspector.Transaction, *contract.Contract) error
}
