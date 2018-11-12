package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type assetDefinitionHandler struct {
	Fee config.Fee
}

// newAssetDefinitionHandler returns a new assetDefinitionHandler.
func newAssetDefinitionHandler(fee config.Fee) assetDefinitionHandler {
	return assetDefinitionHandler{
		Fee: fee,
	}
}

// handle attempts to process the contractRequest.
func (h assetDefinitionHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	ad, ok := r.m.(*protocol.AssetDefinition)
	if !ok {
		return nil, errors.New("Not *protocol.AssetDefinition")
	}

	// Contract
	c := r.contract

	// asset will be assigned to the Issuer address
	issuerAddr := r.senders[0].EncodeAddress()
	holding := contract.NewHolding(issuerAddr, ad.Qty)
	asset := contract.NewAssetFromAssetDefinition(ad, holding)
	c.Assets[asset.ID] = asset

	// Asset Creation <- Asset Definition
	ac := protocol.NewAssetCreation()
	ac.AssetType = ad.AssetType
	ac.AssetID = ad.AssetID
	ac.AssetRevision = 0
	ac.AuthorizationFlags = ad.AuthorizationFlags
	ac.VotingSystem = ad.VotingSystem
	ac.VoteMultiplier = ad.VoteMultiplier
	ac.Qty = ad.Qty
	ac.ContractFeeCurrency = ad.ContractFeeCurrency
	ac.ContractFeeVar = ad.ContractFeeVar
	ac.ContractFeeFixed = ad.ContractFeeFixed
	ac.Payload = ad.Payload

	// Outputs
	outputs, err := h.buildOutputs(r)
	if err != nil {
		return nil, err
	}

	resp := contractResponse{
		Contract: c,
		Message:  &ac,
		outs:     outputs,
	}

	return &resp, nil
}

// buildOutputs
//
// 0 : Contract's Public Address (change, added later when calculated)
// 1 : Issuer's Public Address (546)
// 2 : Contract Fee Address (fee amount)
// 3 : OP_RETURN (Asset Creation, 0 sats)
func (h assetDefinitionHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
	contractAddress, err := r.contract.Address()
	if err != nil {
		return nil, err
	}

	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: contractAddress,
			Value:   dustLimit,
		},
		txbuilder.TxOutput{
			Address: r.senders[0],
			Value:   dustLimit, // any change will be added to this output value
		},
	}

	if h.Fee.Value > 0 {
		feeOutput := txbuilder.TxOutput{
			Address: h.Fee.Address,
			Value:   h.Fee.Value,
		}

		outs = append(outs, feeOutput)
	}

	return outs, nil
}
