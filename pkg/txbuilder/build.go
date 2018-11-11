package txbuilder

import (
	"errors"
	"fmt"
	"sort"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
)

const (
	MaxTxFee          = 425
	OutputFeeP2PKH    = 34
	OutputFeeOpReturn = 20
	OutputOpDataFee   = 3
	InputFeeP2PKH     = 148
	BaseTxFee         = 10
)

const DustMinimumOutput uint64 = 546

const notEnoughValueErrorText = "unable to find enough value to spend"

var notEnoughValueError = errors.New(notEnoughValueErrorText)

func build(spendableTxOuts []*TxOutput,
	outputs []TxOutput,
	privateKey *btcec.PrivateKey,
	changeAddress btcutil.Address,
	opReturn TxOutput) (*Tx, error) {

	sort.Sort(TxOutSortByValue(spendableTxOuts))

	rTx, _, err := buildWithTxOuts(outputs, spendableTxOuts, privateKey, changeAddress, opReturn)
	if err != nil {
		return nil, err
	}
	return rTx, nil
}

func buildWithTxOuts(outputs []TxOutput,
	spendableTxOuts []*TxOutput,
	privateKey *btcec.PrivateKey,
	changeAddress btcutil.Address,
	opReturn TxOutput) (*Tx, []*TxOutput, error) {

	var minInput = uint64(BaseTxFee + InputFeeP2PKH + OutputFeeP2PKH + DustMinimumOutput)

	var spendOutputType TxOutputType
	var txOutsToUse []*TxOutput
	var totalInputValue uint64

	for {
		if len(spendableTxOuts) == 0 {
			return nil, nil, notEnoughValueError
		}
		spendableTxOut := spendableTxOuts[0]
		spendableTxOuts = spendableTxOuts[1:]
		txOutsToUse = append(txOutsToUse, spendableTxOut)

		totalInputValue += spendableTxOut.Value
		if totalInputValue > minInput {
			break
		}
		minInput += InputFeeP2PKH
	}

	var fee = uint64(BaseTxFee+len(txOutsToUse)*InputFeeP2PKH) + OutputFeeP2PKH

	var totalOutputValue uint64
	allOutputs := append(outputs, opReturn)
	for _, spendOutput := range allOutputs {
		totalOutputValue += spendOutput.Value

		outputFee, err := getAppOutputFee(spendOutput)
		if err != nil {
			return nil, nil, err
		}
		fee += uint64(outputFee)
	}

	// minimum fee is dust
	if fee < DustMinimumOutput {
		fee = DustMinimumOutput
	}

	var change = totalInputValue - fee - totalOutputValue

	pk := PrivateKey{
		Secret: privateKey.Serialize(),
	}

	address, err := pk.GetPublicKey().GetAddress()
	if err != nil {
		return nil, nil, err
	}

	changeAddr, err := GetAddressFromString(changeAddress.EncodeAddress())
	if err != nil {
		return nil, nil, err
	}

	if change > 0 {
		// do we have an output for this address already?
		output := TxOutput{}

		s := changeAddress.EncodeAddress()

		fmt.Printf("change address = %v\n", s)

		for i, o := range outputs {
			if o.Address.EncodeAddress() == s {
				o.Value += change

				output = o
				outputs[i] = o

				break
			}
		}

		fmt.Printf("output = %+v\n", output)
		if output.Value == 0 && change >= DustMinimumOutput {
			output = TxOutput{
				Type:    OutputTypeP2PK,
				Address: changeAddr,
				Value:   change,
			}

			outputs = append(outputs, output)
		}
	}

	// add the OP_RETURN payload last
	outputs = append(outputs, opReturn)

	var tx *wire.MsgTx
	tx, err = Create(txOutsToUse, &pk, outputs)
	if err != nil {
		return nil, nil, err
	}

	var inputs []*TxInput
	for _, txOut := range txOutsToUse {
		inputs = append(inputs, &TxInput{
			PkHash:   txOut.PkHash,
			Value:    txOut.Value,
			PrevHash: txOut.GetHashString(),
		})
	}
	txHash := tx.TxHash()

	var index uint32 = 0
	spendableTxOuts = append([]*TxOutput{{
		TransactionHash: txHash.CloneBytes(),
		PkScript:        tx.TxOut[index].PkScript,
		Index:           index,
		Value:           change,
	}}, spendableTxOuts...)

	return &Tx{
		SelfPkHash: address.ScriptAddress(),
		Type:       spendOutputType,
		MsgTx:      tx,
		Inputs:     inputs,
	}, spendableTxOuts, nil
}

func BuildUnsignedWithTxOuts(outputs []TxOutput,
	spendableTxOuts []*TxOutput,
	address btcutil.Address) (*Tx, []*TxOutput, error) {

	var minInput = uint64(BaseTxFee + InputFeeP2PKH + OutputFeeP2PKH + DustMinimumOutput)

	var spendOutputType TxOutputType
	var txOutsToUse []*TxOutput
	var totalInputValue uint64

	for {
		if len(spendableTxOuts) == 0 {
			return nil, nil, notEnoughValueError
		}
		spendableTxOut := spendableTxOuts[0]
		spendableTxOuts = spendableTxOuts[1:]
		txOutsToUse = append(txOutsToUse, spendableTxOut)

		totalInputValue += spendableTxOut.Value
		if totalInputValue > minInput {
			break
		}
		minInput += InputFeeP2PKH
	}

	var fee = uint64(BaseTxFee+len(txOutsToUse)*InputFeeP2PKH) + OutputFeeP2PKH

	var totalOutputValue uint64

	allOutputs := outputs

	for _, spendOutput := range allOutputs {
		totalOutputValue += spendOutput.Value

		outputFee, err := getAppOutputFee(spendOutput)
		if err != nil {
			return nil, nil, err
		}
		fee += uint64(outputFee)
	}

	var change = totalInputValue - fee - totalOutputValue

	if change < DustMinimumOutput {
		// not enough change to pay, leave it for the miner.
		change = 0
	}

	if change > 0 {
		output := TxOutput{
			Type:    OutputTypeP2PK,
			Address: address,
			Value:   change,
		}

		// change is added after the other P2PK outputs
		outputs = append(outputs, output)
	}

	// add the OP_RETURN payload last
	// outputs = append(outputs, opReturn)

	var tx *wire.MsgTx

	tx, err := CreateUnsigned(txOutsToUse, outputs)

	if err != nil {
		return nil, nil, err
	}

	tx.Version = 0x02

	var inputs []*TxInput
	for _, txOut := range txOutsToUse {
		inputs = append(inputs, &TxInput{
			PkHash:   txOut.PkScript,
			Value:    txOut.Value,
			PrevHash: txOut.GetHashString(),
		})
	}

	txHash := tx.TxHash()

	var index uint32 = 0
	spendableTxOuts = append([]*TxOutput{{
		TransactionHash: txHash.CloneBytes(),
		PkScript:        tx.TxOut[index].PkScript,
		Index:           index,
		Value:           change,
	}}, spendableTxOuts...)

	return &Tx{
		SelfPkHash: address.ScriptAddress(),
		Type:       spendOutputType,
		MsgTx:      tx,
		Inputs:     inputs,
	}, spendableTxOuts, nil
}

func getAppOutputFee(output TxOutput) (uint64, error) {
	switch output.Type {
	case OutputTypeP2PK:
		return OutputFeeP2PKH, nil

	case OutputTypeReturn:
		return uint64(len(output.Data)) + 15, nil
	}

	return 0, errors.New("unable to get fee for output type")
}
