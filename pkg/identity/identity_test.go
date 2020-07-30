package identity

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var (
	urls = []string{
		// "http://localhost:8081", // Manually test by removing comment
	}
)

func TestRegister(t *testing.T) {
	ctx := context.Background()

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key : %s", err)
	}

	for _, url := range urls {
		or, err := GetOracle(ctx, url)
		if err != nil {
			t.Fatalf("Failed to get oracle : %s", err)
		}

		or.SetClientKey(key)

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
}

func TestApproveReceive(t *testing.T) {
	ctx := context.Background()

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key : %s", err)
	}

	for _, url := range urls {
		or, err := GetOracle(ctx, url)
		if err != nil {
			t.Fatalf("Failed to get oracle : %s", err)
		}

		or.SetClientKey(key)

		entity := actions.EntityField{
			Name: "Test",
		}

		xkey, err := bitcoin.GenerateMasterExtendedKey()
		if err != nil {
			t.Fatalf("Failed to generate xkey : %s", err)
		}

		xpubs := bitcoin.ExtendedKeys{xkey}

		userID, err := or.RegisterUser(ctx, entity, []bitcoin.ExtendedKeys{xpubs})
		if err != nil {
			t.Fatalf("Failed to register user : %s", err)
		}

		t.Logf("User ID : %s", userID)

		if err := or.RegisterXPub(ctx, "m/0", xpubs, 1); err != nil {
			t.Fatalf("Failed to register xpub : %s", err)
		}

		contractKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
		if err != nil {
			t.Fatalf("Failed to generate contract key : %s", err)
		}

		contractAddress, err := contractKey.RawAddress()
		contract := bitcoin.NewAddressFromRawAddress(contractAddress, bitcoin.MainNet).String()
		assetCode := protocol.AssetCodeFromContract(contractAddress, 0)
		asset := protocol.AssetID(assets.CodeCurrency, *assetCode)

		receiver, blockHash, err := or.ApproveReceive(ctx, contract, asset, 1, 3, xpubs, 0, 1)
		if err != nil {
			t.Fatalf("Failed to approve receive : %s", err)
		}

		if err := or.ValidateReceiveHash(ctx, blockHash, contract, asset, receiver); err != nil {
			t.Fatalf("Failed to validate receive : %s", err)
		}
	}
}
