package request

import (
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcutil"
)

func TestSendHandler_handle(t *testing.T) {
	ctx := newSilentContext()

	hash := newHash("82b1576993052733ca685419ca4be32cde1e6f7c772e839cd76cd931537222b8")

	contractAddr := "1DNTgNSWtTestKs7j1DwaoxmSc4q9sEUsb"

	issuerAddr := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"
	receiverAddr := "123h2RL1DT4AuYyJUseGxcXSAe5imPSeLV"

	asset := Asset{
		ID:  "foo",
		Qty: 20,
		Holdings: map[string]Holding{
			issuerAddr: Holding{
				Address: issuerAddr,
				Balance: 20,
			},
		},
	}
	contract := Contract{
		ID:            contractAddr,
		IssuerAddress: issuerAddr,
		Assets: map[string]Asset{
			asset.ID: asset,
		},
	}

	issue := protocol.NewSend()
	issue.AssetID = []byte(asset.ID)
	issue.AssetType = []byte("RRE")
	issue.TokenQty = 1

	senders := []btcutil.Address{
		decodeAddress(issuerAddr),
	}

	// the receiver at index 1 is the address we are issuing assets to.
	receivers := []txbuilder.TxOutput{
		txbuilder.TxOutput{},
		txbuilder.TxOutput{
			Address: decodeAddress(receiverAddr),
		},
	}

	req := contractRequest{
		hash:      hash,
		contract:  contract,
		senders:   senders,
		receivers: receivers,
		m:         &issue,
	}

	config := newTestConfig()

	h := newSendHandler(config.Fee)
	resp, err := h.handle(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	// check the settlement message
	settlement, ok := resp.Message.(*protocol.Settlement)
	if !ok {
		t.Fatalf("Could not asset as *Settlement : %#+v\n", resp.Message)
	}

	// clear out the timestamps
	c := resp.Contract

	for k, a := range c.Assets {
		a.CreatedAt = 0
		c.Assets[k] = a

		for hk, h := range a.Holdings {
			h.CreatedAt = 0
			c.Assets[k].Holdings[hk] = h
		}
	}

	wantSettlement := protocol.NewSettlement()
	wantSettlement.AssetType = issue.AssetType
	wantSettlement.AssetID = issue.AssetID
	wantSettlement.Party1TokenQty = 19
	wantSettlement.Party2TokenQty = 1
	wantSettlement.RefTxnIDHash = hashToBytes(hash)

	if !reflect.DeepEqual(settlement, &wantSettlement) {
		t.Fatalf("got\n%+v\nwant\n%+v", settlement, &wantSettlement)
	}

	// check contract assets
	wantAssets := map[string]Asset{
		"foo": Asset{
			ID:  "foo",
			Qty: 20,
			Holdings: map[string]Holding{
				issuerAddr: Holding{
					Address: issuerAddr,
					Balance: 19,
				},
				receiverAddr: Holding{
					Address: receiverAddr,
					Balance: 1,
				},
			},
		},
	}

	gotAssets := resp.Contract.Assets
	if !reflect.DeepEqual(gotAssets, wantAssets) {
		t.Fatalf("got\n%+v\nwant\n%+v", gotAssets, wantAssets)
	}
}

func TestSendHandler_broken(t *testing.T) {
	t.Skip("Skipping until new fixtures exist")

	txHash := "7c2ec1581cd3bf519fda40be3e03dcf6752dc8d43c62736ad5336b4d22299046"
	tx := loadFixtureTX(txHash)
	m := loadMessageFromTX(tx)

	asset := Asset{
		ID:                 "mt03807bt2imhmfni8er7wmcjagvh5gx",
		Qty:                20,
		CreatedAt:          1534142942649617992,
		AuthorizationFlags: []byte("/w=="),
		Type:               "COU",
		Holdings: map[string]Holding{
			"18jotoFFwVeZXkXXPYsngo4Y96acvWy9sA": Holding{
				Balance:   1,
				Address:   "18jotoFFwVeZXkXXPYsngo4Y96acvWy9sA",
				CreatedAt: 1534143076060911381,
			},
			"19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC": Holding{
				Balance:   19,
				Address:   "19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC",
				CreatedAt: 1534142942649616893,
			},
		},
	}

	contract := Contract{
		ID:            "1H6h6PQ4hs9SeV2EDqzKawGPJPWa9L8ZmJ",
		IssuerAddress: "19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC",
		Assets: map[string]Asset{
			asset.ID: asset,
		},
	}

	senders := []btcutil.Address{
		decodeAddress("19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC"),
	}

	receivers := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: decodeAddress("1H6h6PQ4hs9SeV2EDqzKawGPJPWa9L8ZmJ"),
			Value:   2200,
		},
		txbuilder.TxOutput{
			Address: decodeAddress("18jotoFFwVeZXkXXPYsngo4Y96acvWy9sA"),
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: decodeAddress("19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC"),
			Value:   546,
		},
	}

	req := contractRequest{
		hash:      tx.TxHash(),
		contract:  contract,
		m:         m,
		senders:   senders,
		receivers: receivers,
	}

	ctx := newSilentContext()

	config := newTestConfig()

	h := newSendHandler(config.Fee)
	resp, err := h.handle(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	gotHoldings := resp.Contract.Assets[asset.ID].Holdings

	for k, h := range gotHoldings {
		h.CreatedAt = 0
		gotHoldings[k] = h
	}

	wantHoldings := map[string]Holding{
		"18jotoFFwVeZXkXXPYsngo4Y96acvWy9sA": Holding{
			Address: "18jotoFFwVeZXkXXPYsngo4Y96acvWy9sA",
			Balance: 2,
		},
		"19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC": Holding{
			Address: "19JkfEsUu3AdL6p58LGrVL7jsCSnj2eueC",
			Balance: 18,
		},
	}

	if !reflect.DeepEqual(gotHoldings, wantHoldings) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", gotHoldings, wantHoldings)
	}

	settlement, ok := resp.Message.(*protocol.Settlement)
	if !ok {
		t.Fatal("could not assert as *protocol.Settlement")
	}

	wantSettlement := protocol.NewSettlement()
	wantSettlement.AssetType = []byte(asset.Type)
	wantSettlement.AssetID = []byte(asset.ID)
	wantSettlement.Party1TokenQty = 18
	wantSettlement.Party2TokenQty = 2
	wantSettlement.RefTxnIDHash = hashToBytes(tx.TxHash())

	if !reflect.DeepEqual(settlement, &wantSettlement) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", settlement, &wantSettlement)
	}
}
