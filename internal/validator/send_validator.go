package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/pkg/inspector"
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
func (h sendValidator) validate(ctx context.Context, itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.Send)

	// Get the asset
	assetKey := string(m.AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		log.Errorf("send : Asset ID not found : contract=%s assetID=%s", c.ID, m.AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	if asset.Holdings == nil {
		log.Errorf("send : No holdings for Asset ID : contract=%s assetID=%s", c.ID, m.AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	// Party 1 (Sender): Reject if no holding
	//
	party1Address := itx.Inputs[0].Address
	party1Addr := party1Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Addr]
	if !ok {
		log.Errorf("send : Party holding not found contract=%s assetID=%s party1=%s", c.ID, m.AssetID, party1Addr)
		return protocol.RejectionCodeInsufficientAssets
	}

	// The balance is an unsigned int, so we can't substract without
	// wrapping. Do a comparison instead.
	if m.TokenQty > party1Holding.Balance {
		log.Errorf("send : Insufficient assets contract=%s assetID=%s party1=%s", c.ID, m.AssetID, party1Addr)
		return protocol.RejectionCodeInsufficientAssets
	}

	// Party 1: Frozen assets
	//
	if party1Holding.HoldingStatus != nil {
		// this holding is marked as Frozen.
		status := party1Holding.HoldingStatus

		if !status.Expired() {
			log.Errorf("send : Party has assets frozen")
			// this order is in force
			return protocol.RejectionCodeFrozen
		}
	}

	// Not enough outputs / Receiver missing
	//
	if len(itx.Outputs) < 2 {
		log.Errorf("send : Not enough outputs")
		return protocol.RejectionCodeReceiverUnspecified
	}

	// Party 2 (Receiver)
	party2Addr := itx.Outputs[1].Address.EncodeAddress()

	// Cannot transfer to self
	//
	if party1Addr == party2Addr {
		log.Errorf("send : Cannot transfer to own self contract=%s assetID=%s party1=%s", c.ID, m.AssetID, party1Addr)
		return protocol.RejectionCodeTransferSelf
	}

	return protocol.RejectionCodeOK
}
