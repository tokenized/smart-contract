package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type assetModificationHandler struct {
	Fee config.Fee
}

func newAssetModificationHandler(fee config.Fee) assetModificationHandler {
	return assetModificationHandler{
		Fee: fee,
	}
}

func (h assetModificationHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	am, ok := r.m.(*protocol.AssetModification)
	if !ok {
		return nil, errors.New("Not *protocol.AssetModification")
	}

	// Contract
	c := r.contract

	// we know the asset exists, already checked
	assetID := string(am.AssetID)
	asset, _ := c.Assets[assetID]

	asset = contract.NewAssetFromAssetModification(asset, am)
	c.Assets[asset.ID] = asset

	ac := buildAssetCreationFromAssetModification(am)

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
func (h assetModificationHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
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
