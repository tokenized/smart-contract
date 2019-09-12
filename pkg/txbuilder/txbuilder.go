package txbuilder

import (
	"bytes"
	"errors"
	"fmt"

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
}

// NewTx returns a new TxBuilder with the specified change PKH
// changePKH (Public Key Hash) is a 20 byte slice.
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

func NewTxBuilderFromWire(changeAddress bitcoin.RawAddress, dustLimit uint64, feeRate float32, tx *wire.MsgTx, inputs []*wire.MsgTx) (*TxBuilder, error) {
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
			IsChange: bytes.Equal(output.PkScript, changeScript),
			IsDust:   false,
		}
		result.Outputs = append(result.Outputs, &newOutput)
	}

	return &result, nil
}

// InputAddress returns the address that is paying to the input.
func (tx *TxBuilder) InputAddress(index int) (bitcoin.RawAddress, error) {
	if index >= len(tx.Inputs) {
		return bitcoin.RawAddress{}, errors.New("Input index out of range")
	}
	return bitcoin.RawAddressFromLockingScript(tx.Inputs[index].LockingScript)
}

// OutputAddress returns the address that the output is paying to.
func (tx *TxBuilder) OutputAddress(index int) (bitcoin.RawAddress, error) {
	if index >= len(tx.MsgTx.TxOut) {
		return bitcoin.RawAddress{}, errors.New("Output index out of range")
	}
	return bitcoin.RawAddressFromLockingScript(tx.MsgTx.TxOut[index].PkScript)
}

// Serialize returns the byte payload of the transaction.
func (tx *TxBuilder) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	if err := tx.MsgTx.Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
