package bitcoin

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

type PublicKey interface {
	// String returns the serialized compressed public key with a checksum, encoded with Base58.
	String() string

	// Bytes returns the serialized compressed public key.
	Bytes() []byte

	// Numbers returns the numeric values of the x and y coordinates.
	Numbers() ([]byte, []byte)
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
	pubkey, err := btcec.ParsePubKey(key, curveS256)
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

// Numbers returns the 32 byte values representing the 256 bit big-endian integer of the x and y coordinates.
func (k *PublicKeyS256) Numbers() ([]byte, []byte) {
	return k.key.X.Bytes(), k.key.Y.Bytes()
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	result := make([]byte, 32)

	// Header byte is 0x02 for even y value and 0x03 for odd
	result[0] = byte(0x02) + byte(y.Bit(0))

	// Put x at end so it is zero padded in front
	b := x.Bytes()
	offset := 32 - len(b)
	copy(result[offset:], b)

	return result
}

func expandPublicKey(k []byte) (*big.Int, *big.Int) {
	y := big.NewInt(0)
	x := big.NewInt(0)
	x.SetBytes(k[1:])

	// y^2 = x^3 + ax^2 + b
	// a = 0
	// => y^2 = x^3 + b
	ySq := big.NewInt(0)
	ySq.Exp(x, big.NewInt(3), nil)
	ySq.Add(ySq, curveS256Params.B)

	y.ModSqrt(ySq, curveS256Params.P)

	Ymod := big.NewInt(0)
	Ymod.Mod(y, big.NewInt(2))

	signY := uint64(k[0]) - 2
	if signY != Ymod.Uint64() {
		y.Sub(curveS256Params.P, y)
	}

	return x, y
}

func publicKeyIsValid(k []byte) error {
	x, y := expandPublicKey(k)

	if x.Sign() == 0 || y.Sign() == 0 {
		return errors.New("invalid public key")
	}

	return nil
}

func addPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	return compressPublicKey(curveS256.Add(x1, y1, x2, y2))
}
