package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/inspector"
)

type validatorInterface interface {
	validate(context.Context, *inspector.Transaction, validatorData) uint8
}
