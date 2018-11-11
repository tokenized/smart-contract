package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type contractOfferHandler struct {
	Fee config.Fee
}

func newContractOfferHandler(fee config.Fee) contractOfferHandler {
	return contractOfferHandler{
		Fee: fee,
	}
}

func (h contractOfferHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	o, ok := r.m.(*protocol.ContractOffer)
	if !ok {
		return nil, errors.New("Not *protocol.ContractOffer")
	}

	// the request was received from the issuer. The Service has already
	// verified that there is no existing Contract at this addrress in
	// prior validation.
	f := buildContractFormationFromContractOffer(ctx, *o)

	outputs, err := h.buildOutputs(r)
	if err != nil {
		return nil, err
	}

	resp := contractResponse{
		Contract: r.contract,
		Message:  f,
		outs:     outputs,
	}

	return &resp, nil
}

func (h contractOfferHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
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
