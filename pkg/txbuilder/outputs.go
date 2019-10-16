package txbuilder

import (
	"bytes"
	"errors"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// OutputSupplement contains data that is not contained in a tx message, but that is needed to
//   perform operations on the tx, like fee and change calculation.
type OutputSupplement struct {
	// Used when calculating fees to put remaining input value in.
	IsRemainder bool `json:"is_remainder"`

	// Used as a notification payment, but if value is added, then the previous dust amount isn't
	//   added to the new amount.
	IsDust bool `json:"is_dust"`

	// This output was added by the fee calculation and can be removed by the fee calculation.
	addedForFee bool

	// Optional identifier for external use to track the key needed to spend.
	KeyID string `json:"key_id,omitempty"`
}

// OutputAddress returns the address that the output is paying to.
func (tx *TxBuilder) OutputAddress(index int) (bitcoin.RawAddress, error) {
	if index >= len(tx.MsgTx.TxOut) {
		return bitcoin.RawAddress{}, errors.New("Output index out of range")
	}
	return bitcoin.RawAddressFromLockingScript(tx.MsgTx.TxOut[index].PkScript)
}

func (tx *TxBuilder) SetChangeAddress(address bitcoin.RawAddress, keyID string) error {
	tx.ChangeAddress = address
	tx.ChangeKeyID = keyID

	// Update outputs
	changeScript, err := tx.ChangeAddress.LockingScript()
	if err != nil {
		return err
	}
	for i, output := range tx.MsgTx.TxOut {
		if bytes.Equal(output.PkScript, changeScript) {
			tx.Outputs[i].IsRemainder = true
			tx.Outputs[i].KeyID = keyID
		}
	}

	return nil
}

// AddPaymentOutput adds an output to TxBuilder with the specified value and a script paying the
//   specified address.
// isChange marks the output to receive remaining bitcoin.
func (tx *TxBuilder) AddPaymentOutput(address bitcoin.RawAddress, value uint64, isRemainder bool) error {
	if value < tx.DustLimit {
		return newError(ErrorCodeBelowDustValue, "")
	}
	script, err := address.LockingScript()
	if err != nil {
		return err
	}
	return tx.AddOutput(script, value, isRemainder, false)
}

// AddP2PKHDustOutput adds an output to TxBuilder with the dust limit amount and a script paying the
//   specified address.
// isRemainder marks the output to receive remaining bitcoin.
// These dust outputs are meant as "notifiers" so that an address will see this transaction and
//   process the data in it. If value is later added to this output, the value replaces the dust
//   limit amount rather than adding to it.
func (tx *TxBuilder) AddDustOutput(address bitcoin.RawAddress, isRemainder bool) error {
	script, err := address.LockingScript()
	if err != nil {
		return err
	}
	return tx.AddOutput(script, tx.DustLimit, isRemainder, true)
}

// AddOutput adds an output to TxBuilder with the specified script and value.
// isChange marks the output to receive remaining bitcoin.
// isDust marks the output as a dust amount which will be replaced by any non-dust amount if an
//    amount is added later.
func (tx *TxBuilder) AddOutput(lockScript []byte, value uint64, isRemainder bool, isDust bool) error {
	output := OutputSupplement{
		IsRemainder: isRemainder,
		IsDust:      isDust,
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
		if value < tx.DustLimit {
			return newError(ErrorCodeBelowDustValue, "")
		}
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
