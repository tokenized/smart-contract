package txbuilder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	SubSystem = "TxBuilder" // For logger
)

const SigHashForkID txscript.SigHashType = 0x40

type Tx struct {
	MsgTx     *wire.MsgTx
	Inputs    []*Input  // Additional Input Data
	Outputs   []*Output // Additional Output Data
	ChangePKH []byte    // The public key hash to pay extra bitcoins to if an output wasn't already specified.
	DustLimit uint64    // Smallest amount of bitcoin
	FeeRate   float32   // The target fee rate in sat/byte
}

// NewTx returns a new Tx with the specified change PKH
// changePKH (Public Key Hash) is a 20 byte slice. i.e. btcutil.Address.ScriptAddress()
func NewTx(changePKH []byte, dustLimit uint64, feeRate float32) *Tx {
	tx := wire.MsgTx{Version: wire.TxVersion, LockTime: 0}
	result := Tx{
		MsgTx:     &tx,
		ChangePKH: changePKH,
		DustLimit: dustLimit,
		FeeRate:   feeRate,
	}
	return &result
}

func NewTxFromWire(changePKH []byte, dustLimit uint64, feeRate float32, tx *wire.MsgTx, inputs []*wire.MsgTx) (*Tx, error) {
	result := Tx{
		MsgTx:     tx,
		ChangePKH: changePKH,
		DustLimit: dustLimit,
		FeeRate:   feeRate,
	}

	// Setup inputs
	for _, input := range result.MsgTx.TxIn {
		found := false
		for _, inputTx := range inputs {
			txHash := inputTx.TxHash()
			if bytes.Equal(txHash[:], input.PreviousOutPoint.Hash[:]) &&
				int(input.PreviousOutPoint.Index) < len(inputTx.TxOut) {
				// Add input
				newInput := Input{
					PkScript: inputTx.TxOut[input.PreviousOutPoint.Index].PkScript,
					Value:    uint64(inputTx.TxOut[input.PreviousOutPoint.Index].Value),
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
		pkh, err := PubKeyHashFromP2PKH(output.PkScript)
		isChange := err == nil && bytes.Equal(pkh, result.ChangePKH)
		newOutput := Output{
			IsChange: isChange,
			IsDust:   false,
		}
		result.Outputs = append(result.Outputs, &newOutput)
	}

	return &result, nil
}

// AddInput Adds an input to Tx.
//   outpoint references to the output being spent.
//   outputScript is the script from the output being spent.
//   value is the value from the output being spent.
func (tx *Tx) AddInput(outpoint wire.OutPoint, outputScript []byte, value uint64) error {
	if !IsP2PKHScript(outputScript) {
		return NotP2PKHScriptError
	}
	input := Input{
		PkScript: outputScript,
		Value:    value,
	}
	tx.Inputs = append(tx.Inputs, &input)

	txin := wire.TxIn{
		PreviousOutPoint: outpoint,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	tx.MsgTx.AddTxIn(&txin)
	return nil
}

// AddP2PKHOutput adds an output to Tx with the specified value and a P2PKH script paying the
//   specified address.
// pkh (Public Key Hash) is a 20 byte slice. i.e. btcutil.Address.ScriptAddress()
func (tx *Tx) AddP2PKHOutput(pkh []byte, value uint64, isChange bool) error {
	output := Output{
		IsChange: isChange,
		IsDust:   false,
	}
	tx.Outputs = append(tx.Outputs, &output)

	txout := wire.TxOut{
		Value:    int64(value),
		PkScript: P2PKHScriptForPKH(pkh),
	}
	tx.MsgTx.AddTxOut(&txout)
	return nil
}

// AddP2PKHDustOutput adds an output to Tx with the dust limit amount and a P2PKH script paying the
//   specified address.
// These dust outputs are meant as "notifiers" so that a address will see this transaction and
//   process the data in it. If value is later added to this output, the value replaces the dust
//   limit amount rather than adding to it.
// pkh (Public Key Hash) is a 20 byte slice. i.e. btcutil.Address.ScriptAddress()
func (tx *Tx) AddP2PKHDustOutput(pkh []byte, isChange bool) error {
	output := Output{
		IsChange: isChange,
		IsDust:   true,
	}
	tx.Outputs = append(tx.Outputs, &output)

	txout := wire.TxOut{
		Value:    int64(tx.DustLimit),
		PkScript: P2PKHScriptForPKH(pkh),
	}
	tx.MsgTx.AddTxOut(&txout)
	return nil
}

// AddOutput adds an output to Tx with the specified script and value.
func (tx *Tx) AddOutput(script []byte, value uint64, isChange bool, isDust bool) error {
	output := Output{
		IsChange: isChange,
		IsDust:   isDust,
	}
	tx.Outputs = append(tx.Outputs, &output)

	txout := wire.TxOut{
		Value:    int64(value),
		PkScript: script,
	}
	tx.MsgTx.AddTxOut(&txout)
	return nil
}

func (tx *Tx) AddValueToOutput(index uint32, value uint64) error {
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

type Input struct {
	PkScript []byte
	Value    uint64
}

type Output struct {
	IsChange bool
	IsDust   bool
	addedForFee bool
}

// InputAddress returns the address that is paying to the input
func (tx *Tx) InputPKH(index int) ([]byte, error) {
	if index >= len(tx.Inputs) {
		return nil, errors.New("Input index out of range")
	}
	return PubKeyHashFromP2PKH(tx.Inputs[index].PkScript)
}

// OutputAddress returns the address that the output is paying to.
func (tx *Tx) OutputPKH(index int) ([]byte, error) {
	if index >= len(tx.MsgTx.TxOut) {
		return nil, errors.New("Output index out of range")
	}
	return PubKeyHashFromP2PKH(tx.MsgTx.TxOut[index].PkScript)
}
