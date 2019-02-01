package request

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

func TestHandleAssetDefinition(t *testing.T) {
	t.Skip("Update needed")

	ctx := newSilentContext()

	tx := "48f8ca6e4161480b080dc0c2f382ad8c859dde5b92a6b95c5db64125fc51bc82"
	contractAddr := "18xNWEsexsBoNCUPfPpDAXmcoUwz9jY7aw"
	issuerAddr := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"
	assetID := "apm2qsznhks23z8d83u41s8019hyri3i"

	hash := newHash(tx)
	issuer := decodeAddress(issuerAddr)

	c := contract.Contract{}

	b := loadFixture(fmt.Sprintf("contracts/%s-cf.json", contractAddr))
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}

	m := protocol.NewAssetDefinition()
	m.AssetID = []byte(assetID)
	m.AssetType = []byte("Alf Pog")
	m.Qty = 10

	config := newTestConfig()

	h := newAssetDefinitionHandler(config.Fee)

	senders := []btcutil.Address{
		issuer,
	}

	req := contractRequest{
		hash:     hash,
		senders:  senders,
		contract: c,
		m:        &m,
	}

	// the test
	resp, err := h.handle(ctx, req)
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}

	gotContract := resp.Contract

	zeroTimestamps(&gotContract)

	wantContract := contract.Contract{
		ID:                 contractAddr,
		CreatedAt:          1533079685112123002,
		IssuerAddress:      issuerAddr,
		Revision:           0x0,
		ContractName:       "Discount Cocktail Specials",
		ContractFileHash:   "",
		GoverningLaw:       "",
		Jurisdiction:       "",
		ContractExpiration: 1533288443589,
		URI:                "https://en.wikipedia.org/wiki/Mana_Bar",
		IssuerID:           "Mana Bar",
		ContractOperatorID: "Tokenized",
		AuthorizationFlags: []byte{0x1f, 0xff},
		VotingSystem:       "N",
		Qty:                2,
		Assets: map[string]contract.Asset{
			"apm2qsznhks23z8d83u41s8019hyri3i": contract.Asset{
				ID:                 "apm2qsznhks23z8d83u41s8019hyri3i",
				Type:               "Alf Pog",
				Revision:           0,
				AuthorizationFlags: []byte{},
				VotingSystem:       0,
				VoteMultiplier:     0,
				Qty:                10,
				TxnFeeType:         0,
				TxnFeeCurrency:     "",
				TxnFeeVar:          0,
				TxnFeeFixed:        0,
				Holdings: map[string]contract.Holding{
					issuerAddr: contract.Holding{
						Address: issuerAddr,
						Balance: 10,
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(gotContract, wantContract) {
		t.Errorf("got\n%#+v\nwant\n%#+v", gotContract, wantContract)
	}

	// check the value returned in the resp
	mout := resp.Message

	_, ok := mout.(*protocol.AssetCreation)
	if !ok {
		t.Fatalf("Failed to assert as *protocol.AssetCreation")
	}
}
