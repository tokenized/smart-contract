package txbuilder

import (
	"bytes"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
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
	SendMax       bool                // When set, AddFunding will add all UTXOs given

	// Optional identifier for external use to track the key needed to spend change
	ChangeKeyID string
}

// NewTxBuilder returns a new TxBuilder with the specified change address.
func NewTxBuilder(dustLimit uint64, feeRate float32) *TxBuilder {
	tx := wire.MsgTx{Version: DefaultVersion, LockTime: 0}
	result := TxBuilder{
		MsgTx:     &tx,
		DustLimit: dustLimit,
		FeeRate:   feeRate,
	}
	return &result
}

// NewTxBuilderFromWire returns a new TxBuilder from a wire.MsgTx and the input txs.
func NewTxBuilderFromWire(dustLimit uint64, feeRate float32, tx *wire.MsgTx,
	inputs []*wire.MsgTx) (*TxBuilder, error) {

	result := TxBuilder{
		MsgTx:     tx,
		DustLimit: dustLimit,
		FeeRate:   feeRate,
		Inputs:    make([]*InputSupplement, len(tx.TxIn)),
	}

	// Setup inputs
	var missingErr error
	for i, input := range result.MsgTx.TxIn {
		found := false
		for _, inputTx := range inputs {
			txHash := inputTx.TxHash()
			if bytes.Equal(txHash[:], input.PreviousOutPoint.Hash[:]) &&
				int(input.PreviousOutPoint.Index) < len(inputTx.TxOut) {
				// Add input
				result.Inputs[i] = &InputSupplement{
					LockingScript: inputTx.TxOut[input.PreviousOutPoint.Index].PkScript,
					Value:         uint64(inputTx.TxOut[input.PreviousOutPoint.Index].Value),
				}
				found = true
				break
			}
		}
		if !found {
			missingErr = newError(ErrorCodeMissingInputData, "")
		}
	}

	// Setup outputs
	result.Outputs = make([]*OutputSupplement, len(result.MsgTx.TxOut))
	for i, _ := range result.Outputs {
		result.Outputs[i] = &OutputSupplement{}
	}

	return &result, missingErr
}

// NewTxBuilderFromWireUTXOs returns a new TxBuilder from a wire.MsgTx and the input UTXOs.
func NewTxBuilderFromWireUTXOs(dustLimit uint64, feeRate float32, tx *wire.MsgTx,
	utxos []bitcoin.UTXO) (*TxBuilder, error) {

	result := TxBuilder{
		MsgTx:     tx,
		DustLimit: dustLimit,
		FeeRate:   feeRate,
		Inputs:    make([]*InputSupplement, len(tx.TxIn)),
	}

	// Setup inputs
	var missingErr error
	for i, input := range result.MsgTx.TxIn {
		found := false
		for _, utxo := range utxos {
			if utxo.Hash.Equal(&input.PreviousOutPoint.Hash) &&
				utxo.Index == input.PreviousOutPoint.Index {
				// Add input
				result.Inputs[i] = &InputSupplement{
					LockingScript: utxo.LockingScript,
					Value:         utxo.Value,
					KeyID:         utxo.KeyID,
				}
				found = true
				break
			}
		}
		if !found {
			missingErr = newError(ErrorCodeMissingInputData, "")
		}
	}

	// Setup outputs
	result.Outputs = make([]*OutputSupplement, len(result.MsgTx.TxOut))
	for i, _ := range result.Outputs {
		result.Outputs[i] = &OutputSupplement{}
	}

	return &result, missingErr
}

// Serialize returns the byte payload of the transaction.
func (tx *TxBuilder) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	if err := tx.MsgTx.Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
