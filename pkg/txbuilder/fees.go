package txbuilder

import (
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	// P2PKH/P2SH input size 149
	//   Previous Transaction ID = 32 bytes
	//   Previous Transaction Output Index = 4 bytes
	//   script size = 1 byte
	//   Signature push to stack = 74
	//       push size = 1 byte
	//       signature up to = 72 bytes
	//       signature hash type = 1 byte
	//   Public key push to stack = 34
	//       push size = 1 byte
	//       public key size = 33 bytes
	//   Sequence number = 4
	MaximumP2PKHInputSize = 32 + 4 + 1 + 74 + 34 + 4

	// Size of output not including script
	OutputBaseSize = 8

	// P2PKH/P2SH output size 33
	//   amount = 8 bytes
	//   script size = 1 byte
	//   Script (25 bytes) OP_DUP OP_HASH160 <Push Data byte, PUB KEY/SCRIPT HASH (20 bytes)> OP_EQUALVERIFY
	//     OP_CHECKSIG
	P2PKHOutputSize = OutputBaseSize + 26

	// BaseTxFee is the size of the tx not included in inputs and outputs.
	//   Version = 4 bytes
	//   LockTime = 4 bytes
	BaseTxSize = 8
)

// The fee should be estimated before signing, then after signing the fee should be checked.
// If the fee is too low after signing, then the fee should be adjusted and the tx re-signed.

func (tx *TxBuilder) Fee() uint64 {
	o := tx.OutputValue(true)
	i := tx.InputValue()
	if o > i {
		return 0
	}
	return i - o
}

// EstimatedSize returns the estimated size in bytes of the tx after signatures are added.
// It assumes all inputs are P2PKH.
func (tx *TxBuilder) EstimatedSize() int {
	result := BaseTxSize + wire.VarIntSerializeSize(uint64(len(tx.MsgTx.TxIn))) +
		wire.VarIntSerializeSize(uint64(len(tx.MsgTx.TxOut)))

	for _, input := range tx.MsgTx.TxIn {
		if len(input.SignatureScript) > 0 {
			result += input.SerializeSize()
		} else {
			result += MaximumP2PKHInputSize // TODO Update when non-P2PKH inputs are implemented
		}
	}

	for _, output := range tx.MsgTx.TxOut {
		result += output.SerializeSize()
	}

	return result
}

func (tx *TxBuilder) EstimatedFee() uint64 {
	return uint64(float32(tx.EstimatedSize()) * tx.FeeRate)
}

func (tx *TxBuilder) CalculateFee() error {
	_, err := tx.adjustFee(int64(tx.EstimatedFee()) - int64(tx.Fee()))
	return err
}

// InputValue returns the sum of the values of the inputs.
func (tx *TxBuilder) InputValue() uint64 {
	inputValue := uint64(0)
	for _, input := range tx.Inputs {
		inputValue += input.Value
	}
	return inputValue
}

// OutputValue returns the sum of the values of the outputs.
func (tx *TxBuilder) OutputValue(includeChange bool) uint64 {
	outputValue := uint64(0)
	for i, output := range tx.MsgTx.TxOut {
		if includeChange || !tx.Outputs[i].IsRemainder {
			outputValue += uint64(output.Value)
		}
	}
	return outputValue
}

// changeSum returns the sum of the values of the outputs.
func (tx *TxBuilder) changeSum() uint64 {
	changeValue := uint64(0)
	for i, output := range tx.MsgTx.TxOut {
		if tx.Outputs[i].IsRemainder {
			changeValue += uint64(output.Value)
		}
	}
	return changeValue
}

// adjustFee adjusts the tx fee up or down depending on if the amount is negative or positive.
// It returns true if no further fee adjustments should be attempted.
func (tx *TxBuilder) adjustFee(amount int64) (bool, error) {
	if amount == int64(0) {
		return true, nil
	}

	done := false

	// Find change output
	changeOutputIndex := 0xffffffff
	for i, output := range tx.Outputs {
		if output.IsRemainder {
			changeOutputIndex = i
			break
		}
	}

	if amount > int64(0) {
		// Increase fee, transfer from change
		if changeOutputIndex == 0xffffffff {
			return false, newError(ErrorCodeInsufficientValue, fmt.Sprintf("No existing change for tx fee"))
		}

		if tx.MsgTx.TxOut[changeOutputIndex].Value < uint64(amount) {
			return false, newError(ErrorCodeInsufficientValue, fmt.Sprintf("Not enough change for tx fee"))
		}

		// Decrease change, thereby increasing the fee
		tx.MsgTx.TxOut[changeOutputIndex].Value -= uint64(amount)

		// Check if change is below dust
		if tx.MsgTx.TxOut[changeOutputIndex].Value < tx.DustLimit {
			if !tx.Outputs[changeOutputIndex].addedForFee {
				// Don't remove outputs unless they were added by fee adjustment
				return false, newError(ErrorCodeInsufficientValue, fmt.Sprintf("Not enough change for tx fee"))
			}
			// Remove change output since it is less than dust. Dust will go to miner.
			tx.MsgTx.TxOut = append(tx.MsgTx.TxOut[:changeOutputIndex], tx.MsgTx.TxOut[changeOutputIndex+1:]...)
			tx.Outputs = append(tx.Outputs[:changeOutputIndex], tx.Outputs[changeOutputIndex+1:]...)
			done = true
		}
	} else {
		// Decrease fee, transfer to change
		if changeOutputIndex == 0xffffffff {
			// Add a change output if it would be more than the dust limit
			if uint64(-amount) > tx.DustLimit {
				if tx.ChangeAddress.IsEmpty() {
					return false, errors.New("Change address needed")
				}
				err := tx.AddPaymentOutput(tx.ChangeAddress, uint64(-amount), true)
				if err != nil {
					return false, err
				}
				tx.Outputs[len(tx.Outputs)-1].KeyID = tx.ChangeKeyID
				tx.Outputs[len(tx.Outputs)-1].addedForFee = true
			} else {
				done = true
			}
		} else {
			// Increase change, thereby decreasing the fee
			// (amount is negative so subracting it increases the change value)
			tx.MsgTx.TxOut[changeOutputIndex].Value += uint64(-amount)
		}
	}

	return done, nil
}
