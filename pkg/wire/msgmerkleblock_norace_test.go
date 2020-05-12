// Exclude from tests when -race is on because it times out --ce
// +build !race

package wire

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

// TestMerkleBlock tests the MsgMerkleBlock API.
func TestMerkleBlock(t *testing.T) {
	pver := ProtocolVersion

	// Block 1 header.
	prevHash := &blockOne.Header.PrevBlock
	merkleHash := &blockOne.Header.MerkleRoot
	bits := blockOne.Header.Bits
	nonce := blockOne.Header.Nonce
	bh := NewBlockHeader(1, prevHash, merkleHash, bits, nonce)

	// Ensure the command is expected value.
	wantCmd := "merkleblock"
	msg := NewMsgMerkleBlock(bh)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgBlock: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num addresses (varInt) + max allowed addresses.
	// wantPayload := uint32(1000000)
	// maxPayload := msg.MaxPayloadLength(pver)
	// if maxPayload != wantPayload {
	// 	t.Errorf("MaxPayloadLength: wrong max payload length for "+
	// 		"protocol version %d - got %v, want %v", pver,
	// 		maxPayload, wantPayload)
	// }

	// Load maxTxPerBlock hashes
	data := make([]byte, 32)
	for i := 0; i < maxTxPerBlock; i++ {
		rand.Read(data)
		hash, err := bitcoin.NewHash32(data)
		if err != nil {
			t.Errorf("NewHash failed: %v\n", err)
			return
		}

		if err = msg.AddTxHash(hash); err != nil {
			t.Errorf("AddTxHash failed: %v\n", err)
			return
		}
	}

	// Add one more Tx to test failure.
	rand.Read(data)
	hash, err := bitcoin.NewHash32(data)
	if err != nil {
		t.Errorf("NewHash failed: %v\n", err)
		return
	}

	if err = msg.AddTxHash(hash); err == nil {
		t.Errorf("AddTxHash succeeded when it should have failed")
		return
	}

	// Test encode with latest protocol version.
	var buf bytes.Buffer
	err = msg.BtcEncode(&buf, pver)
	if err != nil {
		t.Errorf("encode of MsgMerkleBlock failed %v err <%v>", msg, err)
	}

	// Test decode with latest protocol version.
	readmsg := MsgMerkleBlock{}
	err = readmsg.BtcDecode(&buf, pver)
	if err != nil {
		t.Errorf("decode of MsgMerkleBlock failed [%v] err <%v>", buf, err)
	}

	// Force extra hash to test maxTxPerBlock.
	msg.Hashes = append(msg.Hashes, hash)
	err = msg.BtcEncode(&buf, pver)
	if err == nil {
		t.Errorf("encode of MsgMerkleBlock succeeded with too many " +
			"tx hashes when it should have failed")
		return
	}

	// Force too many flag bytes to test maxFlagsPerMerkleBlock.
	// Reset the number of hashes back to a valid value.
	msg.Hashes = msg.Hashes[len(msg.Hashes)-1:]
	msg.Flags = make([]byte, maxFlagsPerMerkleBlock+1)
	err = msg.BtcEncode(&buf, pver)
	if err == nil {
		t.Errorf("encode of MsgMerkleBlock succeeded with too many " +
			"flag bytes when it should have failed")
		return
	}
}
