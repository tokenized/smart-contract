package validator

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/inspector"
)

type validatorInterface interface {
	validate(context.Context, *inspector.Transaction, validatorData) uint8
}
