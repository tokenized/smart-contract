package txbuilder

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/txscript"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/ripemd160"
)

func PubKeyHashFromPrivateKey(key *btcec.PrivateKey) []byte {
	hash256 := sha256.New()
	hash160 := ripemd160.New()

	hash256.Write(key.PubKey().SerializeCompressed())
	hash160.Write(hash256.Sum(nil))
	return hash160.Sum(nil)
}

// InputIsSigned returns true if the input at the specified index already has a signature script.
func (tx *Tx) InputIsSigned(index int) bool {
	if index >= len(tx.MsgTx.TxIn) {
		return false
	}

	return len(tx.MsgTx.TxIn[index].SignatureScript) > 0
}

// AllInputsAreSigned returns true if all inputs have a signature script.
func (tx *Tx) AllInputsAreSigned() bool {
	for _, input := range tx.MsgTx.TxIn {
		if len(input.SignatureScript) == 0 {
			return false
		}
	}
	return true
}

// SignInput sets the signature script on the specified input.
// This should only be used when you aren't signing for all inputs and the fee is overestimated, so it needs no adjustement.
func (tx *Tx) SignInput(index int, key *btcec.PrivateKey) error {
	if index >= len(tx.Inputs) {
		return errors.New("Input index out of range")
	}

	pkh, err := PubKeyHashFromP2PKH(tx.Inputs[index].PkScript)
	if err != nil {
		return err
	}

	if !bytes.Equal(pkh, PubKeyHashFromPrivateKey(key)) {
		return newError(ErrorCodeWrongPrivateKey, fmt.Sprintf("Required : %x", pkh))
	}

	tx.MsgTx.TxIn[index].SignatureScript, err = txscript.SignatureScript(tx.MsgTx, index,
		tx.Inputs[index].PkScript, txscript.SigHashAll+SigHashForkID, key, true,
		int64(tx.Inputs[index].Value))
	return err
}

// Sign estimates and updates the fee, signs all inputs, and corrects the fee if necessary.
//   keys is a slice of all keys required to sign all inputs. They do not have to be in any order.
func (tx *Tx) Sign(keys []*btcec.PrivateKey) error {
	// Update fee to estimated amount
	estimatedFee := int64(float32(tx.EstimatedSize()) * tx.FeeRate)
	inputValue := tx.InputValue()
	outputValue := tx.OutputValue(true)

	if inputValue < outputValue+uint64(estimatedFee) {
		return newError(ErrorCodeInsufficientValue, fmt.Sprintf("%d/%d", inputValue,
			outputValue+uint64(estimatedFee)))
	}

	var err error
	done := false

	currentFee := int64(inputValue) - int64(outputValue)
	done, err = tx.adjustFee(estimatedFee - currentFee)
	if err != nil {
		return err
	}

	attempt := 3 // Max of 3 fee adjustment attempts
	var lastError error
	for {
		// Sign all inputs
		for index, _ := range tx.Inputs {
			signed := false
			for _, key := range keys {
				lastError = tx.SignInput(index, key)
				if IsErrorCode(lastError, ErrorCodeWrongPrivateKey) {
					continue
				}
				if lastError != nil {
					return lastError
				}
				signed = true
				break
			}

			if !signed {
				return newError(ErrorCodeMissingPrivateKey, lastError.Error())
			}
		}

		if done || attempt == 0 {
			break
		}

		// Check fee and adjust if too low
		targetFee := int64(float32(tx.MsgTx.SerializeSize()) * tx.FeeRate)
		inputValue = tx.InputValue()
		outputValue = tx.OutputValue(false)
		changeValue := tx.changeSum()
		if inputValue < outputValue+uint64(targetFee) {
			return newError(ErrorCodeInsufficientValue, fmt.Sprintf("%d/%d", inputValue,
				outputValue+uint64(targetFee)))
		}

		currentFee = int64(inputValue) - int64(outputValue) - int64(changeValue)
		if currentFee >= targetFee && float32(currentFee-targetFee)/float32(targetFee) < 0.05 {
			break // Within 10% of target fee
		}

		done, err = tx.adjustFee(targetFee - currentFee)
		if err != nil {
			return err
		}

		attempt--
	}

	return nil
}
