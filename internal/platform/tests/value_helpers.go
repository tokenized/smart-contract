package tests

import (
	"math/rand"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var testHelperRand *rand.Rand

func RandomHash() *chainhash.Hash {
	data := make([]byte, 32)
	for i, _ := range data {
		data[i] = byte(testHelperRand.Intn(256))
	}
	result, _ := chainhash.NewHash(data)
	return result
}

func RandomTxId() *protocol.TxId {
	data := make([]byte, 32)
	for i, _ := range data {
		data[i] = byte(testHelperRand.Intn(256))
	}
	return protocol.TxIdFromBytes(data)
}
