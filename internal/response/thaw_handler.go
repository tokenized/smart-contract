package response

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type thawHandler struct{}

func newThawHandler() thawHandler {
	return thawHandler{}
}

func (h thawHandler) process(ctx context.Context,
	itx *inspector.Transaction, c *contract.Contract) error {

	msg := itx.MsgProto.(*protocol.Thaw)
	assetKey := string(msg.AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		return fmt.Errorf("freeze : Asset ID not found : contract=%s assetID=%s", c.ID, msg.AssetID)
	}

	// Party 1 (Target)
	party1AddrStr := itx.Outputs[0].Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1AddrStr]
	if !ok {
		party1Holding = contract.NewHolding(party1AddrStr, 0)
	}

	party1Holding.HoldingStatus = nil

	// Put the holding back on the asset
	asset.Holdings[party1AddrStr] = party1Holding

	// Put the asset back  on the contract
	c.Assets[assetKey] = asset

	return nil
}
