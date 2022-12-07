package node

import (
	"context"
	"fmt"

	"github.com/tokenized/inspector"
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

var (
	// ErrSystemError occurs for a non standard response.
	ErrSystemError = errors.New("System error")

	// ErrNoResponse occurs when there is no response.
	ErrNoResponse = errors.New("No response given")

	// ErrRejected occurs for a rejected response.
	ErrRejected = errors.New("Request rejected")

	// ErrInsufficientFunds occurs for a poorly funded request.
	ErrInsufficientFunds = errors.New("Insufficient funds")
)

// Error handles all error responses for the API.
func Error(ctx context.Context, w *ResponseWriter, err error) {
	// switch errors.Cause(err) {
	// }

	// fmt.Printf("Error : %s\n", err)
	LogDepth(ctx, logger.LevelWarn, 1, "%s", err)
}

// RespondReject sends a rejection message.
// If no reject output data is specified, then the remainder is sent to the PKH of the first input.
// Since most bitcoins sent to the contract are just for response tx fee funding, this isn't a real
//   issue.
// The scenario in which it is important is when there is a multi-party transfer involving
//   bitcoins. In this scenario inputs send bitcoins to the smart contract to distribute to
//   receivers based on the transfer request data. We will need to analyze the transfer request
//   data to determine which inputs were to have funded sending bitcoins, and return the bitcoins
//   to them.
func RespondReject(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction,
	wk *wallet.Key, code uint32) error {
	return RespondRejectText(ctx, w, itx, wk, code, "")
}

func RespondRejectText(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction,
	wk *wallet.Key, code uint32, text string) error {

	rejectionCode := actions.RejectionsData(code)
	if rejectionCode == nil {
		Error(ctx, w, fmt.Errorf("Rejection code %d not found", code))
		return ErrNoResponse
	}

	v := ctx.Value(KeyValues).(*Values)

	// Build rejection
	rejection := actions.Rejection{
		RejectionCode: code,
		Message:       rejectionCode.Label,
		Timestamp:     v.Now.Nano(),
	}

	if len(text) > 0 {
		rejection.Message += ": " + text
	}

	// Contract address
	contractLockingScript := wk.LockingScript

	// Find spendable UTXOs
	var utxos []bitcoin.UTXO
	var err error
	if len(w.RejectInputs) > 0 {
		utxos = w.RejectInputs // Custom UTXOs. Just refund anything available to them.
	} else {
		utxos, err = itx.UTXOs().ForLockingScript(contractLockingScript)
		if err != nil {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	if len(utxos) == 0 {
		Error(ctx, w, errors.New("Contract UTXOs not found"))
		return ErrNoResponse // Contract UTXOs not found
	}

	// Create reject tx. Change goes back to requestor.
	rejectTx := txbuilder.NewTxBuilder(w.Config.FeeRate, w.Config.DustFeeRate)
	if len(w.RejectOutputs) > 0 {
		var changeLockingScript bitcoin.Script
		for _, output := range w.RejectOutputs {
			if output.Change {
				changeLockingScript = output.LockingScript
				break
			}
		}
		if len(changeLockingScript) == 0 {
			changeLockingScript = w.RejectOutputs[0].LockingScript
		}
		rejectTx.SetChangeLockingScript(changeLockingScript, "")
	} else {
		rejectTx.SetChangeLockingScript(itx.Inputs[0].LockingScript, "")
	}

	for _, utxo := range utxos {
		rejectTx.AddInputUTXO(utxo)
	}

	// Add a dust output to the requestor, but so they will also receive change.
	if len(w.RejectOutputs) > 0 {
		rejectAddressFound := false
		for i, output := range w.RejectOutputs {
			dustLimit := txbuilder.DustLimitForLockingScript(output.LockingScript,
				w.Config.DustFeeRate)
			if output.Value < dustLimit {
				output.Value = dustLimit
			}
			rejectTx.AddOutput(output.LockingScript, output.Value, output.Change, false)
			rejection.AddressIndexes = append(rejection.AddressIndexes, uint32(i))
			if len(w.RejectLockingScript) > 0 && output.LockingScript.Equal(w.RejectLockingScript) {
				rejectAddressFound = true
				rejection.RejectAddressIndex = uint32(i)
			}
		}
		if !rejectAddressFound && len(w.RejectLockingScript) > 0 {
			rejection.AddressIndexes = append(rejection.AddressIndexes, uint32(len(rejectTx.Outputs)))
			rejectTx.AddOutput(w.RejectLockingScript, 0, false, true)
		}
	} else {
		// Give it all back to the first input. This is the common scenario when the first input is
		//   the only requestor involved.
		rejectTx.AddOutput(itx.Inputs[0].LockingScript, 0, true, true)
		rejection.AddressIndexes = append(rejection.AddressIndexes, 0)
		rejection.RejectAddressIndex = 0
	}

	// Add the rejection payload
	payload, err := protocol.Serialize(&rejection, w.Config.IsTest)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}
	rejectTx.AddOutput(payload, 0, false, false)

	// Sign the tx
	if _, err := rejectTx.Sign([]bitcoin.Key{wk.Key}); err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	responseItx, err := inspector.NewTransactionFromTxBuilder(ctx, rejectTx, w.Config.IsTest)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	if err := Respond(ctx, w, responseItx); err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	Log(ctx, "Sending reject : %s", rejection.Message)
	return ErrRejected
}

// RespondSuccess broadcasts a successful message
func RespondSuccess(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction,
	wk *wallet.Key, msg actions.Action) error {

	// Create respond tx. Use contract address as backup change
	// address if an output wasn't specified
	respondTx := txbuilder.NewTxBuilder(w.Config.FeeRate, w.Config.DustFeeRate)
	respondTx.SetChangeLockingScript(w.Config.FeeLockingScript, "")

	// Get the specified UTXOs, otherwise look up the spendable
	// UTXO's received for the contract address
	var utxos []bitcoin.UTXO
	var err error
	if len(w.Inputs) > 0 {
		utxos = w.Inputs
	} else {
		utxos, err = itx.UTXOs().ForLockingScript(wk.LockingScript)
		if err != nil {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	// Add specified inputs
	for _, utxo := range utxos {
		respondTx.AddInputUTXO(utxo)
	}

	// Add specified outputs
	for _, out := range w.Outputs {
		err := respondTx.AddOutput(out.LockingScript, out.Value, out.Change, false)
		if err != nil {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	// Add the payload
	payload, err := protocol.Serialize(msg, w.Config.IsTest)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}
	respondTx.AddOutput(payload, 0, false, false)

	// Sign the tx
	if _, err := respondTx.Sign([]bitcoin.Key{wk.Key}); err != nil {
		if errors.Cause(err) == txbuilder.ErrInsufficientValue {
			LogWarn(ctx, "Sending reject. Failed to sign tx : %s\n%s", err,
				respondTx.String(w.Config.Net))
			return RespondRejectText(ctx, w, itx, wk, actions.RejectionsInsufficientTxFeeFunding,
				err.Error())
		} else {
			Error(ctx, w, err)
			return ErrNoResponse
		}
	}

	responseItx, err := inspector.NewTransactionFromTxBuilder(ctx, respondTx, w.Config.IsTest)
	if err != nil {
		Error(ctx, w, err)
		return ErrNoResponse
	}

	return Respond(ctx, w, responseItx)
}

// Respond sends a TX to the network.
func Respond(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction) error {
	Log(ctx, "Responding with tx : %s", itx.Hash)

	// Save Tx. Since state isn't saved it will not be considered already processed and will be
	// processed normally when it feeds back through from spynode.
	if err := transactions.AddTx(ctx, w.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	return w.Respond(ctx, itx.MsgTx)
}
