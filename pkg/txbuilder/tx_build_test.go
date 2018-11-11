package txbuilder

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestBuildUnsigned(t *testing.T) {
	recipientAddress := "18chgevayKE8fQDDVsopokEnVSugjFRJGL"
	recipient, err := GetAddressFromString(recipientAddress)
	if err != nil {
		t.Fatal(err)
	}

	outputs := []TxOutput{
		TxOutput{
			Address: recipient,
			Value:   546,
			Type:    OutputTypeP2PK,
		},
	}

	refTx0 := loadFixtureTX("cd089c2c82efe5f463ffbdc147deefc400f71d3f3317fb5057f5393ca2912507.txn")

	refTx1 := loadFixtureTX("e76f57ce870451c0dafaaced05b055b2b46eb1d724908df17290969ada385b90.txn")

	refTx0Hash := refTx0.TxHash()
	refTx1Hash := refTx1.TxHash()

	spendableTxOuts := []*TxOutput{
		&TxOutput{
			TransactionHash: refTx0Hash.CloneBytes(),
			PkScript:        refTx0.TxOut[1].PkScript,
			Index:           1,
			Value:           546,
		},
		&TxOutput{
			TransactionHash: refTx1Hash.CloneBytes(),
			PkScript:        refTx1.TxOut[1].PkScript,
			Index:           1,
			Value:           546,
		},
	}

	tx, _, err := BuildUnsignedWithTxOuts(outputs, spendableTxOuts, recipient)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	tx.MsgTx.Serialize(&buf)

	got := hex.EncodeToString(buf.Bytes())

	want := "0200000002072591a23c39f55750fb17333f1df700c4efde47c1bdff63f4e5ef822c9c08cd0100000000ffffffff905b38da9a969072f18d9024d7b16eb4b255b005edacfadac0510487ce576fe70100000000ffffffff0122020000000000001976a9145389667241a2bb9a8665a463531171e50a7f058588ac00000000"

	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", string(got), want)
	}
}
