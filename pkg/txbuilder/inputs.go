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
}

// AddInput adds an input to TxBuilder.
func (tx *TxBuilder) AddInputUTXO(utxo bitcoin.UTXO) error {
	input := InputSupplement{
		LockingScript: utxo.LockingScript,
		Value:         utxo.Value,
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
func (tx *TxBuilder) AddFunding(utxos []bitcoin.UTXO) error {
	var err error

	inputValue := tx.InputValue()
	outputValue := tx.OutputValue(false)
	feeValue := tx.EstimatedFee()

	if inputValue > outputValue+feeValue {
		return nil // Already funded
	}

	// Calculate additional funding needed. Include cost of first added input.
	// TODO Add support for input scripts other than P2PKH.
	funding := uint64(float32(EstimatedP2PKHInputSize)*tx.FeeRate) + feeValue + outputValue - inputValue

	for _, utxo := range utxos {
		err = tx.AddInput(wire.OutPoint{Hash: utxo.Hash, Index: utxo.Index}, utxo.LockingScript, utxo.Value)
		if err != nil {
			return errors.Wrap(err, "adding input")
		}

		if funding > utxo.Value {
			funding -= utxo.Value // More UTXOs required
		} else {
			// Funding complete
			change := utxo.Value - funding
			if change > tx.DustLimit {
				err = tx.AddPaymentOutput(tx.ChangeAddress, change, true)
				if err != nil {
					return errors.Wrap(err, "adding change")
				}
			}
			funding = 0
			break
		}

		// Add cost of next input
		funding += uint64(float32(EstimatedP2PKHInputSize) * tx.FeeRate)
	}

	if funding != 0 {
		available := uint64(0)
		for _, utxo := range utxos {
			available += utxo.Value
		}
		return fmt.Errorf("insufficient funding %d/%d", available, tx.EstimatedFee())
	}

	return nil
}
