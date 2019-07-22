package bitcoin

import (
	"crypto/elliptic"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	ErrBadKeyLength = errors.New("Key has invalid length")
)

const (
	typePrivKey     = 0x80 // Private Key
	typeTestPrivKey = 0xef // Testnet Private Key
)

type Key interface {
	// String returns the type followed by the key data with a checksum, encoded with Base58.
	String(network wire.BitcoinNet) string

	// Bytes returns type followed by the key data.
	Bytes(network wire.BitcoinNet) []byte

	// PublicKey returns the serialized (compresses) public key data.
	PublicKey() []byte

	// PublicKeyHash returns the Hashs160 of the serialized public key.
	PublicKeyHash() []byte

	// Sign creates a signature from a hash.
	Sign([]byte) ([]byte, error)
}

// DecodeKeyString converts WIF (Wallet Import Format) key text to a key.
func DecodeKeyString(s string) (Key, wire.BitcoinNet, error) {
	b, err := decodeAddress(s)
	if err != nil {
		return nil, MainNet, err
	}

	return DecodeKeyBytes(b)
}

// DecodeKeyBytes decodes a binary bitcoin key (with leading type). It returns the key, the network,
//   and an error if there was an issue.
func DecodeKeyBytes(b []byte) (Key, wire.BitcoinNet, error) {
	var network wire.BitcoinNet
	switch b[0] {
	case typePrivKey:
		network = MainNet
	case typeTestPrivKey:
		network = TestNet
	default:
		return nil, MainNet, ErrBadType
	}

	result, err := KeyS256FromBytes(b[1:])
	return result, network, err
}

/****************************************** S256 **************************************************
/* An elliptic curve private key using the secp256k1 elliptic curve.
*/
type KeyS256 struct {
	key *btcec.PrivateKey
}

// GenerateKeyS256 randomly generates a new key.
func GenerateKeyS256() (*KeyS256, error) {
	privkey, err := btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return nil, err
	}
	return KeyS256FromBytes(privkey.Serialize())
}

// KeyS256FromBytes creates a key from a set of bytes that represents a 256 bit big-endian integer.
func KeyS256FromBytes(key []byte) (*KeyS256, error) {
	privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	return &KeyS256{key: privkey}, nil
}

// String returns the type followed by the key data with a checksum, encoded with Base58.
func (k *KeyS256) String(network wire.BitcoinNet) string {
	return encodeAddress(k.Bytes(network))
}

// Bytes returns type followed by the key data.
func (k *KeyS256) Bytes(network wire.BitcoinNet) []byte {
	var keyType byte

	// Add key type byte in front
	switch network {
	case MainNet:
		keyType = typePrivKey
	default:
		keyType = typeTestPrivKey
	}
	return append([]byte{keyType}, k.key.Serialize()...)
}

// Number returns 32 bytes representing the 256 bit big-endian integer of the private key.
func (k *KeyS256) Number() []byte {
	return k.key.Serialize()
}

// PublicKey returns the serialized (compresses) public key data.
func (k *KeyS256) PublicKey() []byte {
	return k.key.PubKey().SerializeCompressed()
}

// PublicKeyHash returns the Hashs160 of the serialized public key.
func (k *KeyS256) PublicKeyHash() []byte {
	return Hash160(k.PublicKey())
}

// Sign returns the serialized signature of the hash for the private key.
func (k *KeyS256) Sign(hash []byte) ([]byte, error) {
	signature, err := k.key.Sign(hash)
	if err != nil {
		return nil, err
	}
	return signature.Serialize(), nil
}
