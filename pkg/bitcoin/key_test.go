package bitcoin

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcutil"
)

func TestKey(t *testing.T) {
	tests := []struct {
		keyText string
		net     Network
		wif     string
		err     error
	}{
		{
			keyText: "619c335025c7f4012e556c2a58b2506e30b8511b53ade95ea316fd8c3286feb9",
			net:     TestNet,
			wif:     "92KuV1Mtf9jTttTrw1yawobsa9uCZGbfpambH8H1Y7KfdDxxc4d",
			err:     nil,
		},
		{
			keyText: "0C28FCA386C7A227600B2FE50B7CAE11EC86D3BF1FBE471BE89827E19D72AA1D",
			net:     MainNet,
			wif:     "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.keyText, func(t *testing.T) {
			data, err := hex.DecodeString(tt.keyText)
			if err != nil {
				t.Fatal(err)
			}

			key, err := KeyS256FromBytes(data, tt.net)
			if err != nil {
				t.Fatal(err)
			}

			if key.String() != tt.wif {
				t.Errorf("WIF encode: got %s, want %s", key.String(), tt.wif)
			}

			extwif, err := btcutil.DecodeWIF(tt.wif)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(extwif.PrivKey.Serialize(), data) {
				t.Errorf("Ext WIF encode: got %x, want %x", extwif.PrivKey.Serialize(), data)
			}

			reverseKey, err := DecodeKeyString(tt.wif)
			if err != nil {
				t.Fatal(err)
			}

			if reverseKey.Network() != tt.net {
				t.Errorf("Wrong WIF network decoded")
			}

			if !bytes.Equal(reverseKey.Bytes(), key.Bytes()) {
				t.Errorf("WIF decode: got %x, want %x", reverseKey, key)
			}
		})
	}

}
