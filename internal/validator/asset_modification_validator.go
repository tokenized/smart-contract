package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type assetModificationValidator struct {
	Fee config.Fee
}

func newAssetModificationValidator(fee config.Fee) assetModificationValidator {
	return assetModificationValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h assetModificationValidator) validate(ctx context.Context, itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.AssetModification)

	// Asset
	assetID := string(m.AssetID)
	a, ok := c.Assets[assetID]
	if !ok {
		log.Errorf("asset modification : Asset ID not found")
		return protocol.RejectionCodeAssetNotFound
	}

	// @TODO: When reducing an assets available supply, the amount must
	// be deducted from the issuers balance, otherwise the action cannot
	// be performed. i.e: Reduction amount must not be in circulation.

	// @TODO: Likewise when the asset quantity is increased, the amount
	// must be added to the issuers holding balance.

	// Revision mismatch
	if a.Revision != m.AssetRevision {
		log.Errorf("asset modification : Asset Revision does not match current")
		return protocol.RejectionCodeAssetRevision
	}

	return protocol.RejectionCodeOK
}
