package bitcoin

import (
	"bytes"
	"crypto/rand"
	"errors"

	bip32 "github.com/tyler-smith/go-bip32"
)

type ExtendedKey struct {
	key *bip32.Key
}

// GenerateExtendedKey creates a key from random data.
func GenerateMasterExtendedKey() (*ExtendedKey, error) {
	seed := make([]byte, 64)
	rand.Read(seed)
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	return &ExtendedKey{key: key}, nil
}

// ExtendedKeyFromBytes creates a key from bytes.
func ExtendedKeyFromBytes(b []byte) (*ExtendedKey, error) {
	key, err := bip32.Deserialize(b)
	if err != nil {
		return nil, err
	}
	return &ExtendedKey{key: key}, nil
}

// ExtendedKeyFromStr creates a key from a string.
func ExtendedKeyFromStr(s string) (*ExtendedKey, error) {
	key, err := bip32.B58Deserialize(s)
	if err != nil {
		return nil, err
	}
	return &ExtendedKey{key: key}, nil
}

// SetBytes decodes the key from bytes.
func (k *ExtendedKey) SetBytes(b []byte) error {
	key, err := bip32.Deserialize(b)
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

// Bytes returns the key data.
func (k ExtendedKey) Bytes() []byte {
	if k.key == nil {
		return nil
	}
	b, _ := k.key.Serialize()
	return b
}

// String returns the key formatted as text.
func (k *ExtendedKey) String() string {
	if k.key == nil {
		return ""
	}
	return k.key.B58Serialize()
}

// SetString decodes a key from text.
func (k *ExtendedKey) SetString(s string) error {
	key, err := bip32.B58Deserialize(s)
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

// Equal returns true if the other key has the same value
func (k *ExtendedKey) Equal(other *ExtendedKey) bool {
	if k.key == nil {
		return other.key == nil
	}
	if other.key == nil {
		return false
	}
	return bytes.Equal(k.key.Key, other.key.Key)
}

// IsPrivate returns true if the key is a private key.
func (k *ExtendedKey) IsPrivate() bool {
	if k.key == nil {
		return false
	}
	return k.key.IsPrivate
}

// Key returns the (private) key associated with this key.
func (k *ExtendedKey) Key(net Network) Key {
	if k.key == nil {
		return nil
	}
	if !k.IsPrivate() {
		return nil
	}
	r, _ := KeyS256FromBytes(k.key.Key[1:], net) // Skip first zero byte. We just want the 32 byte key value.
	return r
}

// PublicKey returns the public version of this key (xpub).
func (k *ExtendedKey) PublicKey() PublicKey {
	if k.key == nil {
		return nil
	}
	pk := k.key
	if k.IsPrivate() {
		pk = k.key.PublicKey()
	}
	pub, _ := DecodePublicKeyBytes(pk.Key)
	return pub
}

// ExtendedPublicKey returns the public version of this key.
func (k *ExtendedKey) ExtendedPublicKey() *ExtendedKey {
	if k.key == nil {
		return nil
	}
	return &ExtendedKey{key: k.key.PublicKey()}
}

// ChildKey returns the child key at the specified index.
func (k *ExtendedKey) ChildKey(index uint32) (*ExtendedKey, error) {
	if k.key == nil {
		return nil, errors.New("Nil can't create child")
	}
	child, err := k.key.NewChildKey(index)
	if err != nil {
		return nil, err
	}
	return &ExtendedKey{key: child}, nil
}

// MarshalJSON converts to json.
func (k *ExtendedKey) MarshalJSON() ([]byte, error) {
	if k.key == nil {
		return nil, nil
	}
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *ExtendedKey) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (k *ExtendedKey) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("ExtendedKey db column not bytes")
	}

	c := make([]byte, len(b))
	copy(c, b)
	return k.SetBytes(c)
}
