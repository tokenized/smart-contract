package node

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	// ErrSystemError occurs for a non standard response.
	ErrSystemError = errors.New("System error")

	// ErrNoResponse occurs when there is no response.
	ErrNoResponse = errors.New("No response given")

	// ErrRejected occurs for a rejected response.
	ErrRejected = errors.New("Transaction rejected")

	// ErrInsufficientFunds occurs for a poorly funded request.
	ErrInsufficientFunds = errors.New("Insufficient Payment amount")
)

const (
	MinimumForResponse = protocol.LimitDefault
)

// Output is an output address for a response
type Output struct {
	Address btcutil.Address
	Value   uint64
	Change  bool
}

// OutputFee prepares a special fee output based on node configuration
func OutputFee(ctx context.Context, config *Config) *Output {
	if config.FeeValue > 0 {
		feeAddr, _ := btcutil.DecodeAddress(config.FeeAddress, &chaincfg.MainNetParams)
		return &Output{
			Address: feeAddr,
			Value:   config.FeeValue,
		}
	}

	return nil
}

// Error handles all error responses for the API.
func Error(ctx context.Context, mux protomux.Handler, err error) {
	// switch errors.Cause(err) {
	// }

	// This should simply log the message somewhere
}

// RespondReject sends a rejection message
func RespondReject(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey, code uint8) error {

	// Sender is the address that sent the message that we are rejecting.
	sender := itx.Inputs[0].Address

	// Receiver (contract) is the address sending the message (UTXO)
	receiver := itx.Outputs[0]
	if uint64(receiver.Value) < MinimumForResponse {
		// Did not receive enough to fund the response
		Error(ctx, mux, ErrInsufficientFunds)
		return ErrNoResponse
	}

	// Find spendable UTXOs
	utxos, err := itx.UTXOs().ForAddress(receiver.Address)
	if err != nil {
		Error(ctx, mux, ErrInsufficientFunds)
		return ErrNoResponse
	}

	// Build rejection
	rejection := protocol.NewRejection()
	rejection.RejectionType = code
	rejection.Message = protocol.RejectionCodes[code]

	// Sending the message to the sender of the message being rejected
	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: sender,
			Value:   546,
		},
	}

	// We spend the UTXO's to respond to the sender (+ others).
	//
	// The UTXOs to spend are in the TX we received.
	changeAddress := sender

	// Build the new transaction
	newTx, err := wallet.BuildTX(rk, utxos, outs, changeAddress, &rejection)
	if err != nil {
		Error(ctx, mux, err)
	}

	if err := Respond(ctx, mux, newTx); err != nil {
		return err
	}
	return ErrRejected
}

// RespondSuccess broadcasts a successful message
func RespondSuccess(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey,
	msg protocol.OpReturnMessage, outs []Output) error {

	// Get spendable UTXO's received for the contract address
	utxos, err := itx.UTXOs().ForAddress(rk.Address)
	if err != nil {
		return err
	}

	return RespondUTXO(ctx, mux, itx, rk, msg, outs, utxos)
}

// RespondUTXO broadcasts a successful message using a specific UTXO
func RespondUTXO(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey,
	msg protocol.OpReturnMessage, outs []Output, utxos []inspector.UTXO) error {

	var change btcutil.Address

	var buildOuts []txbuilder.TxOutput
	for _, out := range outs {
		buildOuts = append(buildOuts, txbuilder.TxOutput{
			Address: out.Address,
			Value:   uint64(out.Value),
		})

		// Change output
		if out.Change {
			change = out.Address
		}
	}

	// At least one change output is required
	if change == nil {
		return errors.New("Missing change output")
	}

	// Build the new transaction
	newTx, err := wallet.BuildTX(rk, utxos, buildOuts, change, msg)
	if err != nil {
		return err
	}

	return Respond(ctx, mux, newTx)
}

// Respond sends a TX to the network.
func Respond(ctx context.Context, mux protomux.Handler, tx *wire.MsgTx) error {
	return mux.Respond(ctx, tx)
}
