package validator

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/pkg/inspector"
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
func (h exchangeValidator) validate(ctx context.Context, itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.Exchange)

	// Find the asset
	assetKey := string(m.Party1AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		log.Errorf("exchange : Asset ID not found : contract=%s assetID=%s", c.ID, m.Party1AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	if asset.Holdings == nil {
		log.Errorf("exchange : No holdings for Asset ID : contract=%s assetID=%s", c.ID, m.Party1AssetID)
		return protocol.RejectionCodeAssetNotFound
	}

	// Party 1: Reject if no holding
	//
	party1Address := itx.Inputs[0].Address
	party1Addr := party1Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Addr]
	if !ok {
		log.Errorf("exchange : Party holding not found contract=%s assetID=%s party1=%s", c.ID, m.Party1AssetID, party1Addr)
		return protocol.RejectionCodeInsufficientAssets
	}

	// Check the token balance
	//
	if party1Holding.Balance < m.Party1TokenQty {
		log.Errorf("exchange : Insufficient assets contract=%s assetID=%s party1=%s", c.ID, m.Party1AssetID, party1Addr)
		return protocol.RejectionCodeInsufficientAssets
	}

	// Party 1: Frozen assets
	//
	// An order is in force
	if party1Holding.HoldingStatus != nil && !party1Holding.HoldingStatus.Expired() {
		log.Errorf("exchange : Party has assets frozen")
		return protocol.RejectionCodeFrozen
	}

	// Not enough outputs / Receiver missing
	//
	if len(itx.Outputs) < 3 {
		log.Errorf("exchange : Not enough outputs")
		return protocol.RejectionCodeReceiverUnspecified
	}

	// Party 2: Skip if no holding
	//
	party2Address := itx.Inputs[1].Address
	party2Addr := party2Address.EncodeAddress()
	party2Holding, ok := asset.Holdings[party2Addr]

	// An order is in force
	if ok && party2Holding.HoldingStatus != nil && !party2Holding.HoldingStatus.Expired() {
		return protocol.RejectionCodeFrozen
	}

	return protocol.RejectionCodeOK
}
