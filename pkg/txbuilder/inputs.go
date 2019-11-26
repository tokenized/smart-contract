package txbuilder

import (
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// InputSupplement contains data required to sign an input that is not already in the wire.MsgTx.
type InputSupplement struct {
	LockingScript []byte `json:"locking_script"`
	Value         uint64 `json:"value"`

	// Optional identifier for external use to track the key needed to sign the input.
	KeyID string `json:"key_id,omitempty"`
}

// InputAddress returns the address that is paying to the input.
func (tx *TxBuilder) InputAddress(index int) (bitcoin.RawAddress, error) {
	if index >= len(tx.Inputs) {
		return bitcoin.RawAddress{}, errors.New("Input index out of range")
	}
	return bitcoin.RawAddressFromLockingScript(tx.Inputs[index].LockingScript)
}

// AddInput adds an input to TxBuilder.
func (tx *TxBuilder) AddInputUTXO(utxo bitcoin.UTXO) error {
	// Check that utxo isn't already an input.
	for _, input := range tx.MsgTx.TxIn {
		if input.PreviousOutPoint.Hash.Equal(&utxo.Hash) &&
			input.PreviousOutPoint.Index == utxo.Index {
			return newError(ErrorCodeDuplicateInput, "")
		}
	}

	input := InputSupplement{
		LockingScript: utxo.LockingScript,
		Value:         utxo.Value,
		KeyID:         utxo.KeyID,
	}
	tx.Inputs = append(tx.Inputs, &input)

	txin := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{Hash: utxo.Hash, Index: utxo.Index},
		Sequence:         wire.MaxTxInSequenceNum,
	}
	tx.MsgTx.AddTxIn(&txin)
	return nil
}

// AddInput adds an input to TxBuilder.
//   outpoint reference the output being spent.
//   lockScript is the script from the output being spent.
//   value is the number of satoshis from the output being spent.
func (tx *TxBuilder) AddInput(outpoint wire.OutPoint, lockScript []byte, value uint64) error {
	// Check that outpoint isn't already an input.
	for _, input := range tx.MsgTx.TxIn {
		if input.PreviousOutPoint.Hash.Equal(&outpoint.Hash) &&
			input.PreviousOutPoint.Index == outpoint.Index {
			return newError(ErrorCodeDuplicateInput, "")
		}
	}

	input := InputSupplement{
		LockingScript: lockScript,
		Value:         value,
	}
	tx.Inputs = append(tx.Inputs, &input)

	txin := wire.TxIn{
		PreviousOutPoint: outpoint,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	tx.MsgTx.AddTxIn(&txin)
	return nil
}

// AddFunding adds inputs spending the specified UTXOs until the transaction has enough funding to
//   cover the fees and outputs.
// If SendMax is set then all UTXOs are added as inputs.
func (tx *TxBuilder) AddFunding(utxos []bitcoin.UTXO) error {
	inputValue := tx.InputValue()
	outputValue := tx.OutputValue(true)
	estFeeValue := tx.EstimatedFee()

	if !tx.SendMax && inputValue > outputValue && inputValue-outputValue >= estFeeValue {
		return tx.CalculateFee() // Already funded
	}

	// Calculate additional funding needed. Include cost of first added input.
	// TODO Add support for input scripts other than P2PKH.
	neededFunding := estFeeValue + outputValue - inputValue

	estInputFee := uint64(float32(MaximumP2PKHInputSize) * tx.FeeRate)
	estOutputFee := uint64(float32(P2PKHOutputSize) * tx.FeeRate)

	neededFunding += estInputFee // Add cost of next input

	var err error
	for _, utxo := range utxos {
		err = tx.AddInputUTXO(utxo)
		if err != nil {
			if IsErrorCode(err, ErrorCodeDuplicateInput) {
				continue
			}
			return errors.Wrap(err, "adding input")
		}

		if tx.SendMax {
			continue
		}

		if neededFunding <= utxo.Value {
			// Funding complete
			change := utxo.Value - neededFunding
			if change > tx.DustLimit {
				for i, output := range tx.Outputs {
					if output.IsRemainder {
						// Updating existing "change" output
						tx.MsgTx.TxOut[i].Value += change
						return nil
					}
				}

				if change > tx.DustLimit+estOutputFee {
					// Add new change output
					change -= estOutputFee
					if tx.ChangeAddress.IsEmpty() {
						return errors.New("Change address needed")
					}
					err = tx.AddPaymentOutput(tx.ChangeAddress, change, true)
					if err != nil {
						return errors.Wrap(err, "adding change")
					}
					tx.Outputs[len(tx.Outputs)-1].KeyID = tx.ChangeKeyID
				}
			}

			return nil
		}

		// More UTXOs required
		neededFunding -= utxo.Value
		neededFunding += estInputFee // Add cost of next input
	}

	if tx.SendMax {
		return tx.CalculateFee()
	} else {
		available := uint64(0)
		for _, input := range tx.Inputs {
			available += input.Value
		}
		return newError(ErrorCodeInsufficientValue, fmt.Sprintf("%d/%d", available,
			outputValue+tx.EstimatedFee()))
	}

	return nil
}
