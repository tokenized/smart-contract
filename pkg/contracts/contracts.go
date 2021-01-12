package contracts

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
	"github.com/tokenized/spynode/pkg/client"
)

// ContractsHandler provides all of the contract formation actions on the network when it is
// registered as a listener on spynode.
type ContractsHandler struct {
	net           bitcoin.Network
	isTest        bool
	outputFetcher OutputFetcher
	processor     ContractProcessor
}

// NewContractsHandler creates a contracts handler.
func NewContractsHandler(f OutputFetcher, net bitcoin.Network, isTest bool,
	p ContractProcessor) *ContractsHandler {

	return &ContractsHandler{
		net:           net,
		isTest:        isTest,
		outputFetcher: f,
		processor:     p,
	}
}

// ContractProcessor saves or does other processing of contract formation actions.
type ContractProcessor interface {
	SaveContractFormation(ctx context.Context, ra bitcoin.RawAddress, script []byte) error
}

// OutputFetcher provides the ability to fetch the outputs being spent in a tx to allow confirmation
// of the smart contract agent address.
type OutputFetcher interface {
	GetOutputs(ctx context.Context, outpoints []wire.OutPoint) ([]bitcoin.UTXO, error)
}

func (ch *ContractsHandler) HandleTx(ctx context.Context, tx *client.Tx) {
	for _, output := range tx.Tx.TxOut {
		// Check for C2 for identity oracle, authority oracle, or operator
		action, err := protocol.Deserialize(output.PkScript, ch.isTest)
		if err != nil {
			continue // not a Tokenized action
		}

		if action.Code() != actions.CodeContractFormation {
			continue // not a contract formation
		}

		caOut, err := bitcoin.RawAddressFromLockingScript(tx.Tx.TxOut[0].PkScript)
		if err != nil {
			return // not a contract address
		}

		ctx = logger.ContextWithOutLogSubSystem(ctx)
		ctx = logger.ContextWithLogTrace(ctx, tx.Tx.TxHash().String())

		// Fetch outpoint output to verify input address
		if tx.Tx.TxIn[0].PreviousOutPoint.Index == 0xffffffff {
			logger.Warn(ctx, "Contract formation with coinbase input : %s",
				tx.Tx.StringWithAddresses(ch.net))
			return
		}

		outputs, err := ch.outputFetcher.GetOutputs(ctx,
			[]wire.OutPoint{tx.Tx.TxIn[0].PreviousOutPoint})
		if err != nil {
			logger.Error(ctx, "Failed to retrieve outputs : %s", err)
			return
		}

		if int(tx.Tx.TxIn[0].PreviousOutPoint.Index) >= len(outputs) {
			logger.Warn(ctx, "Failed to retrieve outputs : index %d >= output count %d",
				tx.Tx.TxIn[0].PreviousOutPoint.Index, len(outputs))
		}

		ls := outputs[tx.Tx.TxIn[0].PreviousOutPoint.Index].LockingScript
		caIn, err := bitcoin.RawAddressFromLockingScript(ls)
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

		if err := ch.processor.SaveContractFormation(ctx, caIn, output.PkScript); err != nil {
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
