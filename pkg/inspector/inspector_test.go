package inspector

import (
	"context"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tokenized/smart-contract/pkg/wire"
)

func TestParseTX(t *testing.T) {
	ctx := context.Background()

	msgTx := loadFixtureTX("2c68cf3e1216acaa1e274dfd3b665b6a9d1d1d252e68d190f9fffc5f7e11fd27")

	itx, err := NewTransactionFromWire(ctx, &msgTx, true)
	if err != nil {
		t.Fatal(err)
	}

	// Parse outputs
	node := TestNode{}
	if err := itx.ParseOutputs(ctx, &node); err != nil {
		t.Fatal(err)
	}

	// the hash of the TX being parsed.
	txHash := newHash("2c68cf3e1216acaa1e274dfd3b665b6a9d1d1d252e68d190f9fffc5f7e11fd27")

	wantTX := &Transaction{
		Hash:  txHash,
		MsgTx: &msgTx,
		// 	Input{
		// 		Address: decodeAddress("13AHjZXrJWj9GjMsFE2X67o4ZSuXPfj35F"),
		// 		Index:   1,
		// 		Value:   7605340,
		// 		UTXO: UTXO{
		// 			Hash:     newHash("46f7140cf1c97ac140562e50532a74286318b9c4714a2245572f4056c10a73e4"),
		// 			PkScript: []byte{118, 169, 20, 23, 177, 246, 194, 98, 68, 113, 18, 20, 254, 231, 21, 14, 90, 107, 155, 48, 128, 193, 52, 136, 172},
		// 			Index:    1,
		// 			Value:    7605340,
		// 		},
		// 	},
		// },
		Outputs: []Output{
			Output{
				Address: decodeAddress("1AWtnFroMiC7LJWUENVnE8NRKkWW6bQFc"),
				Index:   0,
				Value:   600,
				UTXO: UTXO{
					Hash:     txHash,
					PkScript: []byte{118, 169, 20, 1, 204, 178, 102, 159, 29, 44, 88, 54, 25, 65, 62, 5, 44, 168, 187, 71, 18, 197, 246, 136, 172},
					Index:    0,
					Value:    600,
				},
			},
			Output{
				Address: decodeAddress("1PY39VCHyALcJ7L5EUnu9v7JY2NUh1wxSM"),
				Index:   1,
				Value:   7604510,
				UTXO: UTXO{
					Hash:     txHash,
					PkScript: []byte{118, 169, 20, 247, 49, 116, 38, 84, 195, 208, 193, 148, 143, 52, 84, 240, 127, 2, 157, 14, 128, 197, 170, 136, 172},
					Index:    1,
					Value:    7604510,
				},
			},
		},
	}

	ignore := cmpopts.IgnoreUnexported(btcutil.AddressPubKeyHash{})

	if diff := cmp.Diff(itx, wantTX, ignore); diff != "" {
		t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", "\u2717", diff)
	}
}

type TestNode struct{}

func (n *TestNode) GetTX(context.Context, *chainhash.Hash) (*wire.MsgTx, error) {
	return nil, nil
}

func (n *TestNode) GetTXs(context.Context, []*chainhash.Hash) ([]*wire.MsgTx, error) {
	return nil, nil
}

func (n *TestNode) SaveTX(context.Context, *wire.MsgTx) error {
	return nil
}

func (n *TestNode) GetChainParams() *chaincfg.Params {
	return &chaincfg.MainNetParams
}
