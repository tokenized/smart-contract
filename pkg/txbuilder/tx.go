package txbuilder

import (
	"errors"

	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const SigHashForkID txscript.SigHashType = 0x40

type Tx struct {
	MsgTx     wire.MsgTx
	Inputs    []*Input  // Additional Input Data
	Outputs   []*Output // Additional Output Data
	ChangePKH []byte    // The public key hash to pay extra bitcoins to if an output wasn't already specified.
}

// NewTx returns a new Tx with the specified change PKH
// changePKH (Public Key Hash) is a 20 byte slice. i.e. btcutil.Address.ScriptAddress()
func NewTx(changePKH []byte) *Tx {
	result := Tx{
		MsgTx:     wire.MsgTx{Version: wire.TxVersion, LockTime: 0},
		ChangePKH: changePKH,
	}
	return &result
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
func (tx *Tx) AddP2PKHOutput(pkh []byte, value uint64, isChange bool, isDust bool) error {
	output := Output{
		IsChange: isChange,
		IsDust:   isDust,
	}
	tx.Outputs = append(tx.Outputs, &output)

	txout := wire.TxOut{
		Value:    int64(value),
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
