package txbuilder

import (
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
)

const (
	satsPerByte = 1.25
)

type TxBuilder struct {
	PrivateKey *btcec.PrivateKey
}

func NewTxBuilder(privateKey *btcec.PrivateKey) TxBuilder {
	return TxBuilder{
		PrivateKey: privateKey,
	}
}

func (s TxBuilder) Build(utxos UTXOs,
	outs []PayAddress,
	changeAddress btcutil.Address,
	opReturnPayload []byte) (*wire.MsgTx, error) {

	// gather the spendable output details
	spendableTxOuts := make([]*TxOutput, len(utxos), len(utxos))

	for i, utxo := range utxos {
		out := TxOutput{
			PkScript:        utxo.PkScript,
			Value:           utxo.Value,
			TransactionHash: utxo.Hash.CloneBytes(),
			Index:           uint32(utxo.Index),
		}

		spendableTxOuts[i] = &out
	}

	// outputs
	outputs := []TxOutput{}

	// any other outs to add?
	for _, o := range outs {
		if o.Value == 0 {
			continue
		}

		out := TxOutput{
			Address: o.Address,
			Value:   o.Value,
			Type:    OutputTypeP2PK,
		}

		outputs = append(outputs, out)
	}

	// get the actual payload from the OP_RETURN
	startIndex := 3
	if opReturnPayload[1] < 0x4c {
		startIndex = 2
	}

	data := opReturnPayload[startIndex:]
	opReturn := TxOutput{
		Type: OutputTypeReturn,
		Data: data,
	}

	// Build the TX.
	//
	// The OP_RETURN will be added at the end of all outputs, including any
	// change that will be calculated.
	tx, err := build(spendableTxOuts, outputs, s.PrivateKey, changeAddress, opReturn)
	if err != nil {
		return nil, err
	}

	return tx.MsgTx, nil
}
