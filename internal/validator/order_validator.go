package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/logger"
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
func (h orderValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.Order)

	// Find the asset
	assetKey := string(m.AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		log.Errorf("order : Asset ID not found : contract=%s assetID=%s", c.ID, m.AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	if asset.Holdings == nil {
		log.Errorf("order : No holdings for Asset ID : contract=%s assetID=%s", c.ID, m.AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	// Party 1 (Target): Reject if no holding
	party1Addr := string(m.TargetAddress)
	_, ok = asset.Holdings[party1Addr]
	if !ok {
		log.Errorf("exchange : Party holding not found contract=%s assetID=%s party1=%s", c.ID, m.AssetID, party1Addr)
		return protocol.RejectionCodeInsufficientAssets
	}

	return protocol.RejectionCodeOK
}
