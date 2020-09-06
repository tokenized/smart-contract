package contracts

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/spynode/handlers"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
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

// HandleBlock handles a block message from spynode.
// Implements the spynode Listener interface.
func (ch *ContractsHandler) HandleBlock(ctx context.Context, msgType int,
	block *handlers.BlockMessage) error {
	return nil
}

// HandleTx handles a tx message from spynode.
// Implements the spynode Listener interface.
func (ch *ContractsHandler) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {

	for _, output := range tx.TxOut {
		// Check for C2 for identity oracle, authority oracle, or operator
		action, err := protocol.Deserialize(output.PkScript, ch.isTest)
		if err != nil {
			continue // not a Tokenized action
		}

		if action.Code() != actions.CodeContractFormation {
			continue // not a contract formation
		}

		caOut, err := bitcoin.RawAddressFromLockingScript(tx.TxOut[0].PkScript)
		if err != nil {
			return false, nil // not a contract address
		}

		ctx = logger.ContextWithOutLogSubSystem(ctx)
		ctx = logger.ContextWithLogTrace(ctx, tx.TxHash().String())

		// Fetch outpoint output to verify input address
		if tx.TxIn[0].PreviousOutPoint.Index == 0xffffffff {
			logger.Warn(ctx, "Contract formation with coinbase input : %s",
				tx.StringWithAddresses(ch.net))
			return false, nil
		}

		outputs, err := ch.outputFetcher.GetOutputs(ctx,
			[]wire.OutPoint{tx.TxIn[0].PreviousOutPoint})
		if err != nil {
			logger.Error(ctx, "Failed to retrieve outputs : %s", err)
			return false, nil
		}

		if int(tx.TxIn[0].PreviousOutPoint.Index) >= len(outputs) {
			logger.Warn(ctx, "Failed to retrieve outputs : index %d >= output count %d",
				tx.TxIn[0].PreviousOutPoint.Index, len(outputs))
		}

		ls := outputs[tx.TxIn[0].PreviousOutPoint.Index].LockingScript
		caIn, err := bitcoin.RawAddressFromLockingScript(ls)
		if err != nil {
			logger.Error(ctx, "Contract formation with invalid input address : %s", err)
			return false, nil
		}

		if !caIn.Equal(caOut) {
			logger.Warn(ctx, "Contract formation with invalid input : input %s, output %s",
				bitcoin.NewAddressFromRawAddress(caIn, ch.net).String(),
				bitcoin.NewAddressFromRawAddress(caOut, ch.net).String())
			return false, nil
		}

		logger.Verbose(ctx, "Processing contract formation : %s",
			bitcoin.NewAddressFromRawAddress(caIn, ch.net).String())

		if err := ch.processor.SaveContractFormation(ctx, caIn, output.PkScript); err != nil {
			logger.Error(ctx, "Failed to process contract formation : %s", err)
			return false, nil
		}

		return false, nil // we don't really care about tx states
	}

	return false, nil // we don't really care about tx states
}

// HandleTxState handles a tx state message from spynode.
// Implements the spynode Listener interface.
func (ch *ContractsHandler) HandleTxState(ctx context.Context, msgType int,
	txid bitcoin.Hash32) error {
	return nil
}

// HandleInSync handles an in sync message from spynode.
// Implements the spynode Listener interface.
func (ch *ContractsHandler) HandleInSync(ctx context.Context) error {
	return nil
}
