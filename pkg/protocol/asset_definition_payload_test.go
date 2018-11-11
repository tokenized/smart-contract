package protocol

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestAssetDefinition_Payload(t *testing.T) {
	raw := "6a4cda00000020413100534843333535757a6e7869666a6137616871376a6235376536393873776f683361337500ff4d00000000000000002a0000000000000000000000000000000000000000000000000000505453424c000000000000000000000000426c75652050616e74730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b, err := hex.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}

	m := AssetDefinition{}
	if _, err := m.Write(b); err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	p, err := m.PayloadMessage()
	if err != nil {
		t.Fatal(err)
	}

	want := &AssetTypeShareCommon{
		Ticker:      []byte("PTSBL"),
		Description: []byte("Blue Pants"),
	}

	if !reflect.DeepEqual(p, want) {
		t.Errorf("got\n%+v\nwant\n%+v", p, want)
	}
}
