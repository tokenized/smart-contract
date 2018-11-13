package txbuilder

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func TestUTXO_Address(t *testing.T) {
	raw := "76a9144fd2ffb48fd9717ccefa4fef843740ed4578517d88ac"

	pkScript, err := hex.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}

	hash := newHash("2c2786fe332e94ea61f2a0aef6037cd08bf6495f800a4c829c0f1c07e6104ab8")

	u := UTXO{
		Hash:     hash,
		Index:    0,
		PkScript: pkScript,
		Value:    1100,
	}

	address, err := u.PublicAddress(&chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}

	got := address.EncodeAddress()
	want := "18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUTXOs_InputValue(t *testing.T) {
	// PkScript of the utxo is the vout[index].scriptPubKey.hex value
	//
	// Example.
	//
	// 76a91453bd89833adc6b9a0b9c2a9f40019b9e03bebc0d88ac -> bytes = [118, 169, 20, 83, 189, 137, 131, 58, 220, 107, 154, 11, 156, 42, 159, 64, 1, 155, 158, 3, 190, 188, 13, 136, 172]
	utxos := UTXOs{
		UTXO{
			Hash:     newHash("d78b78b72c3ffc7472c8a70e427c5cb144d40bc9ba966923e4146b98bf83ba27"),
			PkScript: []byte{118, 169, 20, 84, 225, 126, 199, 181, 31, 25, 149, 250, 90, 175, 191, 46, 192, 174, 88, 143, 201, 191, 131, 136, 172},
			Index:    2,
			Value:    107558,
		},
		UTXO{
			Hash:     newHash("fafa6b1d83c6db987e64491f9e6e43ff64701fb62dfb6feb5cfb7968562ab99c"),
			PkScript: []byte{118, 169, 20, 84, 225, 126, 199, 181, 31, 25, 149, 250, 90, 175, 191, 46, 192, 174, 88, 143, 201, 191, 131, 136, 172},
			Index:    0,
			Value:    546,
		},
	}

	want := utxos[0].Value + utxos[1].Value

	got := utxos.Value()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
