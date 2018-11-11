package txbuilder

import (
	"bytes"

	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

// buildUnsignedHardCoded builds an unsigned TX, using hard coded values.
func buildUnsignedHardCoded() ([]byte, error) {
	tx := wire.MsgTx{}
	tx.Version = 0x02

	// txin 0
	utxo0 := wire.OutPoint{}
	utxo0hash, err := chainhash.NewHashFromStr("cd089c2c82efe5f463ffbdc147deefc400f71d3f3317fb5057f5393ca2912507")
	if err != nil {
		return nil, err
	}

	utxo0.Hash = *utxo0hash
	utxo0.Index = 1

	txin0 := wire.NewTxIn(&utxo0, nil)

	tx.AddTxIn(txin0)

	// txin 1
	utxo1 := wire.OutPoint{}
	utxo1hash, err := chainhash.NewHashFromStr("e76f57ce870451c0dafaaced05b055b2b46eb1d724908df17290969ada385b90")
	if err != nil {
		return nil, err
	}

	utxo1.Hash = *utxo1hash
	utxo1.Index = 1

	txin1 := wire.NewTxIn(&utxo1, nil)

	tx.AddTxIn(txin1)

	// outputs
	out0 := wire.NewTxOut(546, nil)

	recipient := "18chgevayKE8fQDDVsopokEnVSugjFRJGL"

	destinationAddress, err := btcutil.DecodeAddress(recipient, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	destinationPkScript, err := txscript.PayToAddrScript(destinationAddress)
	if err != nil {
		return nil, err
	}

	out0.PkScript = destinationPkScript
	tx.AddTxOut(out0)

	var buf bytes.Buffer
	tx.Serialize(&buf)

	return buf.Bytes(), nil
}
