package validator

import (
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type sendValidator struct {
	Fee config.Fee
}

func newSendValidator(fee config.Fee) sendValidator {
	return sendValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h sendValidator) validate(itx *inspector.Transaction, vd validatorData) uint8 {

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.Send)

	// Get the asset
	assetKey := string(m.AssetID)
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
	party1Addr := party1Address.Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Addr]
	if !ok {
		return protocol.RejectionCodeInsufficientAssets
	}

	// The balance is an unsigned int, so we can't substract without
	// wrapping. Do a comparison instead.
	if m.TokenQty > party1Holding.Balance {
		return protocol.RejectionCodeInsufficientAssets
	}

	// Party 1: Frozen assets
	//
	if party1Holding.HoldingStatus != nil {
		// this holding is marked as Frozen.
		status := party1Holding.HoldingStatus

		if !status.Expired() {
			// this order is in force
			return protocol.RejectionCodeFrozen
		}
	}

	// Not enough outputs / Receiver missing
	//
	if len(itx.Outputs) < 2 {
		return protocol.RejectionCodeReceiverUnspecified
	}

	party2Addr := itx.Outputs[1].Address.EncodeAddress()

	// Cannot transfer to self
	//
	if party1Addr == party2Addr {
		return protocol.RejectionCodeTransferSelf
	}

	return protocol.RejectionCodeOK
}
