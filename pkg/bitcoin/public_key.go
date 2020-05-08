package bitcoin

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/pkg/errors"
)

const (
	PublicKeyCompressedLength = 33
)

// PublicKey is an elliptic curve public key using the secp256k1 elliptic curve.
type PublicKey struct {
	X, Y big.Int
}

// PublicKeyFromString converts key text to a key.
func PublicKeyFromStr(s string) (PublicKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return PublicKey{}, err
	}

	return PublicKeyFromBytes(b)
}

// PublicKeyFromBytes decodes a binary bitcoin public key. It returns the key and an error if
//   there was an issue.
func PublicKeyFromBytes(b []byte) (PublicKey, error) {
	if len(b) != PublicKeyCompressedLength {
		return PublicKey{}, errors.New("Invalid public key length")
	}

	x, y := expandPublicKey(b)
	if err := publicKeyIsValid(x, y); err != nil {
		return PublicKey{}, err
	}
	return PublicKey{X: x, Y: y}, nil
}

// RawAddress returns a raw address for this key.
func (k PublicKey) RawAddress() (RawAddress, error) {
	return NewRawAddressPKH(Hash160(k.Bytes()))
}

// String returns the key data with a checksum, encoded with Base58.
func (k PublicKey) String() string {
	return hex.EncodeToString(k.Bytes())
}

// SetString decodes a public key from hex text.
func (k *PublicKey) SetString(s string) error {
	nk, err := PublicKeyFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// SetBytes decodes the key from bytes.
func (k *PublicKey) SetBytes(b []byte) error {
	nk, err := PublicKeyFromBytes(b)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// Bytes returns serialized compressed key data.
func (k PublicKey) Bytes() []byte {
	return compressPublicKey(k.X, k.Y)
}

// Numbers returns the 32 byte values representing the 256 bit big-endian integer of the x and y coordinates.
func (k PublicKey) Numbers() ([]byte, []byte) {
	return k.X.Bytes(), k.Y.Bytes()
}

// IsEmpty returns true if the value is zero.
func (k PublicKey) IsEmpty() bool {
	return k.X.Cmp(&zeroBigInt) == 0 && k.Y.Cmp(&zeroBigInt) == 0
}

func (k PublicKey) Equal(o PublicKey) bool {
	return k.X.Cmp(&o.X) == 0 && k.Y.Cmp(&o.Y) == 0
}

func (k PublicKey) Serialize(buf *bytes.Buffer) error {
	if _, err := buf.Write(k.Bytes()); err != nil {
		return err
	}
	return nil
}

func (k *PublicKey) Deserialize(buf *bytes.Reader) error {
	b := make([]byte, PublicKeyCompressedLength)
	if _, err := buf.Read(b); err != nil {
		return err
	}

	return k.SetBytes(b)
}

// MarshalJSON converts to json.
func (k PublicKey) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *PublicKey) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (k *PublicKey) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("Public Key db column not bytes")
	}

	c := make([]byte, len(b))
	copy(c, b)
	return k.SetBytes(c)
}

func compressPublicKey(x big.Int, y big.Int) []byte {
	result := make([]byte, PublicKeyCompressedLength)

	// Header byte is 0x02 for even y value and 0x03 for odd
	result[0] = byte(0x02) + byte(y.Bit(0))

	// Put x at end so it is zero padded in front
	b := x.Bytes()
	offset := PublicKeyCompressedLength - len(b)
	copy(result[offset:], b)

	return result
}

func expandPublicKey(k []byte) (big.Int, big.Int) {
	y := big.Int{}
	x := big.Int{}
	x.SetBytes(k[1:])

	// y^2 = x^3 + ax^2 + b
	// a = 0
	// => y^2 = x^3 + b
	ySq := big.NewInt(0)
	ySq.Exp(&x, big.NewInt(3), nil)
	ySq.Add(ySq, curveS256Params.B)

	y.ModSqrt(ySq, curveS256Params.P)

	Ymod := big.NewInt(0)
	Ymod.Mod(&y, big.NewInt(2))

	signY := uint64(k[0]) - 2
	if signY != Ymod.Uint64() {
		y.Sub(curveS256Params.P, &y)
	}

	return x, y
}

func publicKeyIsValid(x, y big.Int) error {
	if x.Sign() == 0 || y.Sign() == 0 {
		return ErrOutOfRangeKey
	}

	return nil
}

func compressedPublicKeyIsValid(k []byte) error {
	x, y := expandPublicKey(k)

	return publicKeyIsValid(x, y)
}

func addCompressedPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	x, y := curveS256.Add(&x1, &y1, &x2, &y2)
	return compressPublicKey(*x, *y)
}
