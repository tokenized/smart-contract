package txbuilder

import (
	"errors"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// OutputSupplement contains data that
type OutputSupplement struct {
	IsChange    bool
	IsDust      bool
	addedForFee bool
}

// AddPaymentOutput adds an output to TxBuilder with the specified value and a script paying the
//   specified address.
// isChange marks the output to receive remaining bitcoin.
func (tx *TxBuilder) AddPaymentOutput(address bitcoin.RawAddress, value uint64, isChange bool) error {
	script, err := address.LockingScript()
	if err != nil {
		return err
	}
	return tx.AddOutput(script, value, isChange, false)
}

// AddP2PKHDustOutput adds an output to TxBuilder with the dust limit amount and a script paying the
//   specified address.
// isChange marks the output to receive remaining bitcoin.
// These dust outputs are meant as "notifiers" so that an address will see this transaction and
//   process the data in it. If value is later added to this output, the value replaces the dust
//   limit amount rather than adding to it.
func (tx *TxBuilder) AddDustOutput(address bitcoin.RawAddress, isChange bool) error {
	script, err := address.LockingScript()
	if err != nil {
		return err
	}
	return tx.AddOutput(script, tx.DustLimit, isChange, true)
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
		Value:    value,
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
		tx.MsgTx.TxOut[index].Value = value
	} else {
		tx.MsgTx.TxOut[index].Value += value
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
