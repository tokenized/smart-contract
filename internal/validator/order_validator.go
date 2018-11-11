package validator

import (
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type orderValidator struct {
	Fee config.Fee
}

func newOrderValidator(fee config.Fee) orderValidator {
	return orderValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h orderValidator) validate(itx *inspector.Transaction, vd validatorData) uint8 {
	return protocol.RejectionCodeOK
}
