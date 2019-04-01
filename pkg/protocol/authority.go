package protocol

import (
	"context"
	"crypto/sha256"

	"github.com/pkg/errors"
)

// OrderAuthoritySigHash returns a Double SHA256 of the data in an order op return that needs to
//   be signed by an authority for the OrderSignature field.
// The Order must be fully populated including the AuthorityName.
func OrderAuthoritySigHash(ctx context.Context, contractPKH *PublicKeyHash, order *Order) ([]byte, error) {

	// Calculate the hash
	digest := sha256.New()

	digest.Write(contractPKH.Bytes())
	digest.Write([]byte{order.ComplianceAction})
	digest.Write([]byte(order.AuthorityName))
	digest.Write(order.AssetCode.Bytes())
	digest.Write(order.SupportingEvidenceHash[:])

	data, err := order.FreezePeriod.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to serialize freeze timeout")
	}
	digest.Write(data)

	for _, target := range order.TargetAddresses {
		data, err := target.Serialize()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to serialize target address")
		}
		digest.Write(data)
	}

	digest.Write(order.DepositAddress.Bytes())

	hash := sha256.Sum256(digest.Sum(nil))
	return hash[:], nil
}
