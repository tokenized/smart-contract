package node

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
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
	MinimumForResponse = 2000 // TODO This should be variable depending on the size of the payload. Fee rate * (payload size + response inputs and outputs)
)

// Error handles all error responses for the API.
func Error(ctx context.Context, w *ResponseWriter, err error) {
	logger.Error(ctx, "%s", err)
	// switch errors.Cause(err) {
	// }

	// This should simply log the message somewhere
}

// RespondReject sends a rejection message
func RespondReject(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey, code uint8) error {

	// Sender is the address that sent the message that we are rejecting.
	sender := itx.Inputs[0].Address

	// Receiver (contract) is the address sending the message (UTXO)
	receiver := itx.Outputs[0]
	if uint64(receiver.Value) < MinimumForResponse {
		// Did not receive enough to fund the response
		Error(ctx, w, ErrInsufficientFunds)
		return ErrNoResponse
	}

	// Create reject tx
	rejectTx := txbuilder.NewTx(receiver.Address.ScriptAddress(), w.Config.DustLimit, w.Config.FeeRate)

	// Find spendable UTXOs
	utxos, err := itx.UTXOs().ForAddress(receiver.Address)
	if err != nil {
		Error(ctx, w, ErrInsufficientFunds)
		return ErrNoResponse
	}

	for _, utxo := range utxos {
		rejectTx.AddInput(wire.OutPoint{Hash: utxo.Hash, Index: utxo.Index}, utxo.PkScript, uint64(utxo.Value))
	}

	// Add a dust output to the sender, but so they will also receive change.
	rejectTx.AddP2PKHDustOutput(sender.ScriptAddress(), true)

	// Build rejection
	rejection := protocol.Rejection{
		RejectionType:  code,
		MessagePayload: string(protocol.RejectionCodes[code]),
	}

	// Add the rejection payload
	payload, err := protocol.Serialize(&rejection)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}
	rejectTx.AddOutput(payload, 0, false, false)

	// Sign the tx
	err = rejectTx.Sign(ctx, []*btcec.PrivateKey{rk.PrivateKey})
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	if err := Respond(ctx, w, rejectTx.MsgTx); err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}
	return ErrRejected
}

// RespondSuccess broadcasts a successful message
func RespondSuccess(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey, msg protocol.OpReturnMessage) error {

	// Create respond tx. Use contract address as backup change
	//address if an output wasn't specified
	respondTx := txbuilder.NewTx(rk.Address.ScriptAddress(), w.Config.DustLimit, w.Config.FeeRate)

	// Get the specified UTXOs, otherwise look up the spendable
	// UTXO's received for the contract address
	var utxos []inspector.UTXO
	var err error
	if len(w.Inputs) > 0 {
		utxos = w.Inputs
	} else {
		utxos, err = itx.UTXOs().ForAddress(rk.Address)
		if err != nil {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	// Add specified inputs
	for _, utxo := range utxos {
		respondTx.AddInput(wire.OutPoint{Hash: utxo.Hash, Index: utxo.Index}, utxo.PkScript, uint64(utxo.Value))
	}

	// Add specified outputs
	for _, out := range w.Outputs {
		err := respondTx.AddOutput(txbuilder.P2PKHScriptForPKH(out.Address.ScriptAddress()), out.Value, out.Change, false)
		if err != nil {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	// Add the payload
	payload, err := protocol.Serialize(msg)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}
	respondTx.AddOutput(payload, 0, false, false)

	// Sign the tx
	err = respondTx.Sign(ctx, []*btcec.PrivateKey{rk.PrivateKey})
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	return Respond(ctx, w, respondTx.MsgTx)
}

// Respond sends a TX to the network.
func Respond(ctx context.Context, w *ResponseWriter, tx *wire.MsgTx) error {
	return w.Respond(ctx, tx)
}
