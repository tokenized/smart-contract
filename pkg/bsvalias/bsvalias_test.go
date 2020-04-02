package bsvalias

// import (
// 	"context"
// 	"strings"
// 	"testing"

// 	"github.com/tokenized/smart-contract/pkg/bitcoin"
// )

// Keep commented so we don't hit moneybutton.com on every test run.

var (
	handles = []string{
		"test@localhost:8080",
		"loosethinker@moneybutton.com",
	}
)

// func TestCapabilities(t *testing.T) {
// 	ctx := context.Background()

// 	for _, handle := range handles {
// 		fields := strings.Split(handle, "@")
// 		site, err := GetSite(ctx, fields[1])
// 		if err != nil {
// 			t.Fatalf("Failed to get site : %s", err)
// 		}

// 		t.Logf("Moneybutton.com Site : %+v", site)
// 	}
// }

// func TestIdentity(t *testing.T) {
// 	ctx := context.Background()

// 	for _, handle := range handles {
// 		fields := strings.Split(handle, "@")

// 		id, err := NewIdentity(ctx, handle)
// 		if err != nil {
// 			t.Fatalf("Failed to get identity : %s", err)
// 		}

// 		if id.Alias != fields[0] {
// 			t.Fatalf("Failed to parse alias : %s != %s", id.Alias, fields[0])
// 		}

// 		if id.Hostname != fields[1] {
// 			t.Fatalf("Failed to parse domain : %s != %s", id.Hostname, fields[1])
// 		}

// 		t.Logf("ID : %+v", id)
// 	}
// }

// func TestPublicKey(t *testing.T) {
// 	ctx := context.Background()

// 	for _, handle := range handles {
// 		id, err := NewIdentity(ctx, handle)
// 		if err != nil {
// 			t.Fatalf("Failed to get identity : %s", err)
// 		}

// 		publicKey, err := id.GetPublicKey(ctx)
// 		if err != nil {
// 			t.Fatalf("Failed to get public key : %s", err)
// 		}

// 		t.Logf("Public Key : %s", publicKey.String())
// 	}
// }

// func TestPaymentDestination(t *testing.T) {
// 	ctx := context.Background()

// 	for _, handle := range handles {
// 		id, err := NewIdentity(ctx, handle)
// 		if err != nil {
// 			t.Fatalf("Failed to get identity : %s", err)
// 		}

// 		script, err := id.GetPaymentDestination("John Bitcoin", "john@bitcoin.com", "Test payment", 10000, nil)
// 		if err != nil {
// 			t.Fatalf("Failed to get payment destination : %s", err)
// 		}

// 		ra, err := bitcoin.RawAddressFromLockingScript(script)
// 		if err != nil {
// 			t.Fatalf("Failed to parse locking script : %s", err)
// 		}

// 		ad := bitcoin.NewAddressFromRawAddress(ra, bitcoin.MainNet)
// 		t.Logf("Payment Destination : %s", ad.String())
// 	}
// }
