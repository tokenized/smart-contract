package bitcoin

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

var (
	curveS256       = btcec.S256()
	curveS256Params = curveS256.Params()
	curveHalfOrder  = new(big.Int).Rsh(curveS256.N, 1)

	ErrBadKeyLength = errors.New("Key has invalid length")

	zeroBigInt big.Int
)

const (
	typeMainPrivKey = 0x80 // Private Key
	typeTestPrivKey = 0xef // Testnet Private Key

	typeIntPrivKey = 0x40
)

var (
	ErrBadKeyType    = errors.New("Key type unknown")
	ErrOutOfRangeKey = errors.New("Out of range key")
)

// Key is an elliptic curve private key using the secp256k1 elliptic curve.
type Key struct {
	value big.Int
	net   Network
}

// KeyFromStr converts WIF (Wallet Import Format) key text to a key.
func KeyFromStr(s string) (Key, error) {
	b, err := decodeAddress(s)
	if err != nil {
		return Key{}, err
	}

	var network Network
	switch b[0] {
	case typeMainPrivKey:
		network = MainNet
	case typeTestPrivKey:
		network = TestNet
	default:
		return Key{}, ErrBadKeyType
	}

	if len(b) == 34 {
		if b[len(b)-1] != 0x01 {
			return Key{}, fmt.Errorf("Key not for compressed public : %x", b[len(b)-1:])
		}
		return KeyFromNumber(b[1:33], network)
	} else if len(b) == 33 {
		return KeyFromNumber(b[1:], network)
	}

	return Key{}, fmt.Errorf("Key unknown format length %d", len(b))
}

// KeyFromBytes decodes a binary bitcoin key. It returns the key and an error if there was an
//   issue.
func KeyFromBytes(b []byte, net Network) (Key, error) {
	if b[0] != typeIntPrivKey {
		return Key{}, ErrBadKeyType
	}
	if err := privateKeyIsValid(b[1:]); err != nil {
		return Key{}, err
	}

	result := Key{net: net}
	result.value.SetBytes(b[1:])
	return result, nil
}

// KeyFromNumber creates a key from a byte representation of a big number.
func KeyFromNumber(b []byte, net Network) (Key, error) {
	if err := privateKeyIsValid(b); err != nil {
		return Key{}, err
	}
	result := Key{net: net}
	result.value.SetBytes(b)
	return result, nil
}

// GenerateKey randomly generates a new key.
func GenerateKey(net Network) (Key, error) {
	key, err := ecdsa.GenerateKey(curveS256, rand.Reader)
	if err != nil {
		return Key{}, err
	}

	return Key{net: net, value: *key.D}, nil
}

// String returns the type followed by the key data with a checksum, encoded with Base58.
func (k Key) String() string {
	var keyType byte

	// Add key type byte in front
	switch k.net {
	case MainNet:
		keyType = typeMainPrivKey
	default:
		keyType = typeTestPrivKey
	}

	b := append([]byte{keyType}, k.value.Bytes()...)
	//b = append(b, 0x01) // compressed public key // Don't know if we want this or not.
	return encodeAddress(b)
}

// Numbers returns the 32 byte values representing the 256 bit big-endian integer of the x and y coordinates.
// Network returns the network id for the key.
func (k Key) Network() Network {
	return k.net
}

// SetString decodes a key from hex text.
func (k *Key) SetString(s string) error {
	nk, err := KeyFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// SetBytes decodes the key from bytes.
func (k *Key) SetBytes(b []byte) error {
	nk, err := KeyFromBytes(b, typeIntPrivKey)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// Bytes returns type followed by the key data.
func (k Key) Bytes() []byte {
	b := k.value.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}

	return append([]byte{typeIntPrivKey}, b...)
}

// Number returns 32 bytes representing the 256 bit big-endian integer of the private key.
func (k Key) Number() []byte {
	b := k.value.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}
	return b
}

// PublicKey returns the public key.
func (k Key) PublicKey() PublicKey {
	x, y := curveS256.ScalarBaseMult(k.value.Bytes())
	return PublicKey{X: *x, Y: *y}
}

// RawAddress returns a raw address for this key.
func (k Key) RawAddress() (RawAddress, error) {
	return k.PublicKey().RawAddress()
}

// IsEmpty returns true if the value is zero.
func (k Key) IsEmpty() bool {
	return k.value.Cmp(&zeroBigInt) == 0
}

// Sign returns the serialized signature of the hash for the private key.
func (k Key) Sign(hash []byte) (Signature, error) {
	return signRFC6979(k.value, hash)
}

// MarshalJSON converts to json.
func (k Key) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *Key) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (k *Key) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("Key db column not bytes")
	}

	c := make([]byte, len(b))
	copy(c, b)
	return k.SetBytes(c)
}

var zeroKeyValue [32]byte

func privateKeyIsValid(b []byte) error {
	// Check for zero private key
	if bytes.Equal(b, zeroKeyValue[:]) {
		return ErrOutOfRangeKey
	}

	// Check for key outside curve
	if bytes.Compare(b, curveS256Params.N.Bytes()) >= 0 {
		return ErrOutOfRangeKey
	}

	return nil
}

func addPrivateKeys(key1 []byte, key2 []byte) []byte {
	var key1Int big.Int
	var key2Int big.Int
	key1Int.SetBytes(key1)
	key2Int.SetBytes(key2)

	key1Int.Add(&key1Int, &key2Int)
	key1Int.Mod(&key1Int, curveS256Params.N)

	b := key1Int.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}
	return b
}
