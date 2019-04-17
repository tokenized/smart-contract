package txbuilder

import (
	"fmt"

	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	// P2PKH/P2SH input size 147
	//   Previous Transaction ID = 32 bytes
	//   Previous Transaction Output Index = 4 bytes
	//   script size = 2 bytes
	//   Signature push to stack = 75
	//       push size = 1 byte
	//       signature up to = 73 bytes
	//       signature hash type = 1 byte
	//   Public key push to stack = 34
	//       push size = 1 byte
	//       public key size = 33 bytes
	EstimatedInputSize = 32 + 4 + 2 + 75 + 34

	// Size of output not including script
	OutputBaseSize = 8

	// P2PKH/P2SH output size 33
	//   amount = 8 bytes
	//   script size = 1 byte
	//   Script (24 bytes) OP_DUP OP_HASH160 <PUB KEY/SCRIPT HASH (20 bytes)> OP_EQUALVERIFY
	//     OP_CHECKSIG
	P2PKHOutputSize = OutputBaseSize + 25

	// BaseTxFee is the size of the tx not included in inputs and outputs.
	//   Version = 4 bytes
	//   LockTime = 4 bytes
	BaseTxSize = 8
)

// The fee should be estimated before signing, then after signing the fee should be checked.
// If the fee is too low after signing, then the fee should be adjusted and the tx re-signed.

func (tx *Tx) Fee() uint64 {
	return tx.InputValue() - tx.OutputValue(true)
}

// EstimatedSize returns the estimated size in bytes of the tx after signatures are added.
// It assumes all inputs are P2PKH.
func (tx *Tx) EstimatedSize() int {
	result := BaseTxSize + wire.VarIntSerializeSize(uint64(len(tx.MsgTx.TxIn))) +
		wire.VarIntSerializeSize(uint64(len(tx.MsgTx.TxOut)))

	for _, input := range tx.MsgTx.TxIn {
		if len(input.SignatureScript) > 0 {
			result += input.SerializeSize()
		} else {
			result += EstimatedInputSize
		}
	}

	for _, output := range tx.MsgTx.TxOut {
		result += output.SerializeSize()
	}

	return result
}

func (tx *Tx) EstimatedFee() uint64 {
	return uint64(float32(tx.EstimatedSize()) * tx.FeeRate)
}

// InputValue returns the sum of the values of the inputs.
func (tx *Tx) InputValue() uint64 {
	inputValue := uint64(0)
	for _, input := range tx.Inputs {
		inputValue += input.Value
	}
	return inputValue
}

// OutputValue returns the sum of the values of the outputs.
func (tx *Tx) OutputValue(includeChange bool) uint64 {
	outputValue := uint64(0)
	for i, output := range tx.MsgTx.TxOut {
		if includeChange || !tx.Outputs[i].IsChange {
			outputValue += uint64(output.Value)
		}
	}
	return outputValue
}

// changeSum returns the sum of the values of the outputs.
func (tx *Tx) changeSum() uint64 {
	changeValue := uint64(0)
	for i, output := range tx.MsgTx.TxOut {
		if tx.Outputs[i].IsChange {
			changeValue += uint64(output.Value)
		}
	}
	return changeValue
}

// adjustFee
func (tx *Tx) adjustFee(amount int64) error {
	if amount == int64(0) {
		return nil
	}

	// Find change output
	changeOutputIndex := 0xffffffff
	for i, output := range tx.Outputs {
		if output.IsChange {
			changeOutputIndex = i
			break
		}
	}

	if amount > int64(0) {
		// Increase fee, transfer from change
		if changeOutputIndex == 0xffffffff {
			return newError(ErrorCodeInsufficientValue, fmt.Sprintf("No existing change for tx fee"))
		}

		if tx.MsgTx.TxOut[changeOutputIndex].Value < amount {
			return newError(ErrorCodeInsufficientValue, fmt.Sprintf("Not enough change for tx fee"))
		}

		// Decrease change, thereby increasing the fee
		tx.MsgTx.TxOut[changeOutputIndex].Value -= amount

		// Check if change is below dust
		if uint64(tx.MsgTx.TxOut[changeOutputIndex].Value) < tx.DustLimit {
			if !tx.Outputs[changeOutputIndex].addedForFee {
				// Don't remove outputs unless they were added by fee adjustment
				return newError(ErrorCodeInsufficientValue, fmt.Sprintf("Not enough change for tx fee"))
			}
			// Remove change output since it is less than dust. Dust will go to miner.
			tx.MsgTx.TxOut = append(tx.MsgTx.TxOut[:changeOutputIndex], tx.MsgTx.TxOut[changeOutputIndex+1:]...)
		}
	} else {
		// Decrease fee, transfer to change
		if changeOutputIndex == 0xffffffff {
			// Add a change output if it would be more than the dust limit
			if uint64(-amount) > tx.DustLimit {
				tx.AddP2PKHOutput(tx.ChangePKH, uint64(-amount), true)
				tx.Outputs[len(tx.Outputs)-1].addedForFee = true
			}
			return nil
		}

		// Increase change, thereby decreasing the fee
		// (amount is negative so subracting it increases the change value)
		tx.MsgTx.TxOut[changeOutputIndex].Value -= amount
	}

	return nil
}
