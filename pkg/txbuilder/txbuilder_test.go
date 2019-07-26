package txbuilder

import (
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestTxBuilder(t *testing.T) {
	key, err := bitcoin.GenerateKeyS256(bitcoin.TestNet)
	if err != nil {
		t.Errorf("Failed to create private key : %s", err)
	}

	pkh := bitcoin.Hash160(key.PublicKey().Bytes())
	address, err := bitcoin.NewScriptTemplatePKH(pkh)
	if err != nil {
		t.Errorf("Failed to create pkh address : %s", err)
	}

	key2, err := bitcoin.GenerateKeyS256(bitcoin.TestNet)
	if err != nil {
		t.Errorf("Failed to create private key 2 : %s", err)
	}

	pkh2 := bitcoin.Hash160(key2.PublicKey().Bytes())
	address2, err := bitcoin.NewScriptTemplatePKH(pkh2)
	if err != nil {
		t.Errorf("Failed to create pkh address 2 : %s", err)
	}

	inputTx := NewTxBuilder(address2, 500, 1.0)
	err = inputTx.AddPaymentOutput(address, 10000, true)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	tx := NewTxBuilder(address, 500, 1.0)

	err = tx.AddInput(wire.OutPoint{Hash: inputTx.MsgTx.TxHash(), Index: 0},
		inputTx.MsgTx.TxOut[0].PkScript, uint64(inputTx.MsgTx.TxOut[0].Value))
	if err != nil {
		t.Errorf("Failed to add input : %s", err)
	}

	err = tx.AddPaymentOutput(address2, 5000, false)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	err = tx.AddDustOutput(address, true)
	if err != nil {
		t.Errorf("Failed to add output : %s", err)
	}

	// Test single valid private key
	err = tx.Sign([]bitcoin.Key{key})
	if err != nil {
		t.Errorf("Failed to sign tx : %s", err)
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test extra private key
	err = tx.Sign([]bitcoin.Key{key, key2})
	if err != nil {
		t.Errorf("Failed to sign tx with both keys : %s", err)
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test wrong private key
	err = tx.Sign([]bitcoin.Key{key2})
	if IsErrorCode(err, ErrorCodeWrongPrivateKey) {
		if err != nil {
			t.Errorf("Failed to return wrong private key error : %s", err)
		} else {
			t.Errorf("Failed to return wrong private key error")
		}
	}
	t.Logf("Tx Fee : %d", tx.Fee())

	// Test bad PkScript
	txMalformed := NewTxBuilder(address, 500, 1.0)
	err = txMalformed.AddInput(wire.OutPoint{Hash: inputTx.MsgTx.TxHash(), Index: 0},
		append(inputTx.MsgTx.TxOut[0].PkScript, 5), uint64(inputTx.MsgTx.TxOut[0].Value))
	if IsErrorCode(err, ErrorCodeWrongScriptTemplate) {
		if err != nil {
			t.Errorf("Failed to return \"Not P2PKH Script\" error : %s", err)
		} else {
			t.Errorf("Failed to return \"Not P2PKH Script\" error")
		}
	}
}

func TestTxBuilderSample(t *testing.T) {
	// Load your private key
	wif := "cQDgbH4C7HP3LSJevMSb1dPMCviCPoLwJ28mxnDRJueMSCa72xjm"
	key, err := bitcoin.DecodeKeyString(wif)
	if err != nil {
		t.Fatalf("Failed to decode key : %s", err)
	}

	// Decode an address to use for "change".
	// Middle return parameter is the network detected. This should be checked to ensure the address
	//   was encoded for the currently specified network.
	changeAddress, _ := bitcoin.DecodeAddress("mq4htwkZSAG9isuVbEvcLaAiNL59p26W64")

	// Create an instance of the TxBuilder using 512 as the dust limit and 1.1 sat/byte fee rate.
	builder := NewTxBuilder(changeAddress, 512, 1.1)

	// Add an input
	// To spend an input you need the txid, output index, and the locking script and value from that output.
	hash, _ := chainhash.NewHashFromStr("c762a29a4beb4821ad843590c3f11ffaed38b7eadc74557bdf36da3539921531")
	index := uint32(0)
	value := uint64(2000)
	spendAddress, _ := bitcoin.DecodeAddress("mupiWN44gq3NZmvZuMMyx8KbRwism69Gbw")
	_ = builder.AddInput(*wire.NewOutPoint(hash, index), spendAddress.LockingScript(), value)

	// add an output to the recipient
	paymentAddress, _ := bitcoin.DecodeAddress("n1kBjpqmH82jgiRnEHLmFMNv77kvugBomm")
	_ = builder.AddPaymentOutput(paymentAddress, 1000, false)

	// sign the first and only input
	_ = builder.Sign([]bitcoin.Key{key})

	// get the raw TX bytes
	_, _ = builder.Serialize()
}
