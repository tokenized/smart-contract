package protocol

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// TransferRegisterSigHash returns a Double SHA256 of the data required to identify a
//   token receiver approval by a register.
// The block hash of the chain tip - 1 should be used. The signature will be considered valid
//   until 1 hour past the timestamp of the block after the block hash specified (chain tip).
func TransferRegisterSigHash(ctx context.Context, contractPKH *PublicKeyHash, assetCode *AssetCode,
	receiverPKH *PublicKeyHash, quantity uint64, blockHash *chainhash.Hash) ([]byte, error) {

	// Calculate the hash
	digest := sha256.New()

	digest.Write(contractPKH.Bytes())
	digest.Write(assetCode.Bytes())
	digest.Write(receiverPKH.Bytes())
	binary.Write(digest, DefaultEndian, &quantity)
	digest.Write(blockHash[:])

	hash := sha256.Sum256(digest.Sum(nil))
	return hash[:], nil
}
