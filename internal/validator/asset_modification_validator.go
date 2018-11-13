package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/logger"
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
func (h assetModificationValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.AssetModification)

	// does the asset exist?
	k := string(m.AssetID)

	a, ok := c.Assets[k]
	if !ok {
		log.Errorf("asset modification : Asset ID not found")
		return protocol.RejectionCodeAssetNotFound
	}

	// TODO check asset revision
	if a.Revision != m.AssetRevision {
		log.Errorf("asset modification : Asset Revision does not match current")
		return protocol.RejectionCodeAssetRevision
	}

	return protocol.RejectionCodeOK
}
