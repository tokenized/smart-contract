package txbuilder

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

const (
	satsPerUnit = 100000000.0
	satsPerByte = 1.25

	MaxTxFee          = 425
	OutputFeeP2PKH    = 34
	OutputFeeOpReturn = 20
	OutputOpDataFee   = 3
	InputFeeP2PKH     = 148
	BaseTxFee         = 10

	StringP2pk   = "p2pk"
	StringReturn = "return"
)

const SigHashForkID txscript.SigHashType = 0x40
const DustMinimumOutput uint64 = 546

var notEnoughValueError = errors.New("unable to find enough value to spend")

type NetInterface interface {
	GetTX(context.Context, *chainhash.Hash) (*wire.MsgTx, error)
}

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

func ConvertBCHToSatoshis(f float32) uint64 {
	return uint64(f * satsPerUnit)
}
