package request

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type sendHandler struct {
	Fee config.Fee
}

func newSendHandler(fee config.Fee) sendHandler {
	return sendHandler{
		Fee: fee,
	}
}

func (h sendHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	issue, ok := r.m.(*protocol.Send)
	if !ok {
		return nil, errors.New("Not *protocol.Issue")
	}

	// Contract
	c := r.contract

	// get the asset.
	k := string(issue.AssetID)
	asset, ok := c.Assets[k]
	if !ok {
		return nil, fmt.Errorf("send : Asset ID not found : contract=%s assetID=%s", c.ID, issue.AssetID)
	}

	// Bounds check for receivers - contract, receiver
	if len(r.receivers) < 2 {
		return nil, fmt.Errorf("Missing receivers")
	}

	// Party 1 (Sender)
	party1Addr := r.senders[0].EncodeAddress()
	party1Holding, ok := asset.Holdings[party1Addr]
	if !ok {
		return nil, fmt.Errorf("send : holding not found contract=%s assetID=%s party1=%s", c.ID, issue.AssetID, party1Addr)
	}
	party1Balance := party1Holding.Balance

	// Check the token balance
	if party1Balance < issue.TokenQty {
		return nil, fmt.Errorf("send : insufficient assets contract=%s assetID=%s party1=%s", c.ID, issue.AssetID, party1Addr)
	}

	// Party 2 (Receiver)
	party2Addr := r.receivers[1].Address.EncodeAddress()
	if party1Addr == party2Addr {
		return nil, fmt.Errorf("send : cannot transfer to own self contract=%s assetID=%s party1=%s", c.ID, issue.AssetID, party1Addr)
	}

	party2Holding, ok := asset.Holdings[party2Addr]
	party2Balance := uint64(0)
	if ok {
		party2Balance = party2Holding.Balance
	}

	// Modify balances
	party1Balance -= issue.TokenQty
	party2Balance += issue.TokenQty

	// Settlement <- Send
	settlement := protocol.NewSettlement()
	settlement.AssetType = issue.AssetType
	settlement.AssetID = issue.AssetID
	settlement.Party1TokenQty = party1Balance
	settlement.Party2TokenQty = party2Balance
	settlement.Timestamp = uint64(time.Now().Unix())

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

func (h sendHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
	party1Addr := r.senders[0]
	party2Addr := r.receivers[1].Address

	contractAddress, err := r.contract.Address()
	if err != nil {
		return nil, err
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
		feeOutput := txbuilder.TxOutput{
			Address: h.Fee.Address,
			Value:   h.Fee.Value,
		}

		outs = append(outs, feeOutput)
	}

	return outs, nil
}
