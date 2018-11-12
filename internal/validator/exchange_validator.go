package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type exchangeValidator struct {
	Fee config.Fee
}

func newExchangeValidator(fee config.Fee) exchangeValidator {
	return exchangeValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h exchangeValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.Exchange)

	// Find the asset
	assetKey := string(m.Party1AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		return protocol.RejectionCodeAssetNotFound
	}

	if asset.Holdings == nil {
		return protocol.RejectionCodeAssetNotFound
	}

	// Party 1: Reject if no holding
	//
	party1Address := itx.Outputs[1]
	party1Key := party1Address.Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Key]
	if !ok {
		return protocol.RejectionCodeInsufficientAssets
	}

	// Check the token balance
	//
	if party1Holding.Balance < m.Party1TokenQty {
		return protocol.RejectionCodeInsufficientAssets
	}

	// Party 1: Frozen assets
	//
	// An order is in force
	if party1Holding.HoldingStatus != nil && !party1Holding.HoldingStatus.Expired() {
		return protocol.RejectionCodeFrozen
	}

	// Party 2: Skip if no holding
	//
	party2Address := itx.Outputs[2]
	party2Key := party2Address.Address.EncodeAddress()
	party2Holding, ok := asset.Holdings[party2Key]

	// An order is in force
	if ok && party2Holding.HoldingStatus != nil && !party2Holding.HoldingStatus.Expired() {
		return protocol.RejectionCodeFrozen
	}

	return protocol.RejectionCodeOK
}
