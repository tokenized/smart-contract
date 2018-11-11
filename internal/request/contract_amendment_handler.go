package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type contractAmendmentHandler struct {
	Fee config.Fee
}

// newContractAmendmentHandler returns a new contractAmendmentHandler.
func newContractAmendmentHandler(fee config.Fee) contractAmendmentHandler {
	return contractAmendmentHandler{
		Fee: fee,
	}
}

// handle attempts to process the contractRequest.
func (h contractAmendmentHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	a, ok := r.m.(*protocol.ContractAmendment)
	if !ok {
		return nil, errors.New("Not *protocol.ContractAmendment")
	}

	// Contract
	contract := r.contract
	newContract, b := buildContractFormationFromContractAmendment(contract, a)

	outputs, err := h.buildOutputs(r)
	if err != nil {
		return nil, err
	}

	// outs := []txbuilder.TxOutput{
	// 	txbuilder.TxOutput{
	// 		Address: r.sender,
	// 		Value:   dustLimit,
	// 	},
	// }

	resp := contractResponse{
		Contract: newContract,
		Message:  &b,
		outs:     outputs,
	}

	return &resp, nil
}

func (h contractAmendmentHandler) buildOutputs(r contractRequest) ([]txbuilder.TxOutput, error) {
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
