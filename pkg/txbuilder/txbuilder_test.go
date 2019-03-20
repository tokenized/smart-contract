package txbuilder

import (
	"crypto/elliptic"
	"testing"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
)

func TestTx(t *testing.T) {
	key, err := btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to create private key : %s", err)
	}

	pkh := PubKeyHashFromPrivateKey(key)

	key2, err := btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to create private key2 : %s", err)
	}

	pkh2 := PubKeyHashFromPrivateKey(key2)

	inputTx := NewTx(pkh2, 500, 1.0)
	err = inputTx.AddP2PKHOutput(pkh, 10000, true)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	tx := NewTx(pkh, 500, 1.0)

	err = tx.AddInput(wire.OutPoint{Hash: inputTx.MsgTx.TxHash(), Index: 0}, inputTx.MsgTx.TxOut[0].PkScript, uint64(inputTx.MsgTx.TxOut[0].Value))
	if err != nil {
		t.Errorf("Failed to add input : %s", err)
	}

	err = tx.AddP2PKHOutput(pkh2, 5000, false)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	err = tx.AddP2PKHDustOutput(pkh, true)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	// Test single valid private key
	err = tx.Sign([]*btcec.PrivateKey{key})
	if err != nil {
		t.Errorf("Failed to sign tx : %s", err)
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test extra private key
	err = tx.Sign([]*btcec.PrivateKey{key, key2})
	if err != nil {
		t.Errorf("Failed to sign tx with both keys : %s", err)
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test wrong private key
	err = tx.Sign([]*btcec.PrivateKey{key2})
	if err != MissingPrivateKeyError {
		if err != nil {
			t.Errorf("Failed to return wrong private key error : %s", err)
		} else {
			t.Errorf("Failed to return wrong private key error")
		}
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test bad PkScript
	txMalformed := NewTx(pkh, 500, 1.0)
	err = txMalformed.AddInput(wire.OutPoint{Hash: inputTx.MsgTx.TxHash(), Index: 0}, append(inputTx.MsgTx.TxOut[0].PkScript, 5), uint64(inputTx.MsgTx.TxOut[0].Value))
	if err != NotP2PKHScriptError {
		if err != nil {
			t.Errorf("Failed to return \"Not P2PKH Script\" error : %s", err)
		} else {
			t.Errorf("Failed to return \"Not P2PKH Script\" error")
		}
	}
}
