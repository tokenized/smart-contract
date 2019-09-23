package txbuilder

import (
	"bytes"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

const (
	SubSystem = "TxBuilder" // For logger

	// DefaultVersion is the default TX version to use by the Buidler.
	DefaultVersion = int32(1)
)

type TxBuilder struct {
	MsgTx         *wire.MsgTx
	Inputs        []*InputSupplement  // Input Data that is not in wire.MsgTx
	Outputs       []*OutputSupplement // Output Data that is not in wire.MsgTx
	ChangeAddress bitcoin.RawAddress  // The address to pay extra bitcoins to if a change output isn't specified
	DustLimit     uint64              // Smallest amount of bitcoin for a valid spendable output
	FeeRate       float32             // The target fee rate in sat/byte

	// Optional identifier for external use to track the key needed to spend change
	ChangeKeyID string
}

// NewTxBuilder returns a new TxBuilder with the specified change address.
func NewTxBuilder(changeAddress bitcoin.RawAddress, dustLimit uint64, feeRate float32) *TxBuilder {
	tx := wire.MsgTx{Version: DefaultVersion, LockTime: 0}
	result := TxBuilder{
		MsgTx:         &tx,
		ChangeAddress: changeAddress,
		DustLimit:     dustLimit,
		FeeRate:       feeRate,
	}
	return &result
}

// NewTxBuilderFromWire returns a new TxBuilder from a wire.MsgTx and the additional information
//   required.
func NewTxBuilderFromWire(changeAddress bitcoin.RawAddress, dustLimit uint64, feeRate float32,
	tx *wire.MsgTx, inputs []*wire.MsgTx) (*TxBuilder, error) {

	result := TxBuilder{
		MsgTx:         tx,
		ChangeAddress: changeAddress,
		DustLimit:     dustLimit,
		FeeRate:       feeRate,
	}

	// Setup inputs
	for _, input := range result.MsgTx.TxIn {
		found := false
		for _, inputTx := range inputs {
			txHash := inputTx.TxHash()
			if bytes.Equal(txHash[:], input.PreviousOutPoint.Hash[:]) &&
				int(input.PreviousOutPoint.Index) < len(inputTx.TxOut) {
				// Add input
				newInput := InputSupplement{
					LockingScript: inputTx.TxOut[input.PreviousOutPoint.Index].PkScript,
					Value:         uint64(inputTx.TxOut[input.PreviousOutPoint.Index].Value),
				}
				result.Inputs = append(result.Inputs, &newInput)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("Input tx not found : %s %d", input.PreviousOutPoint.Hash, input.PreviousOutPoint.Index)
		}
	}

	// Setup outputs
	changeScript, err := result.ChangeAddress.LockingScript()
	if err != nil {
		return nil, err
	}
	for _, output := range result.MsgTx.TxOut {
		newOutput := OutputSupplement{
			IsRemainder: bytes.Equal(output.PkScript, changeScript),
			IsDust:      false,
			KeyID:       result.ChangeKeyID,
		}
		result.Outputs = append(result.Outputs, &newOutput)
	}

	return &result, nil
}

// NewSendMaxTxBuilder returns a new TxBuilder that spends all specified UTXOs to the specified
//   addresses.
// If more than one address is specified, then the amounts have to be specified for all but the
//   last. The last output amount will be the remaining value after the fee is taken.
// Note : The last output will be marked as "change" even though it is not necessarily an owned
//   "change" address. This is to ensure it behaves properly and takes the remaining balance.
func NewSendMaxTxBuilder(utxos []bitcoin.UTXO, toAddresses []bitcoin.RawAddress, toAmounts []uint64,
	dustLimit uint64, feeRate float32) (*TxBuilder, error) {

	if len(toAddresses) != len(toAmounts)+1 {
		return nil, errors.New("Must provide 1 more addresses than amounts in send max")
	}

	tx := wire.MsgTx{Version: DefaultVersion, LockTime: 0}
	result := TxBuilder{
		MsgTx:   &tx,
		FeeRate: feeRate,
	}

	var err error
	totalAmount := uint64(0)

	// Add all UTXOs as inputs.
	for _, utxo := range utxos {
		totalAmount += utxo.Value

		err = result.AddInputUTXO(utxo)
		if err != nil {
			return nil, errors.Wrap(err, "adding input")
		}
	}

	// Add payment outputs to the toAddresses.
	sentAmount := uint64(0)
	for i, toAddress := range toAddresses {
		if i < len(toAmounts) {
			sentAmount += toAmounts[i]
			if sentAmount > totalAmount {
				return nil, newError(ErrorCodeInsufficientValue, fmt.Sprintf("Sent too much in send max"))
			}

			err = result.AddPaymentOutput(toAddress, toAmounts[i], false)
			if err != nil {
				return nil, errors.Wrap(err, "adding output")
			}
		} else { // Last output
			if totalAmount-sentAmount < result.DustLimit {
				return nil, newError(ErrorCodeInsufficientValue, fmt.Sprintf("Last output below dust in send max"))
			}
			err = result.AddPaymentOutput(toAddress, totalAmount-sentAmount, true)
			if err != nil {
				return nil, errors.Wrap(err, "adding output")
			}
		}
	}

	// Calculate fee
	_, err = result.adjustFee(int64(result.EstimatedFee()))
	if err != nil {
		return nil, errors.Wrap(err, "adjusting fee")
	}

	return &result, nil
}

// Serialize returns the byte payload of the transaction.
func (tx *TxBuilder) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	if err := tx.MsgTx.Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
