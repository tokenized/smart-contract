package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/inspector"
)

type validatorInterface interface {
	validate(context.Context, *inspector.Transaction, validatorData) uint8
}
