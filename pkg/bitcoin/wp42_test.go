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

func BenchmarkWP42Private(b *testing.B) {
	key, err := GenerateKey(MainNet)
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	seed, err := GenerateSeedValue()
	if err != nil {
		b.Fatalf("Failed to generate seed value : %s", err)
	}

	var hash Hash32
	if err := hash.SetBytes(seed.Bytes()); err != nil {
		b.Fatalf("Failed to set hash : %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash = NextHash(hash)

		key, err = NextKey(key, hash)
		if err != nil {
			b.Fatalf("Failed to generate next key : %s", err)
		}
	}
	b.StopTimer()
}

func BenchmarkWP42Public(b *testing.B) {
	key, err := GenerateKey(MainNet)
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	publicKey := key.PublicKey()

	seed, err := GenerateSeedValue()
	if err != nil {
		b.Fatalf("Failed to generate seed value : %s", err)
	}

	var hash Hash32
	if err := hash.SetBytes(seed.Bytes()); err != nil {
		b.Fatalf("Failed to set hash : %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash = NextHash(hash)
		publicKey, err = NextPublicKey(publicKey, hash)
		if err != nil {
			b.Fatalf("Failed to generate next key : %s", err)
		}
	}
	b.StopTimer()
}

func BenchmarkWP42PrivateChild(b *testing.B) {
	key, err := GenerateKey(MainNet)
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	seed, err := GenerateSeedValue()
	if err != nil {
		b.Fatalf("Failed to generate seed value : %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = ChildKey(key, seed, uint64(1000))
		if err != nil {
			b.Fatalf("Failed to generate child key : %s", err)
		}
	}
	b.StopTimer()
}

func BenchmarkWP42PublicChild(b *testing.B) {
	key, err := GenerateKey(MainNet)
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	publicKey := key.PublicKey()

	seed, err := GenerateSeedValue()
	if err != nil {
		b.Fatalf("Failed to generate seed value : %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = ChildPublicKey(publicKey, seed, uint64(1000))
		if err != nil {
			b.Fatalf("Failed to generate child key : %s", err)
		}
	}
	b.StopTimer()
}
