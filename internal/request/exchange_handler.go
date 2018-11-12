package request

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type exchangeHandler struct {
	Fee config.Fee
}

func newExchangeHandler(fee config.Fee) exchangeHandler {
	return exchangeHandler{
		Fee: fee,
	}
}

func (h exchangeHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	exchange, ok := r.m.(*protocol.Exchange)
	if !ok {
		return nil, errors.New("Not *protocol.Exchange")
	}

	// Contract
	c := r.contract

	// find the asset
	assetKey := string(exchange.Party1AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		return nil, fmt.Errorf("exchange : Asset ID not found : contract=%s assetID=%s", c.ID, exchange.Party1AssetID)
	}

	// bounds check for receivers - contract, party1, party2
	if len(r.receivers) < 3 {
		return nil, fmt.Errorf("Missing receivers")
	}

	// does the seller hold the asset?
	party1Address := r.receivers[1]
	party1Key := party1Address.Address.EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Key]
	if !ok {
		return nil, fmt.Errorf("exchange : holding not found contract=%s assetID=%s party1=%s", c.ID, assetKey, party1Key)
	}

	// get or create a holding for party2
	party2Address := r.receivers[2]
	party2Key := party2Address.Address.EncodeAddress()
	party2Holding, ok := asset.Holdings[party2Key]
	if !ok {
		// create an empty holding for party2
		party2Holding = contract.NewHolding(party2Key, 0)
	}

	// check the token balance
	if party1Holding.Balance < exchange.Party1TokenQty {
		return nil, fmt.Errorf("exchange : insufficient assets contract=%s assetID=%s party1=%s", c.ID, assetKey, party1Key)
	}

	if party1Holding.HoldingStatus != nil {
		// this holding is marked as Frozen.
		status := party1Holding.HoldingStatus

		if !status.Expired() {
			// this order is in force
			return nil, fmt.Errorf("exchange : assets are frozen contract=%s assetID=%s party1=%s", c.ID, assetKey, party1Key)
		}

		// clear the expired freeze
		party1Holding.HoldingStatus = nil
	}

	if party2Holding.HoldingStatus != nil {
		// this holding is marked as Frozen.
		status := party2Holding.HoldingStatus

		if !status.Expired() {
			// this order is in force
			return nil, fmt.Errorf("exchange : assets are frozen contract=%s assetID=%s party2=%s", c.ID, assetKey, party2Key)
		}

		// clear the expired freeze
		party2Holding.HoldingStatus = nil
	}

	logger := logger.NewLoggerFromContext(ctx).Sugar()
	logger.Infof("exchange party1=%s party2=%s contract=%s asset_id=%s qty=%v",
		party1Key,
		party2Key,
		c.ID,
		assetKey,
		exchange.Party1TokenQty)

	party1Holding.Balance -= exchange.Party1TokenQty
	party2Holding.Balance += exchange.Party1TokenQty

	party1Balance := party1Holding.Balance
	party2Balance := party2Holding.Balance

	// put the holdings back on the asset
	asset.Holdings[party1Key] = party1Holding
	asset.Holdings[party2Key] = party2Holding

	// put the asset back  on the contract
	c.Assets[assetKey] = asset

	// Settlement <- Exchange
	settlement := protocol.NewSettlement()
	settlement.AssetType = exchange.Party1AssetType
	settlement.AssetID = exchange.Party1AssetID
	settlement.Party1TokenQty = party1Balance
	settlement.Party2TokenQty = party2Balance
	settlement.Timestamp = uint64(time.Now().Unix())

	// try to return as much as possible to party1, and we need to send
	// to the party2 address as well
	//
	// 2200 if the fee for Exchange
	party1OutValue := r.receivers[0].Value - 2200

	if party1OutValue < 546 {
		party1OutValue = 546
	}

	// Outputs
	outputs, err := h.buildOutputs(r)
	if err != nil {
		return nil, err
	}

	resp := contractResponse{
		Contract: c,
		Message:  &settlement,
		outs:     outputs,
	}

	return &resp, nil
}

func (h exchangeHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
	party1Addr := r.senders[0]
	party2Addr := r.receivers[2].Address

	contractAddress, err := r.contract.Address()
	if err != nil {
		return nil, err
	}

	exchange, ok := r.m.(*protocol.Exchange)
	if !ok {
		return nil, errors.New("Not *protocol.Exchange")
	}

	// the TX needs to pay to the Receiver as well, so add that here.
	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: party1Addr,
			Value:   dustLimit, // any change will be added to this output value
		},
		txbuilder.TxOutput{
			Address: party2Addr,
			Value:   dustLimit,
		},
		txbuilder.TxOutput{
			Address: contractAddress,
			Value:   dustLimit,
		},
	}

	// optional contract fee
	if h.Fee.Value > 0 {
		o := txbuilder.TxOutput{
			Address: h.Fee.Address,
			Value:   h.Fee.Value,
		}

		outs = append(outs, o)
	}

	// Optional exchange fee.
	if exchange.ExchangeFeeFixed > 0 {
		a := string(exchange.ExchangeFeeAddress)
		addr, err := btcutil.DecodeAddress(a, &chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}

		// convert BCH to Satoshi's
		o := txbuilder.TxOutput{
			Address: addr,
			Value:   txbuilder.ConvertBCHToSatoshis(exchange.ExchangeFeeFixed),
		}

		outs = append(outs, o)
	}

	return outs, nil
}
