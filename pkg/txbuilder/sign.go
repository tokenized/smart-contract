package txbuilder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// InputIsSigned returns true if the input at the specified index already has a signature script.
func (tx *TxBuilder) InputIsSigned(index int) bool {
	if index >= len(tx.MsgTx.TxIn) {
		return false
	}

	return len(tx.MsgTx.TxIn[index].SignatureScript) > 0
}

// AllInputsAreSigned returns true if all inputs have a signature script.
func (tx *TxBuilder) AllInputsAreSigned() bool {
	for _, input := range tx.MsgTx.TxIn {
		if len(input.SignatureScript) == 0 {
			return false
		}
	}
	return true
}

// SignPKHInput sets the signature script on the specified PKH input.
// This should only be used when you aren't signing for all inputs and the fee is overestimated, so
//   it needs no adjustement.
func (tx *TxBuilder) SignInput(index int, key bitcoin.Key, hashCache *SigHashCache) error {
	if index >= len(tx.Inputs) {
		return errors.New("Input index out of range")
	}

	address, err := bitcoin.RawAddressFromLockingScript(tx.Inputs[index].LockScript)
	if err != nil {
		return err
	}

	pkhAddress, ok := address.(*bitcoin.RawAddressPKH)
	if !ok {
		return newError(ErrorCodeWrongScriptTemplate, "Not a P2PKH locking script")
	}

	if !bytes.Equal(pkhAddress.PKH(), bitcoin.Hash160(key.PublicKey().Bytes())) {
		return newError(ErrorCodeWrongPrivateKey, fmt.Sprintf("Required : %x", pkhAddress.PKH()))
	}

	tx.MsgTx.TxIn[index].SignatureScript, err = PKHUnlockingScript(key, tx.MsgTx, index,
		tx.Inputs[index].LockScript, tx.Inputs[index].Value, SigHashAll+SigHashForkID, hashCache)

	return err
}

// Sign estimates and updates the fee, signs all inputs, and corrects the fee if necessary.
//   keys is a slice of all keys required to sign all inputs. They do not have to be in any order.
// TODO Upgrade to sign more than just P2PKH inputs.
func (tx *TxBuilder) Sign(keys []bitcoin.Key) error {
	// Update fee to estimated amount
	estimatedFee := int64(float32(tx.EstimatedSize()) * tx.FeeRate)
	inputValue := tx.InputValue()
	outputValue := tx.OutputValue(true)
	shc := SigHashCache{}
	pkhs := make([][]byte, 0, len(keys))

	for _, key := range keys {
		pkhs = append(pkhs, bitcoin.Hash160(key.PublicKey().Bytes()))
	}

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
	for {
		shc.ClearOutputs()

		// Sign all inputs
		for index, input := range tx.Inputs {
			address, err := bitcoin.RawAddressFromLockingScript(input.LockScript)
			if err != nil {
				return err
			}

			switch a := address.(type) {
			case *bitcoin.RawAddressPKH:
				signed := false
				for i, pkh := range pkhs {
					if !bytes.Equal(pkh, a.PKH()) {
						continue
					}

					tx.MsgTx.TxIn[index].SignatureScript, err = PKHUnlockingScript(keys[i], tx.MsgTx,
						index, tx.Inputs[index].LockScript, tx.Inputs[index].Value,
						SigHashAll+SigHashForkID, &shc)

					if err != nil {
						return err
					}
					signed = true
					break
				}

				if !signed {
					return newError(ErrorCodeMissingPrivateKey, "")
				}

			default:
				return newError(ErrorCodeWrongScriptTemplate, "Not a P2PKH locking script")
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

func PKHUnlockingScript(key bitcoin.Key, tx *wire.MsgTx, index int,
	lockScript []byte, value uint64, hashType SigHashType, hashCache *SigHashCache) ([]byte, error) {
	// <Signature> <PublicKey>
	sig, err := InputSignature(key, tx, index, lockScript, value, hashType, hashCache)
	if err != nil {
		return nil, err
	}

	pubkey := key.PublicKey().Bytes()

	buf := bytes.NewBuffer(make([]byte, 0, len(sig)+len(pubkey)+2))
	err = bitcoin.WritePushDataScript(buf, sig)
	if err != nil {
		return nil, err
	}
	err = bitcoin.WritePushDataScript(buf, pubkey)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SHUnlockingScript(script []byte) ([]byte, error) {
	// <RedeemScript>
	return nil, errors.New("SH Unlocking Script Not Implemented") // TODO Implement SH unlocking script
}

func MultiPKHUnlockingScript(keys [][]byte) ([]byte, error) {
	return nil, errors.New("MultiPKH Unlocking Script Not Implemented") // TODO Implement MultiPKH unlocking script
}

func RPHUnlockingScript(k []byte) ([]byte, error) {
	// <PublicKey> <Signature(containing r)>
	// k is 256 bit number used to calculate sig with r
	return nil, errors.New("RPH Unlocking Script Not Implemented") // TODO Implement RPH unlocking script
}

// InputSignature returns the serialized ECDSA signature for the input index of the specified
//   transaction, with hashType appended to it.
func InputSignature(key bitcoin.Key, tx *wire.MsgTx, index int, lockScript []byte,
	value uint64, hashType SigHashType, hashCache *SigHashCache) ([]byte, error) {

	hash := signatureHash(tx, index, lockScript, value, hashType, hashCache)
	sig, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(sig, byte(hashType)), nil
}
