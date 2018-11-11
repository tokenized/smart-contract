package validator

import (
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type contractOfferValidator struct {
	Fee config.Fee
}

func newContractOfferValidator(fee config.Fee) contractOfferValidator {
	return contractOfferValidator{
		Fee: fee,
	}
}

func (h contractOfferValidator) validate(itx *inspector.Transaction, vd validatorData) uint8 {
	return protocol.RejectionCodeOK
}
