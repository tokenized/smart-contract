package bitcoin

import (
	"crypto/rand"
	"testing"
)

func TestWP42(t *testing.T) {
	var hash Hash32
	for i := 0; i < 10; i++ {
		_, err := rand.Read(hash[:])
		if err != nil {
			t.Errorf("Failed to generate random hash")
		}

		key, err := GenerateKey(MainNet)
		if err != nil {
			t.Errorf("Failed to generate key")
		}
		publicKey := key.PublicKey()

		key2, err := NextKey(key, hash)
		if err != nil {
			t.Errorf("Failed to calculate next key")
		}
		pubKey2 := key2.PublicKey()

		publicKey2, err := NextPublicKey(publicKey, hash)
		if err != nil {
			t.Errorf("Failed to calculate next public key")
		}

		if !pubKey2.Equal(publicKey2) {
			t.Errorf("Public keys not equal : \n%s\n%s", pubKey2.String(), publicKey2.String())
		}

		// t.Logf("Generated Key : %s", key.String())
		// t.Logf("Generated Public Key : %s", publicKey.String())
		// t.Logf("Next Key : %s", key2.String())
		// t.Logf("Next Public Key : %s", pubKey2.String())
	}
}
