package identity

import (
	"context"
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

const baseURL = "http://localhost:8081"

func TestRegister(t *testing.T) {
	ctx := context.Background()

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key : %s", err)
	}

	or, err := GetOracle(ctx, baseURL, key)
	if err != nil {
		t.Fatalf("Failed to get oracle : %s", err)
	}

	entity := actions.EntityField{
		Name: "Test",
	}

	xkey, err := bitcoin.GenerateMasterExtendedKey()
	if err != nil {
		t.Fatalf("Failed to generate xkey : %s", err)
	}

	id, err := or.RegisterUser(ctx, entity, []bitcoin.ExtendedKeys{bitcoin.ExtendedKeys{xkey}})
	if err != nil {
		t.Fatalf("Failed to register user : %s", err)
	}

	t.Logf("User ID : %s", id)
}
