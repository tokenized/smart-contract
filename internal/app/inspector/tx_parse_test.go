package inspector

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestConfiscate_qty(t *testing.T) {
	t.Skip("Skipping until new fixtures exist")

	raw := "010000000190d620cfa0bd180da4a4cf79bdc767da05fad2c829cb355276cb90068f0d1d3a000000006a47304402203d4fd95fb2b69d1e5c94047685c718cddbf6f3f552f9ded7fdb4002a746f147002202d10ea377a94da32707b1e0775b7f5ec9a9123735acb693a445412bcbceaf097412103857917f762f4bc2a46a92dc50eee2966f1708e1e57403ed7dfcd3481e925f512ffffffff0422020000000000001976a914f97529f423157da58bf42ebec2648bd1d934f4c588ac22020000000000001976a91454e17ec7b51f1995fa5aafbf2ec0ae588fc9bf8388acaa020000000000001976a9148f6e421ac0bd857a7ba493b1e5c70818925fcd7188ac00000000000000009a6a4c9700000020453453484337346d6e6577386e616574796d39736f6673663572676f3935386d30376235343a1d0d8f0690cb765235cb29c8d2fa05da67c7bd79cfa4a40d18bda0cf20d6900000000000000001000000000000005c436f6e666973636174652066697800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	tx := loadTXFromHex(raw)

	m := loadMessageFromTX(tx)

	c, ok := m.(*protocol.Confiscation)
	if !ok {
		t.Fatal("could not assert as *protocol.Confiscate")
	}

	wantHeader := []byte{0x6a, 0x4c, 0x97}
	if !reflect.DeepEqual(c.Header, wantHeader) {
		t.Errorf("got %v, want %v", c.Header, wantHeader)
	}

	wantProtocolID := uint32(0x00000020)
	if c.ProtocolID != wantProtocolID {
		t.Errorf("got %v, want %v", c.ProtocolID, wantProtocolID)
	}

	wantActionPrefix := []byte{0x45, 0x34}
	if !reflect.DeepEqual(c.ActionPrefix, wantActionPrefix) {
		t.Errorf("got %v, want %v", c.ActionPrefix, wantActionPrefix)
	}

	wantAssetType := []byte("SHC")
	if !reflect.DeepEqual(c.AssetType, wantAssetType) {
		t.Errorf("got %v, want %v", c.AssetType, wantAssetType)
	}

	wantAssetID := []byte("74mnew8naetym9sofsf5rgo958m07b54")
	if !reflect.DeepEqual(c.AssetID, wantAssetID) {
		t.Errorf("got %v, want %v", c.AssetID, wantAssetID)
	}

	// the tx hash
	txhash := "3a1d0d8f0690cb765235cb29c8d2fa05da67c7bd79cfa4a40d18bda0cf20d690"
	h, _ := chainhash.NewHashFromStr(txhash)

	wantRefTxnID, _ := hex.DecodeString(h.String())

	if !reflect.DeepEqual(c.RefTxnID, wantRefTxnID) {
		t.Errorf("got\n%x\nwant\n%x", c.RefTxnID, wantRefTxnID)
	}

	wantOffenderQty := uint64(1)
	if c.OffendersQty != wantOffenderQty {
		t.Errorf("got %v, want %v", c.OffendersQty, wantOffenderQty)
	}

	wantCustodianQty := uint64(92)
	if c.CustodiansQty != wantCustodianQty {
		t.Errorf("got %v, want %v", c.CustodiansQty, wantCustodianQty)
	}

	wantMessage := "Confiscate fix"
	if string(c.Message) != wantMessage {
		t.Errorf("got %s, want %v", c.Message, wantMessage)
	}
}
