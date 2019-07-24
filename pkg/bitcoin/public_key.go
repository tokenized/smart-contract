package bitcoin

import (
	"github.com/btcsuite/btcd/btcec"
)

type PublicKey interface {
	// String returns the serialized compressed public key with a checksum, encoded with Base58.
	String() string

	// Bytes returns the serialized compressed public key.
	Bytes() []byte
}

// DecodePublicKeyString converts key text to a key.
func DecodePublicKeyString(s string) (PublicKey, error) {
	b, err := decodeAddress(s)
	if err != nil {
		return nil, err
	}

	return DecodePublicKeyBytes(b)
}

// DecodePublicKeyBytes decodes a binary bitcoin public key. It returns the key and an error if
//   there was an issue.
func DecodePublicKeyBytes(b []byte) (PublicKey, error) {
	result, err := PublicKeyS256FromBytes(b)
	return result, err
}

/****************************************** S256 **************************************************
/* An elliptic curve public key using the secp256k1 elliptic curve.
*/
type PublicKeyS256 struct {
	key *btcec.PublicKey
}

// PublicKeyS256FromBytes creates a key from a serialized compressed public key.
func PublicKeyS256FromBytes(key []byte) (*PublicKeyS256, error) {
	pubkey, err := btcec.ParsePubKey(key, btcec.S256())
	return &PublicKeyS256{key: pubkey}, err
}

// publicKeyS256FromBTCEC creates a key from a btcec public key.
func publicKeyS256FromBTCEC(key *btcec.PublicKey) *PublicKeyS256 {
	return &PublicKeyS256{key: key}
}

// String returns the key data with a checksum, encoded with Base58.
func (k *PublicKeyS256) String() string {
	return encodeAddress(k.Bytes())
}

// Bytes returns serialized compressed key data.
func (k *PublicKeyS256) Bytes() []byte {
	return k.key.SerializeCompressed()
}
