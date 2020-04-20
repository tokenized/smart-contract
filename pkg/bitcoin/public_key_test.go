package bitcoin

import (
	"encoding/json"
	"testing"
)

func TestPublicKeyMarshal(t *testing.T) {

	key, err := GenerateKey(MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key")
	}

	pubkey := key.PublicKey()

	t.Logf("Public key : %s", pubkey.String())

	req := struct {
		PubKey PublicKey `json:"pubkey"`
	}{
		PubKey: pubkey,
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to json marshal : %s", err)
	}

	var res struct {
		PubKey PublicKey `json:"pubkey"`
	}

	err = json.Unmarshal(b, &res)
	if err != nil {
		t.Fatalf("Failed to json unmarshal : %s", err)
	}

	if !res.PubKey.Equal(pubkey) {
		t.Fatalf("Unmarshalled pub key doesn't match")
	}

	t.Logf("Unmarshalled public key : %s", res.PubKey.String())
}
