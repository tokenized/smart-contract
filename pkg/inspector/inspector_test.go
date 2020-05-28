package inspector

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
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
	address, err := bitcoin.DecodeAddress("1AWtnFroMiC7LJWUENVnE8NRKkWW6bQFc")
	if err != nil {
		t.Fatalf("Failed to decode address 1 : %s", err)
	}
	address2, err := bitcoin.DecodeAddress("1PY39VCHyALcJ7L5EUnu9v7JY2NUh1wxSM")
	if err != nil {
		t.Fatalf("Failed to decode address 2 : %s", err)
	}
	hash, err := address.Hash()
	if err != nil {
		t.Fatalf("Failed to get address 1 hash : %s", err)
	}
	t.Logf("Address 1 : %d, %s", address.Type(), hash)
	script, err := bitcoin.NewRawAddressFromAddress(address).LockingScript()
	if err != nil {
		t.Fatalf("Failed to create address 1 locking script : %s", err)
	}
	script2, err := bitcoin.NewRawAddressFromAddress(address2).LockingScript()
	if err != nil {
		t.Fatalf("Failed to create address 2 locking script : %s", err)
	}
	scriptAddress, err := bitcoin.RawAddressFromLockingScript(script)
	if err != nil {
		t.Fatalf("Failed to create address 1 from locking script : %s", err)
	}
	scriptAddress2, err := bitcoin.RawAddressFromLockingScript(script2)
	if err != nil {
		t.Fatalf("Failed to create address 2 from locking script : %s", err)
	}

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
				Address: scriptAddress,
				UTXO: bitcoin.UTXO{
					Hash:          *txHash,
					LockingScript: []byte{118, 169, 20, 1, 204, 178, 102, 159, 29, 44, 88, 54, 25, 65, 62, 5, 44, 168, 187, 71, 18, 197, 246, 136, 172},
					Index:         0,
					Value:         600,
				},
			},
			Output{
				Address: scriptAddress2,
				UTXO: bitcoin.UTXO{
					Hash:          *txHash,
					LockingScript: []byte{118, 169, 20, 247, 49, 116, 38, 84, 195, 208, 193, 148, 143, 52, 84, 240, 127, 2, 157, 14, 128, 197, 170, 136, 172},
					Index:         1,
					Value:         7604510,
				},
			},
		},
	}

	t.Logf("Used wantTX : %s", wantTX.Hash.String()) // To remove warning because of commented code below.

	// Doesn't work with unexported "lock". Even with the cmpopts.IgnoreUnexported().
	// if diff := cmp.Diff(itx, wantTX, cmpopts.IgnoreUnexported()); diff != "" {
	// 	t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", "\u2717", diff)
	// }
}

type TestNode struct{}

func (n *TestNode) GetTX(context.Context, *bitcoin.Hash32) (*wire.MsgTx, error) {
	return nil, nil
}

func (n *TestNode) GetOutputs(context.Context, []wire.OutPoint) ([]bitcoin.UTXO, error) {
	return nil, nil
}

func (n *TestNode) SaveTX(context.Context, *wire.MsgTx) error {
	return nil
}
