package txbuilder

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// UTXOSetBuilder creates new sets of UTXO's.
type UTXOSetBuilder struct {
	Network NetInterface
}

// NewUTXOSetBuilder returns a new UTXOSetBuilder.
func NewUTXOSetBuilder(r NetInterface) UTXOSetBuilder {
	return UTXOSetBuilder{
		Network: r,
	}
}

// Build creates a new UTXOs.
func (b UTXOSetBuilder) Build(tx *wire.MsgTx) (UTXOs, error) {

	ctx := context.Background()
	utxos := UTXOs{}

	for _, txIn := range tx.TxIn {
		previousHash := txIn.PreviousOutPoint.Hash
		n := txIn.PreviousOutPoint.Index

		// get the previous TX as a raw tx
		raw, err := b.Network.GetTX(ctx, &previousHash)
		if err != nil {
			return nil, err
		}

		utxo := NewUTXOFromTX(*raw, n)

		utxos = append(utxos, utxo)

	}

	return utxos, nil
}

func (b UTXOSetBuilder) BuildFromOutputs(tx *wire.MsgTx) (UTXOs, error) {

	utxos := UTXOs{}

	for i, txOut := range tx.TxOut {
		if txOut.Value == 0 {
			continue
		}

		utxo := NewUTXOFromTX(*tx, uint32(i))

		utxos = append(utxos, utxo)
	}

	return utxos, nil
}
