package txbuilder

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcutil"
)

func TestSign(t *testing.T) {
	secret := "5JhvsapkHeHjy2FiUQYwXh1d74evuMd3rGcKGnifCdFR5G8e6nH"

	// load the WIF
	wif, err := btcutil.DecodeWIF(secret)
	if err != nil {
		t.Fatal(err)
	}

	privateKey := wif.PrivKey

	unsigned := "0200000002072591a23c39f55750fb17333f1df700c4efde47c1bdff63f4e5ef822c9c08cd0100000000ffffffff905b38da9a969072f18d9024d7b16eb4b255b005edacfadac0510487ce576fe70100000000ffffffff0122020000000000001976a9145389667241a2bb9a8665a463531171e50a7f058588ac00000000"

	raw, err := hex.DecodeString(unsigned)
	if err != nil {
		t.Fatal(err)
	}

	// load the TX containing utxo[0]
	refTx0 := loadFixtureTX("cd089c2c82efe5f463ffbdc147deefc400f71d3f3317fb5057f5393ca2912507.txn")

	// load the TX containing utxo[1]
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

	b, err := sign(privateKey, raw, spendableTxOuts)
	if err != nil {
		t.Fatal(err)
	}

	got := hex.EncodeToString(b)

	want := "0200000002072591a23c39f55750fb17333f1df700c4efde47c1bdff63f4e5ef822c9c08cd010000006a47304402206bc655f75005e5201bffc28512f5b1b18a2d6c5dd58217cfd0fc16b5ecaaf56f022074de9e00fbc0c9e417ea524cd8d776aa33cfab69b2f95d7a4061e4327919e656412103a4f338e1e97da5aa41f3e326809bb28d0d33ba728dc551922b30f7999640d27affffffff905b38da9a969072f18d9024d7b16eb4b255b005edacfadac0510487ce576fe7010000006a473044022073d1efb9899fe7e8d71919f79069949392ba32aaec7803739862fb482c3e39f70220219d628c0c01e2169ee4b1c7712c83d28ba929ff9a4128229d4831257abab5f5412103a4f338e1e97da5aa41f3e326809bb28d0d33ba728dc551922b30f7999640d27affffffff0122020000000000001976a9145389667241a2bb9a8665a463531171e50a7f058588ac00000000"

	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", string(got), want)
	}
}
