package tests

import (
	"math/rand"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var testHelperRand *rand.Rand

func RandomHash() *bitcoin.Hash32 {
	data := make([]byte, 32)
	for i, _ := range data {
		data[i] = byte(testHelperRand.Intn(256))
	}
	result, _ := bitcoin.NewHash32(data)
	return result
}

func RandomTxId() *protocol.TxId {
	data := make([]byte, 32)
	for i, _ := range data {
		data[i] = byte(testHelperRand.Intn(256))
	}
	return protocol.TxIdFromBytes(data)
}
