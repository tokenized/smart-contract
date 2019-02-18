package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type assetDefinitionValidator struct {
	Fee config.Fee
}

// newAssetDefinitionValidator returns a new assetDefinitionValidator.
func newAssetDefinitionValidator(fee config.Fee) assetDefinitionValidator {
	return assetDefinitionValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h assetDefinitionValidator) validate(ctx context.Context, itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.AssetDefinition)

	// Reasons to reject
	assetID := string(m.AssetID)

	if _, ok := c.Assets[assetID]; ok {
		log.Errorf("asset definition : Asset ID already exists")
		return protocol.RejectionCodeDuplicateAssetID
	}

	// check that the contract can have more assets added.
	if !h.canHaveMoreAssets(c) {
		log.Errorf("asset definition : Number of assets exceeds contract Qty")
		return protocol.RejectionCodeFixedQuantity
	}

	return protocol.RejectionCodeOK
}

// canHaveMoreAssets returns true if an Asset can be added to the Contract,
// false otherwise.
//
// A "dynamic" contract is permitted to have unlimted assets if the
// contract.Qty == 0.
func (h assetDefinitionValidator) canHaveMoreAssets(contract *contract.Contract) bool {
	if contract.Qty == 0 {
		return true
	}

	// number of current assets
	total := uint64(len(contract.Assets))

	// more assets can be added if the current total is less than the limit
	// imposed by the contract.
	return total < contract.Qty
}
