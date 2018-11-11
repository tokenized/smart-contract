package request

import (
	"reflect"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/btcsuite/btcutil"
)

func TestExchangeHandler_handle(t *testing.T) {
	ctx := newSilentContext()

	// the addresses we will receive in the outs of the TX
	contractAddr := "1hezzpRJnet3NL38eqwiT6g33J3bS76z9"
	party1Addr := "1J5NEGEfYAqnhzXHkEyWFXre4BBdzjvK1H"
	party2Addr := "1Af3Hu5t7HTwLHxfPcgFLB6E3puzT2Ci9C"

	// receivers from the TX outs
	receivers := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: decodeAddress(contractAddr),
			Value:   8666,
		},
		txbuilder.TxOutput{
			Address: decodeAddress(party1Addr),
			Value:   128803,
		},
		txbuilder.TxOutput{
			Address: decodeAddress(party2Addr),
			Value:   16407,
		},
	}

	// build the existing asset
	asset := Asset{
		ID:   "ebr4e35mwoa1wohklcdmcmg1gbd10otk",
		Qty:  100000000000000,
		Type: "SHC",
		Holdings: map[string]Holding{
			party1Addr: Holding{
				Address: party1Addr,
				Balance: 99999999999000,
			},
		},
	}

	// setup the existing contract
	contract := Contract{
		ID: contractAddr,
		Assets: map[string]Asset{
			asset.ID: asset,
		},
		Qty: 1,
	}

	tx := loadFixtureTX("1a20534bc530128e0522ab43b94a3caa85292582db4502bf7ca97fdef8c1daa8")

	exchange := protocol.NewExchange()
	exchange.Party1AssetType = []byte(asset.Type)
	exchange.Party1AssetID = []byte(asset.ID)
	exchange.Party1TokenQty = 1000
	exchange.OfferValidUntil = uint64(time.Now().Add(time.Hour * 1).UnixNano())

	// the real TX doesn't have this
	exchange.ExchangeFeeFixed = 0.00000555
	exchange.ExchangeFeeAddress = []byte("1HQ2ULuD7T5ykaucZ3KmTo4i29925Qa6ic")

	senders := []btcutil.Address{
		decodeAddress(party1Addr),
		decodeAddress(party2Addr),
	}

	req := contractRequest{
		hash:      tx.TxHash(),
		tx:        &tx,
		contract:  contract,
		senders:   senders,
		receivers: receivers,
		m:         &exchange,
	}

	config := newTestConfig()

	h := newExchangeHandler(config.Fee)

	resp, err := h.handle(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	// check the message
	s, ok := resp.Message.(*protocol.Settlement)
	if !ok {
		t.Fatal("Could not assert message as *protocol.Settlement")
	}

	wantSettlement := protocol.NewSettlement()
	wantSettlement.AssetType = exchange.Party1AssetType
	wantSettlement.AssetID = exchange.Party1AssetID
	wantSettlement.Party1TokenQty = 99999999998000
	wantSettlement.Party2TokenQty = 1000
	wantSettlement.RefTxnIDHash = hashToBytes(tx.TxHash())

	if !reflect.DeepEqual(s, &wantSettlement) {
		t.Fatalf("got\n%+v\nwant\n%+v", s, &wantSettlement)
	}

	// check the asset that was changed
	gotAsset, ok := resp.Contract.Assets[asset.ID]
	if !ok {
		t.Fatalf("Missing asset id %v", asset.ID)

	}

	for hk, h := range gotAsset.Holdings {
		h.CreatedAt = 0
		gotAsset.Holdings[hk] = h
	}

	wantAsset := Asset{
		ID:   asset.ID,
		Type: "SHC",
		Qty:  100000000000000,
		Holdings: map[string]Holding{
			party1Addr: Holding{
				Address: party1Addr,
				Balance: 99999999998000,
			},
			party2Addr: Holding{
				Address: party2Addr,
				Balance: 1000,
			},
		},
	}

	if !reflect.DeepEqual(gotAsset, wantAsset) {
		t.Fatalf("got\n%+v\nwant\n%+v", gotAsset, wantAsset)
	}

	// check the outs, we have an extra to specify
	wantOuts := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: decodeAddress(party1Addr),
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: decodeAddress(party2Addr),
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: decodeAddress(contractAddr),
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: config.Fee.Address,
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: decodeAddress("1HQ2ULuD7T5ykaucZ3KmTo4i29925Qa6ic"),
			Value:   555,
		},
	}

	if !reflect.DeepEqual(resp.outs, wantOuts) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", resp.outs, wantOuts)
	}
}
