package contracts

import (
	"context"

	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
	"github.com/tokenized/spynode/pkg/client"
)

// ContractsHandler provides all of the contract formation actions on the network when it is
// registered as a listener on spynode.
type ContractsHandler struct {
	net       bitcoin.Network
	isTest    bool
	processor ContractProcessor
}

// NewContractsHandler creates a contracts handler.
func NewContractsHandler(net bitcoin.Network, isTest bool,
	p ContractProcessor) *ContractsHandler {

	return &ContractsHandler{
		net:       net,
		isTest:    isTest,
		processor: p,
	}
}

// ContractProcessor saves or does other processing of contract formation actions.
type ContractProcessor interface {
	SaveContractFormation(ctx context.Context, ra bitcoin.RawAddress, script []byte) error
}

func (ch *ContractsHandler) HandleTx(ctx context.Context, tx *client.Tx) {
	if len(tx.Outputs) < 1 {
		return
	}

	for _, output := range tx.Tx.TxOut {
		// Check for C2 for identity oracle, authority oracle, or operator
		action, err := protocol.Deserialize(output.LockingScript, ch.isTest)
		if err != nil {
			continue // not a Tokenized action
		}

		if action.Code() != actions.CodeContractFormation {
			continue // not a contract formation
		}

		caOut, err := bitcoin.RawAddressFromLockingScript(tx.Tx.TxOut[0].LockingScript)
		if err != nil {
			return // not a contract address
		}

		ctx = logger.ContextWithOutLogSubSystem(ctx)
		ctx = logger.ContextWithLogTrace(ctx, tx.Tx.TxHash().String())

		if len(tx.Outputs) == 0 {
			logger.Warn(ctx, "Missing outputs")
			return
		}

		caIn, err := bitcoin.RawAddressFromLockingScript(tx.Outputs[0].LockingScript)
		if err != nil {
			logger.Error(ctx, "Contract formation with invalid input address : %s", err)
			return
		}

		if !caIn.Equal(caOut) {
			logger.Warn(ctx, "Contract formation with invalid input : input %s, output %s",
				bitcoin.NewAddressFromRawAddress(caIn, ch.net).String(),
				bitcoin.NewAddressFromRawAddress(caOut, ch.net).String())
			return
		}

		logger.Verbose(ctx, "Processing contract formation : %s",
			bitcoin.NewAddressFromRawAddress(caIn, ch.net).String())

		if err := ch.processor.SaveContractFormation(ctx, caIn, output.LockingScript); err != nil {
			logger.Error(ctx, "Failed to process contract formation : %s", err)
			return
		}
	}
}

func (ch *ContractsHandler) HandleTxUpdate(ctx context.Context, update *client.TxUpdate) {

}

func (ch *ContractsHandler) HandleHeaders(ctx context.Context, headers *client.Headers) {

}

func (ch *ContractsHandler) HandleInSync(ctx context.Context) {

}
