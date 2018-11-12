package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type responseInterface interface {
	process(context.Context, *inspector.Transaction, *contract.Contract) error
}
