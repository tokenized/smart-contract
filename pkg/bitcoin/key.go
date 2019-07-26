package bitcoin

import (
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	ErrBadKeyLength = errors.New("Key has invalid length")
)

const (
	typeMainPrivKey = 0x80 // Private Key
	typeTestPrivKey = 0xef // Testnet Private Key

	typeIntPrivKey = 0x40
)

type Key interface {
	// String returns the type followed by the key data with a checksum, encoded with Base58.
	String() string

	// Network returns the network id for the address.
	Network() wire.BitcoinNet

	// Bytes returns non-network specific type followed by the key data.
	Bytes() []byte

	// PublicKey returns the public key.
	PublicKey() PublicKey

	// Sign creates a signature from a hash.
	Sign([]byte) ([]byte, error)
}

// DecodeKeyString converts WIF (Wallet Import Format) key text to a key.
func DecodeKeyString(s string) (Key, error) {
	b, err := decodeAddress(s)
	if err != nil {
		return nil, err
	}

	var network wire.BitcoinNet
	switch b[0] {
	case typeMainPrivKey:
		network = MainNet
	case typeTestPrivKey:
		network = TestNet
	default:
		return nil, ErrBadType
	}

	if len(b) == 34 {
		if b[len(b)-1] != 0x01 {
			return nil, fmt.Errorf("Key not for compressed public : %x", b[len(b)-1:])
		}
		return KeyS256FromBytes(b[1:33], network)
	} else if len(b) == 33 {
		return KeyS256FromBytes(b[1:], network)
	}

	return nil, fmt.Errorf("Key unknown format length %d", len(b))
}

// DecodeKeyBytes decodes a binary bitcoin key. It returns the key and an error if there was an
//   issue.
func DecodeKeyBytes(b []byte, net wire.BitcoinNet) (Key, error) {
	if b[0] != typeIntPrivKey {
		return nil, ErrBadType
	}

	return KeyS256FromBytes(b[1:], net)
}

/****************************************** S256 **************************************************
/* An elliptic curve private key using the secp256k1 elliptic curve.
*/
type KeyS256 struct {
	key *btcec.PrivateKey
	net wire.BitcoinNet
}

// GenerateKeyS256 randomly generates a new key.
func GenerateKeyS256(net wire.BitcoinNet) (*KeyS256, error) {
	privkey, err := btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return nil, err
	}
	return KeyS256FromBytes(privkey.Serialize(), net)
}

// KeyS256FromBytes creates a key from a set of bytes that represents a 256 bit big-endian integer.
func KeyS256FromBytes(key []byte, net wire.BitcoinNet) (*KeyS256, error) {
	privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	return &KeyS256{key: privkey, net: net}, nil
}

// String returns the type followed by the key data with a checksum, encoded with Base58.
func (k *KeyS256) String() string {
	var keyType byte

	// Add key type byte in front
	switch k.net {
	case MainNet:
		keyType = typeMainPrivKey
	default:
		keyType = typeTestPrivKey
	}

	b := append([]byte{keyType}, k.key.Serialize()...)
	//b = append(b, 0x01) // compressed public key // Don't know if we want this or not.
	return encodeAddress(b)
}

// Network returns the network id for the key.
func (k *KeyS256) Network() wire.BitcoinNet {
	return k.net
}

// Bytes returns type followed by the key data.
func (k *KeyS256) Bytes() []byte {
	return append([]byte{typeIntPrivKey}, k.key.Serialize()...)
}

// Number returns 32 bytes representing the 256 bit big-endian integer of the private key.
func (k *KeyS256) Number() []byte {
	return k.key.Serialize()
}

// PublicKey returns the public key.
func (k *KeyS256) PublicKey() PublicKey {
	return publicKeyS256FromBTCEC(k.key.PubKey())
}

// Sign returns the serialized signature of the hash for the private key.
func (k *KeyS256) Sign(hash []byte) ([]byte, error) {
	signature, err := k.key.Sign(hash)
	if err != nil {
		return nil, err
	}
	return signature.Serialize(), nil
}
