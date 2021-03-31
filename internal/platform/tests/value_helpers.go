package tests

import (
	"math/rand"

	"github.com/tokenized/pkg/bitcoin"
)

func RandomHash() *bitcoin.Hash32 {
	var hash bitcoin.Hash32
	rand.Read(hash[:])
	return &hash
}

func RandomTxId() *bitcoin.Hash32 {
	return RandomHash()
}
