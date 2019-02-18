package request

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/config"
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

	ca, ok := r.m.(*protocol.ContractAmendment)
	if !ok {
		return nil, errors.New("Not *protocol.ContractAmendment")
	}

	// Contract
	contract := r.contract

	// Bump the revision
	newRevision := contract.Revision + 1

	// Contract Formation <- Contract Amendment
	cf := protocol.NewContractFormation()
	cf.Version = ca.Version
	cf.ContractName = ca.ContractName
	cf.ContractFileHash = ca.ContractFileHash
	cf.GoverningLaw = ca.GoverningLaw
	cf.Jurisdiction = ca.Jurisdiction
	cf.ContractExpiration = ca.ContractExpiration
	cf.URI = ca.URI
	cf.ContractRevision = newRevision
	cf.IssuerID = ca.IssuerID
	cf.IssuerType = ca.IssuerType
	cf.ContractOperatorID = ca.ContractOperatorID
	cf.AuthorizationFlags = ca.AuthorizationFlags
	cf.VotingSystem = ca.VotingSystem
	cf.InitiativeThreshold = ca.InitiativeThreshold
	cf.InitiativeThresholdCurrency = ca.InitiativeThresholdCurrency
	cf.RestrictedQty = ca.RestrictedQty

	// Outputs
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
		Contract: contract,
		Message:  &cf,
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
			Address: r.senders[0].Address,
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
