package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/pkg/inspector"
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

func (h contractOfferValidator) validate(ctx context.Context, itx *inspector.Transaction, vd validatorData) uint8 {
	return protocol.RejectionCodeOK
}
