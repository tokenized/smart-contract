package bsvalias

import (
	"crypto/sha256"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const signatureMessagePrefix = "Bitcoin Signed Message:\n"

// SignatureHashForMessage calculates a double SHA256 hash for a message to be used for signing.
// Based on MoneyButton's BSV library.
func SignatureHashForMessage(message string) (bitcoin.Hash32, error) {
	hasher := sha256.New()

	wire.WriteVarInt(hasher, 0, uint64(len(signatureMessagePrefix)))
	hasher.Write([]byte(signatureMessagePrefix))

	wire.WriteVarInt(hasher, 0, uint64(len(message)))
	hasher.Write([]byte(message))

	hash, err := bitcoin.NewHash32(bitcoin.Sha256(hasher.Sum(nil))) // Double SHA256
	return *hash, err
}
