package txbuilder

import (
	"reflect"
	"testing"
)

func TestUTXOSetBuilder_InputValue(t *testing.T) {
	tx := loadFixtureTX("48b1907069c81c3df98f028d0c3a58f195843654929012642cdd062dc8e4c568")

	r := newMockNetworkInterface()
	builder := NewUTXOSetBuilder(r)

	ctx := newSilentContext()
	utxos, err := builder.Build(ctx, &tx)
	if err != nil {
		t.Fatal(err)
	}

	wantUTXOs := UTXOs{
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

	if !reflect.DeepEqual(utxos, wantUTXOs) {
		t.Fatalf("got\n%+v\nwant\n%+v", utxos, wantUTXOs)
	}
}
