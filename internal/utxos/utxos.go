package utxos

import (
	"bytes"
	"errors"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
)

var zeroTxId bitcoin.Hash32

type UTXOs struct {
	list []*UTXO
}

type UTXO struct {
	OutPoint wire.OutPoint
	Output   wire.TxOut
	SpentBy  *bitcoin.Hash32 // Tx Id of transaction that spent utxo
}

// Add adds/spends UTXOs based on the tx.
func (us *UTXOs) Add(tx *wire.MsgTx, addresses []bitcoin.RawAddress) {
	txHash := tx.TxHash()

	// Check for payments to pkh
	for index, output := range tx.TxOut {
		outputAddress, err := bitcoin.RawAddressFromLockingScript(output.LockingScript)
		if err != nil {
			continue
		}

		for _, address := range addresses {
			if address.Equal(outputAddress) {
				found := false

				// Ensure not to duplicate
				for _, existing := range us.list {
					if bytes.Equal(existing.OutPoint.Hash[:], txHash[:]) &&
						existing.OutPoint.Index == uint32(index) {
						found = true
						break
					}
				}

				if !found {
					// Add
					newUTXO := UTXO{
						OutPoint: wire.OutPoint{Hash: *txHash, Index: uint32(index)},
						Output:   *output,
						SpentBy:  &bitcoin.Hash32{},
					}
					us.list = append(us.list, &newUTXO)
				}
			}
		}
	}

	// Check for spends from UTXOs
	for _, input := range tx.TxIn {
		for _, existing := range us.list {
			if bytes.Equal(input.PreviousOutPoint.Hash[:], existing.OutPoint.Hash[:]) &&
				input.PreviousOutPoint.Index == existing.OutPoint.Index {
				existing.SpentBy = txHash
			}
		}
	}
}

// Remove removes UTXOs in the tx from the set.
func (us *UTXOs) Remove(tx *wire.MsgTx, addresses []bitcoin.RawAddress) {
	for index, output := range tx.TxOut {
		outputAddress, err := bitcoin.RawAddressFromLockingScript(output.LockingScript)
		if err != nil {
			continue
		}

		for _, address := range addresses {
			if address.Equal(outputAddress) {
				txHash := tx.TxHash()
				// Ensure not to duplicate
				for i, existing := range us.list {
					if bytes.Equal(existing.OutPoint.Hash[:], txHash[:]) &&
						existing.OutPoint.Index == uint32(index) {
						us.list = append(us.list[:i], us.list[i+1:]...)
					}
				}
			}
		}
	}
}

// Get returns UTXOs (FIFO) totaling at least the specified amount.
func (us *UTXOs) Get(amount uint64, address bitcoin.RawAddress) ([]*UTXO, error) {
	resultAmount := uint64(0)
	result := make([]*UTXO, 0, 5)
	for _, existing := range us.list {
		if bytes.Equal(existing.SpentBy[:], zeroTxId[:]) {
			outputAddress, err := bitcoin.RawAddressFromLockingScript(existing.Output.LockingScript)
			if err != nil || !address.Equal(outputAddress) {
				continue
			}
			result = append(result, existing)
			resultAmount += uint64(existing.Output.Value)
			if resultAmount > amount {
				return result, nil
			}
		}
	}

	return result, errors.New("Not enough funds")
}
