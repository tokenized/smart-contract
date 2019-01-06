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

func TestSendHandler_handle(t *testing.T) {
	ctx := newSilentContext()

	hash := newHash("82b1576993052733ca685419ca4be32cde1e6f7c772e839cd76cd931537222b8")

	contractAddr := "1DNTgNSWtTestKs7j1DwaoxmSc4q9sEUsb"

	issuerAddr := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"
	receiverAddr := "123h2RL1DT4AuYyJUseGxcXSAe5imPSeLV"

	asset := contract.Asset{
		ID:  "foo",
		Qty: 20,
		Holdings: map[string]contract.Holding{
			issuerAddr: contract.Holding{
				Address: issuerAddr,
				Balance: 20,
			},
		},
	}

	c := contract.Contract{
		ID:            contractAddr,
		IssuerAddress: issuerAddr,
		Assets: map[string]contract.Asset{
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
		contract:  c,
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

	// timestamp should be close
	ts := time.Now().Unix()
	if !isCloseTo(int64(settlement.Timestamp), ts, 1) {
		t.Errorf("Expected %v to be close to %v", settlement.Timestamp, ts)
	}

	wantSettlement := protocol.NewSettlement()
	wantSettlement.AssetType = issue.AssetType
	wantSettlement.AssetID = issue.AssetID
	wantSettlement.Party1TokenQty = 19
	wantSettlement.Party2TokenQty = 1

	// we already confirmed timestamp is good
	wantSettlement.Timestamp = settlement.Timestamp

	if !reflect.DeepEqual(settlement, &wantSettlement) {
		t.Fatalf("got\n%+v\nwant\n%+v", settlement, &wantSettlement)
	}
}
