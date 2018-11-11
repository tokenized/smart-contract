package rebuilder

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func TestContractBuilder_Rebuild(t *testing.T) {
	peerRepo := newMockNetworkInterface()

	addr := "15poEvgsAkJu17obc72CEBcLTLAj8gh5uz"
	address, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}

	store := newMockStorage()
	contractState := NewStateService(store)
	txService := NewParser(peerRepo)
	config := newTestConfig()

	cs := NewContractService(contractState, config.Fee)

	s := NewContractBuilder(cs, peerRepo, txService)

	ctx := logger.NewContext()

	c, err := s.Rebuild(ctx, address)
	if err != nil {
		t.Fatal(err)
	}

	b := loadFixture("contract-15poEvgsAkJu17obc72CEBcLTLAj8gh5uz.json")
	expectedContract := Contract{}
	if err := json.Unmarshal(b, &expectedContract); err != nil {
		t.Fatal(err)
	}

	// zero the timestamps
	zeroTimestamps(c)
	zeroTimestamps(&expectedContract)

	if !reflect.DeepEqual(expectedContract, *c) {
		t.Errorf("got\n%#+v\nwant\n%#+v", expectedContract, *c)
	}
}

func TestContractBuilder_Rebuild_15poEvgs(t *testing.T) {
	t.Skip("Skipping until test fixtures available")

	peerRepo := newMockNetworkInterface()

	addr := "15poEvgsAkJu17obc72CEBcLTLAj8gh5uz"
	address, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}

	store := newMockStorage()
	contractState := NewStateService(store)
	txService := NewParser(peerRepo)
	config := newTestConfig()

	cs := NewContractService(contractState, config.Fee)

	s := NewContractBuilder(cs, peerRepo, txService)

	ctx := logger.NewContext()

	c, err := s.Rebuild(ctx, address)
	if err != nil {
		t.Fatal(err)
	}

	b := loadFixture("contract-15poEvgsAkJu17obc72CEBcLTLAj8gh5uz.json")
	expectedContract := Contract{}
	if err := json.Unmarshal(b, &expectedContract); err != nil {
		t.Fatal(err)
	}

	// zero the timestamps
	zeroTimestamps(c)
	zeroTimestamps(&expectedContract)

	if !reflect.DeepEqual(expectedContract, *c) {
		t.Errorf("got\n%#+v\nwant\n%#+v", expectedContract, *c)
	}
}
