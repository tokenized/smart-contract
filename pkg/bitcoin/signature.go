package bitcoin

import (
	"github.com/btcsuite/btcd/btcec"
)

type Signature interface {
	// String returns the serialized signature with a checksum, encoded with Base58.
	String() string

	// Bytes returns the serialized signature.
	Bytes() []byte

	// Verify returns true if the signature is valid for this public key and hash.
	Verify(hash []byte, pubkey PublicKey) bool
}

// DecodeSignatureString converts key text to a key.
func DecodeSignatureString(s string) (Signature, error) {
	b, err := decodeAddress(s)
	if err != nil {
		return nil, err
	}

	return DecodeSignatureBytes(b)
}

// DecodeSignatureBytes decodes a binary bitcoin signature. It returns the signature and an error if
//   there was an issue.
func DecodeSignatureBytes(b []byte) (Signature, error) {
	result, err := SignatureS256FromBytes(b)
	return result, err
}

/****************************************** S256 **************************************************
/* An elliptic curve signature using the secp256k1 elliptic curve.
*/
type SignatureS256 struct {
	sig *btcec.Signature
}

// SignatureS256FromBytes creates a key from a serialized signature.
func SignatureS256FromBytes(sig []byte) (*SignatureS256, error) {
	signature, err := btcec.ParseSignature(sig, curveS256)
	return &SignatureS256{sig: signature}, err
}

// SignatureS256FromBTCEC creates a key from a btcec signature.
func SignatureS256FromBTCEC(sig *btcec.Signature) *SignatureS256 {
	return &SignatureS256{sig: sig}
}

// String returns the signature data with a checksum, encoded with Base58.
func (s *SignatureS256) String() string {
	return encodeAddress(s.Bytes())
}

// Bytes returns serialized compressed key data.
func (s *SignatureS256) Bytes() []byte {
	return s.sig.Serialize()
}

// Verify returns true if the signature is valid for this public key and hash.
func (s *SignatureS256) Verify(hash []byte, pubkey PublicKey) bool {
	pk, ok := pubkey.(*PublicKeyS256)
	if !ok {
		return false
	}

	return s.sig.Verify(hash, pk.key)
}
