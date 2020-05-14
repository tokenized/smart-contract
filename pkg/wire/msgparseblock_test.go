package wire

import (
	"bytes"
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

func TestMsgParseBlockOne(t *testing.T) {
	block := &MsgParseBlock{}

	r := bytes.NewReader(blockOneBytes) // from msgblock_test.go
	if err := block.BtcDecode(r, 1); err != nil {
		t.Fatalf("Failed to decode parse block : %s", err)
	}

	count := block.TxCount
	if count != 1 {
		t.Fatalf("Wrong tx count : got %d, want %d", count, 1)
	}

	if !block.IsMerkleRootValid() {
		t.Fatalf("Invalid merkle root hash : \n  got  %s\n  want %s", block.MerkleRoot.String(),
			block.Header.MerkleRoot.String())
	}

	tx, err := block.GetNextTx()
	if err != nil {
		t.Fatalf("Failed to get first tx : %s", err)
	}

	if !tx.TxHash().Equal(&block.MerkleRoot) {
		t.Fatalf("Invalid txid : \n  got  %s\n  want %s", tx.TxHash().String(),
			block.MerkleRoot.String())
	}

	t.Logf("Tx : \n%s", tx.StringWithAddresses(bitcoin.MainNet))

	tx2, err := block.GetNextTx()
	if err != nil {
		t.Fatalf("Failed to get second tx : %s", err)
	}

	if tx2 != nil {
		t.Fatalf("Should not have gotten second tx : %s", tx2.TxHash().String())
	}

	t.Logf("No second tx")
}
