package identity

import "github.com/tokenized/pkg/bitcoin"

// ApprovedEntityPublicKey is the data returned after approving an entity is associated with a
// public key.
type ApprovedEntityPublicKey struct {
	SigAlgorithm uint32            `json:"algorithm"`
	Signature    bitcoin.Signature `json:"signature"`
	BlockHeight  uint32            `json:"block_height"`
	PublicKey    bitcoin.PublicKey `json:"public_key"`
}
