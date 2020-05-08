package bitcoin

import (
	"testing"
)

func TestPK(t *testing.T) {
	key, err := GenerateKey(MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key : %s", err)
	}

	publicKey := key.PublicKey()

	ra, err := NewRawAddressPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to create raw address : %s", err)
	}

	if ra.Type() != ScriptTypePK {
		t.Fatalf("Incorrect script type for raw address : got %d, want %d", ra.Type(), ScriptTypePK)
	}

	pk, err := ra.GetPublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key : %s", err)
	}

	if !pk.Equal(publicKey) {
		t.Fatalf("Incorrect public key for raw address : got %s, want %s", pk.String(),
			publicKey.String())
	}

	script, err := ra.LockingScript()
	if err != nil {
		t.Fatalf("Failed to create locking script : %s", err)
	}

	t.Logf("Locking Script : %x", script)

	raParse, err := RawAddressFromLockingScript(script)
	if err != nil {
		t.Fatalf("Failed to parse locking script : %s", err)
	}

	if !ra.Equal(raParse) {
		t.Fatalf("Incorrect parsed raw address : got %x, want %x", raParse.Bytes(), ra.Bytes())
	}
}
