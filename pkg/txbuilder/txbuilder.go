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
	Inputs        []*InputSupplement     // Input Data that is not in wire.MsgTx
	Outputs       []*OutputSupplement    // Output Data that is not in wire.MsgTx
	ChangeAddress bitcoin.ScriptTemplate // The address to pay extra bitcoins to if a change output isn't specified
	DustLimit     uint64                 // Smallest amount of bitcoin for a valid spendable output
	FeeRate       float32                // The target fee rate in sat/byte
}

// NewTx returns a new TxBuilder with the specified change PKH
// changePKH (Public Key Hash) is a 20 byte slice.
func NewTxBuilder(changeAddress bitcoin.ScriptTemplate, dustLimit uint64, feeRate float32) *TxBuilder {
	tx := wire.MsgTx{Version: DefaultVersion, LockTime: 0}
	result := TxBuilder{
		MsgTx:         &tx,
		ChangeAddress: changeAddress,
		DustLimit:     dustLimit,
		FeeRate:       feeRate,
	}
	return &result
}

func NewTxBuilderFromWire(changeAddress bitcoin.ScriptTemplate, dustLimit uint64, feeRate float32, tx *wire.MsgTx, inputs []*wire.MsgTx) (*TxBuilder, error) {
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
					LockScript: inputTx.TxOut[input.PreviousOutPoint.Index].PkScript,
					Value:      uint64(inputTx.TxOut[input.PreviousOutPoint.Index].Value),
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
	for _, output := range result.MsgTx.TxOut {
		newOutput := OutputSupplement{
			IsChange: bytes.Equal(output.PkScript, result.ChangeAddress.LockingScript()),
			IsDust:   false,
		}
		result.Outputs = append(result.Outputs, &newOutput)
	}

	return &result, nil
}

// AddInput Adds an input to TxBuilder.
//   outpoint reference the output being spent.
//   lockScript is the script from the output being spent.
//   value is the number of satoshis from the output being spent.
func (tx *TxBuilder) AddInput(outpoint wire.OutPoint, lockScript []byte, value uint64) error {
	input := InputSupplement{
		LockScript: lockScript,
		Value:      value,
	}
	tx.Inputs = append(tx.Inputs, &input)

	txin := wire.TxIn{
		PreviousOutPoint: outpoint,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	tx.MsgTx.AddTxIn(&txin)
	return nil
}

// AddPaymentOutput adds an output to TxBuilder with the specified value and a script paying the
//   specified address.
// isChange marks the output to receive remaining bitcoin.
func (tx *TxBuilder) AddPaymentOutput(address bitcoin.ScriptTemplate, value uint64, isChange bool) error {
	return tx.AddOutput(address.LockingScript(), value, isChange, false)
}

// AddP2PKHDustOutput adds an output to TxBuilder with the dust limit amount and a script paying the
//   specified address.
// isChange marks the output to receive remaining bitcoin.
// These dust outputs are meant as "notifiers" so that an address will see this transaction and
//   process the data in it. If value is later added to this output, the value replaces the dust
//   limit amount rather than adding to it.
func (tx *TxBuilder) AddDustOutput(address bitcoin.ScriptTemplate, isChange bool) error {
	return tx.AddOutput(address.LockingScript(), tx.DustLimit, isChange, true)
}

// AddOutput adds an output to TxBuilder with the specified script and value.
// isChange marks the output to receive remaining bitcoin.
// isDust marks the output as a dust amount which will be replaced by any non-dust amount if an
//    amount is added later.
func (tx *TxBuilder) AddOutput(lockScript []byte, value uint64, isChange bool, isDust bool) error {
	output := OutputSupplement{
		IsChange: isChange,
		IsDust:   isDust,
	}
	tx.Outputs = append(tx.Outputs, &output)

	txout := wire.TxOut{
		Value:    int64(value),
		PkScript: lockScript,
	}
	tx.MsgTx.AddTxOut(&txout)
	return nil
}

// AddValueToOutput adds more bitcoin to an existing output.
func (tx *TxBuilder) AddValueToOutput(index uint32, value uint64) error {
	if int(index) >= len(tx.MsgTx.TxOut) {
		return errors.New("Output index out of range")
	}

	if tx.Outputs[index].IsDust {
		tx.Outputs[index].IsDust = false
		tx.MsgTx.TxOut[index].Value = int64(value)
	} else {
		tx.MsgTx.TxOut[index].Value += int64(value)
	}
	return nil
}

// UpdateOutput updates the locking script of an output.
func (tx *TxBuilder) UpdateOutput(index uint32, lockScript []byte) error {
	if int(index) >= len(tx.MsgTx.TxOut) {
		return errors.New("Output index out of range")
	}

	tx.MsgTx.TxOut[index].PkScript = lockScript
	return nil
}

// InputSupplement contains data required to sign an input that is not already in the wire.MsgTx.
type InputSupplement struct {
	LockScript []byte
	Value      uint64
}

// OutputSupplement contains data that
type OutputSupplement struct {
	IsChange    bool
	IsDust      bool
	addedForFee bool
}

// InputAddress returns the address that is paying to the input.
func (tx *TxBuilder) InputAddress(index int) (bitcoin.ScriptTemplate, error) {
	if index >= len(tx.Inputs) {
		return nil, errors.New("Input index out of range")
	}
	return bitcoin.ScriptTemplateFromLockingScript(tx.Inputs[index].LockScript)
}

// OutputAddress returns the address that the output is paying to.
func (tx *TxBuilder) OutputAddress(index int) (bitcoin.ScriptTemplate, error) {
	if index >= len(tx.MsgTx.TxOut) {
		return nil, errors.New("Output index out of range")
	}
	return bitcoin.ScriptTemplateFromLockingScript(tx.MsgTx.TxOut[index].PkScript)
}

// Serialize returns the byte payload of the transaction.
func (tx *TxBuilder) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	if err := tx.MsgTx.Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
