package response

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type settlementHandler struct{}

func newSettlementHandler() settlementHandler {
	return settlementHandler{}
}

func (h settlementHandler) process(ctx context.Context, itx *inspector.Transaction, c *contract.Contract) error {

	msg := itx.MsgProto.(*protocol.Settlement)
	assetKey := string(msg.AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		return fmt.Errorf("settlement : Asset ID not found : contract=%s assetID=%s", c.ID, msg.AssetID)
	}

	// Party 1
	party1AddrStr := itx.Outputs[0].Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1AddrStr]
	if !ok {
		party1Holding = contract.NewHolding(party1AddrStr, 0)
	}
	party1Holding.Balance = msg.Party1TokenQty

	// Clear Expired Holding Status
	if party1Holding.HoldingStatus != nil && party1Holding.HoldingStatus.Expired() {
		party1Holding.HoldingStatus = nil
	}

	// Party 2
	party2AddrStr := itx.Outputs[1].Address.EncodeAddress()
	party2Holding, ok := asset.Holdings[party2AddrStr]
	if !ok {
		party2Holding = contract.NewHolding(party2AddrStr, 0)
	}
	party2Holding.Balance = msg.Party2TokenQty

	// Clear Expired Holding Status
	if party2Holding.HoldingStatus != nil && party2Holding.HoldingStatus.Expired() {
		party2Holding.HoldingStatus = nil
	}

	// Put the holdings back on the asset
	asset.Holdings[party1AddrStr] = party1Holding
	asset.Holdings[party2AddrStr] = party2Holding

	// Put the asset back  on the contract
	c.Assets[assetKey] = asset

	return nil
}
