package request

import (
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

func TestOrderHandler_handle_freeze(t *testing.T) {
	t.Skip("Update needed")

	ctx := newSilentContext()

	hash := newHash("5be5489c453798eaacf410cebdad27e0b40cccc483bd649a52ec7adf00b1d9c2")

	contractAddr := "1hezzpRJnet3NL38eqwiT6g33J3bS76z9"
	contractAddress := decodeAddress(contractAddr)

	issuerAddr := "1J5NEGEfYAqnhzXHkEyWFXre4BBdzjvK1H"
	targetAddr := "1Af3Hu5t7HTwLHxfPcgFLB6E3puzT2Ci9C"

	asset := contract.Asset{
		ID:   "SHCebr4e35mwoa1wohklcdmcmg1gbd10otk",
		Qty:  100000000000000,
		Type: "SHC",
		Holdings: map[string]contract.Holding{
			issuerAddr: contract.Holding{
				Address: issuerAddr,
				Balance: 99999999998000,
			},
			targetAddr: contract.Holding{
				Address: targetAddr,
				Balance: 1000,
			},
		},
	}

	c := contract.Contract{
		ID: contractAddr,
		Assets: map[string]contract.Asset{
			asset.ID: asset,
		},
	}

	order := protocol.NewOrder()
	order.AssetID = []byte(asset.ID)
	order.AssetType = []byte(asset.Type)
	order.ComplianceAction = byte('F')
	order.TargetAddress = []byte(targetAddr)
	order.DepositAddress = []byte(contractAddr)
	order.Message = []byte("Keep calm.")

	senders := []btcutil.Address{
		contractAddress,
	}

	req := contractRequest{
		hash:     hash,
		contract: c,
		senders:  senders,
		m:        &order,
	}

	config := newTestConfig()

	h := newOrderHandler(config.Fee)
	resp, err := h.handle(ctx, req)

	if err != nil {
		t.Fatal(err)
	}

	freeze, ok := resp.Message.(*protocol.Freeze)
	if !ok {
		t.Fatalf("could not assert as *protocol.Freeze")
	}

	// timestamp should be close
	ts := time.Now().Unix()
	if !isCloseTo(int64(freeze.Timestamp), ts, 1) {
		t.Errorf("Expected %v to be close to %v", freeze.Timestamp, ts)
	}

	wantFreeze := protocol.NewFreeze()
	wantFreeze.AssetID = []byte(asset.ID)
	wantFreeze.AssetType = []byte(asset.Type)
	wantFreeze.Message = order.Message

	// we've already confirmed the timestamp is ok
	wantFreeze.Timestamp = freeze.Timestamp

	if !reflect.DeepEqual(*freeze, wantFreeze) {
		t.Fatalf("got\n%+v\nwant\n%+v", *freeze, wantFreeze)
	}

	// check the contract holdings
	holdings := resp.Contract.Assets[asset.ID].Holdings

	// clear out timestamps
	for k, h := range holdings {
		h.CreatedAt = 0
		holdings[k] = h
	}

	wantHoldings := map[string]contract.Holding{
		issuerAddr: contract.Holding{
			Address: issuerAddr,
			Balance: 99999999998000,
		},
		targetAddr: contract.Holding{
			Address: targetAddr,
			Balance: 1000,
			HoldingStatus: &contract.HoldingStatus{
				Code: "F",
			},
		},
	}

	if !reflect.DeepEqual(holdings, wantHoldings) {
		t.Fatalf("got\n%+v\nwant\n%+v", holdings, wantHoldings)
	}

	wantOutputs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: decodeAddress(targetAddr),
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: contractAddress,
			Value:   546, // change will be added to this output later
		},
		txbuilder.TxOutput{
			Address: config.Fee.Address,
			Value:   config.Fee.Value,
		},
	}

	if !reflect.DeepEqual(resp.outs, wantOutputs) {
		t.Fatalf("got\n%+v\nwant\n%+v", resp.outs, wantOutputs)
	}
}

func TestOrderHandler_handle_confiscate(t *testing.T) {
	t.Skip("Update needed")

	ctx := newSilentContext()

	hash := newHash("82b1576993052733ca685419ca4be32cde1e6f7c772e839cd76cd931537222b8")

	contractAddr := "1DNTgNSWtTestKs7j1DwaoxmSc4q9sEUsb"
	contractAddress := decodeAddress(contractAddr)

	depositAddr := "1Cessj8TyzEypaVzp9V8oZhiMLokVDNSR5"
	targetAddr := "1PXGsWY44yAKTcp7H2Y6KPskzG6Kn9rgLA"

	asset := contract.Asset{
		ID:   "foo",
		Qty:  20,
		Type: "GOO",
		Holdings: map[string]contract.Holding{
			depositAddr: contract.Holding{
				Address: depositAddr,
				Balance: 40,
			},
			targetAddr: contract.Holding{
				Address: targetAddr,
				Balance: 12,
			},
		},
	}

	c := contract.Contract{
		ID: contractAddr,
		Assets: map[string]contract.Asset{
			asset.ID: asset,
		},
	}

	order := protocol.NewOrder()
	order.AssetID = []byte(asset.ID)
	order.AssetType = []byte(asset.Type)
	order.ComplianceAction = byte('C')
	order.TargetAddress = []byte(targetAddr)
	order.DepositAddress = []byte(depositAddr)
	order.SupportingEvidenceHash = []byte("b1946ac92492d2347c6235b4d2611184")
	order.Qty = 2
	order.Message = []byte("soz!")

	senders := []btcutil.Address{
		contractAddress,
	}

	req := contractRequest{
		hash:     hash,
		contract: c,
		senders:  senders,
		m:        &order,
	}

	config := newTestConfig()

	h := newOrderHandler(config.Fee)
	resp, err := h.handle(ctx, req)

	if err != nil {
		t.Fatal(err)
	}

	confiscation, ok := resp.Message.(*protocol.Confiscation)
	if !ok {
		t.Fatalf("could not assert as *protocol.Confiscation")
	}

	// timestamp should be close
	ts := time.Now().Unix()
	if !isCloseTo(int64(confiscation.Timestamp), ts, 1) {
		t.Errorf("Expected %v to be close to %v", confiscation.Timestamp, ts)
	}

	wantConfiscation := protocol.NewConfiscation()
	wantConfiscation.AssetID = []byte(asset.ID)
	wantConfiscation.AssetType = []byte(asset.Type)
	wantConfiscation.DepositsQty = 42
	wantConfiscation.TargetsQty = 10
	wantConfiscation.Message = order.Message

	// we've already confirmed the timestamp is ok
	wantConfiscation.Timestamp = confiscation.Timestamp

	if !reflect.DeepEqual(*confiscation, wantConfiscation) {
		t.Fatalf("got\n%+v\nwant\n%+v", *confiscation, wantConfiscation)
	}

	// check the contract holdings
	holdings := resp.Contract.Assets[asset.ID].Holdings

	// clear out timestamps
	for k, h := range holdings {
		h.CreatedAt = 0
		holdings[k] = h
	}

	wantHoldings := map[string]contract.Holding{
		depositAddr: contract.Holding{
			Address: depositAddr,
			Balance: 42,
		},
		targetAddr: contract.Holding{
			Address: targetAddr,
			Balance: 10,
		},
	}

	if !reflect.DeepEqual(holdings, wantHoldings) {
		t.Fatalf("got\n%+v\nwant\n%+v", holdings, wantHoldings)
	}
}
