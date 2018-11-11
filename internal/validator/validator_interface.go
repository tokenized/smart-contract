package validator

import (
	"github.com/tokenized/smart-contract/internal/app/inspector"
)

type validatorInterface interface {
	validate(*inspector.Transaction, validatorData) uint8
}
