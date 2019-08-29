package bitcoin

import (
	"bytes"
	"errors"

	bip32 "github.com/tyler-smith/go-bip32"
)

type BIP32Key struct {
	key *bip32.Key
}

// BIP32KeyFromBytes creates a key from bytes.
func BIP32KeyFromBytes(b []byte) (*BIP32Key, error) {
	key, err := bip32.Deserialize(b)
	if err != nil {
		return nil, err
	}
	return &BIP32Key{key: key}, nil
}

// BIP32KeyFromStr creates a key from a string.
func BIP32KeyFromStr(s string) (*BIP32Key, error) {
	key, err := bip32.B58Deserialize(s)
	if err != nil {
		return nil, err
	}
	return &BIP32Key{key: key}, nil
}

// SetBytes decodes the key from bytes.
func (k *BIP32Key) SetBytes(b []byte) error {
	key, err := bip32.Deserialize(b)
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

// Bytes returns the key data.
func (k *BIP32Key) Bytes() []byte {
	b, _ := k.key.Serialize()
	return b
}

// String returns the key formatted as text.
func (k *BIP32Key) String() string {
	return k.key.B58Serialize()
}

// SetString decodes a key from text.
func (k *BIP32Key) SetString(s string) error {
	key, err := bip32.B58Deserialize(s)
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

// Equal returns true if the other key has the same value
func (k *BIP32Key) Equal(other *BIP32Key) bool {
	return bytes.Equal(k.key.Key, other.key.Key)
}

// IsPrivate returns true if the key is a private key.
func (k *BIP32Key) IsPrivate() bool {
	return k.key.IsPrivate
}

// Key returns the (private) key associated with this key.
func (k *BIP32Key) Key(net Network) Key {
	if !k.IsPrivate() {
		return nil
	}
	r, _ := KeyS256FromBytes(k.key.Key[1:], net) // Skip first zero byte. We just want the 32 byte key value.
	return r
}

// PublicKey returns the public version of this key (xpub).
func (k *BIP32Key) PublicKey() PublicKey {
	pk := k.key
	if k.IsPrivate() {
		pk = k.key.PublicKey()
	}
	pub, _ := DecodePublicKeyBytes(pk.Key)
	return pub
}

// BIP32PublicKey returns the public version of this key (xpub).
func (k *BIP32Key) BIP32PublicKey() *BIP32Key {
	return &BIP32Key{key: k.key.PublicKey()}
}

// ChildKey returns the child key at the specified index.
func (k *BIP32Key) ChildKey(index uint32) (*BIP32Key, error) {
	child, err := k.key.NewChildKey(index)
	if err != nil {
		return nil, err
	}
	return &BIP32Key{key: child}, nil
}

// MarshalJSON converts to json.
func (k *BIP32Key) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *BIP32Key) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1:len(data)-1]))
}

// Scan converts from a database column.
func (k *BIP32Key) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("BIP32Key db column not bytes")
	}

	return k.SetBytes(b)
}
