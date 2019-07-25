package bitcoin

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestKey(t *testing.T) {
	pk := "619c335025c7f4012e556c2a58b2506e30b8511b53ade95ea316fd8c3286feb9"

	data, err := hex.DecodeString(pk)
	if err != nil {
		t.Fatal(err)
	}

	key, err := KeyS256FromBytes(data, TestNet)
	if err != nil {
		t.Fatal(err)
	}

	wif := "92KuV1Mtf9jTttTrw1yawobsa9uCZGbfpambH8H1Y7KfdDxxc4d"

	if key.String() != wif {
		t.Errorf("WIF encode: got %x, want %x", key.String(), wif)
	}

	reverseKey, err := DecodeKeyString(wif)
	if err != nil {
		t.Fatal(err)
	}

	if reverseKey.Network() != TestNet {
		t.Errorf("Wrong WIF network decoded")
	}

	if !bytes.Equal(reverseKey.Bytes(), key.Bytes()) {
		t.Errorf("WIF decode: got %x, want %x", reverseKey, key)
	}

}
