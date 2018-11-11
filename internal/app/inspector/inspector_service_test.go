package inspector

import (
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func TestParser_GetOutputs(t *testing.T) {
	tx := loadFixtureTX("tx_message_chair.txt")

	config := newTestConfig()

	peerRepo := newTestPeerRepo(config)

	// setup the service
	ctx := newSilentContext()
	service := NewParser(peerRepo)

	// run the test!
	got, err := service.GetOutputs(ctx, &tx)
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}

	addr0, err := btcutil.DecodeAddress("1JYbmmZtLsRPxJnfidqwG7e7trNAuwwwmV",
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}

	addr1, err := btcutil.DecodeAddress("13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg",
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}

	want := []Output{
		Output{
			Address: addr0,
			Value:   5000,
		},
		Output{
			Address: addr1,
			Index:   1,
			Value:   78930,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %v", got, want)
	}
}
