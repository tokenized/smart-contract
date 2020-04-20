package bitcoin

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	bip32 "github.com/tyler-smith/go-bip32"
)

const (
	ExtendedKeysHeader    = 0x41
	ExtendedKeysURLPrefix = "bitcoin-xkeys"
)

var (
	ErrNotExtendedKeys = errors.New("Data not an xkeys")
)

type ExtendedKeys []ExtendedKey

// ExtendedKeysFromBytes creates a list of keys from bytes.
func ExtendedKeysFromBytes(b []byte) (ExtendedKeys, error) {
	buf := bytes.NewReader(b)

	header, err := buf.ReadByte()
	if err != nil {
		return nil, errors.Wrap(err, "read header")
	}
	if header != ExtendedKeysHeader {
		// Fall back to BIP-0032 format
		bip32Key, err := bip32.Deserialize(b)
		if err != nil {
			return ExtendedKeys{}, ErrNotExtendedKeys
		}

		result, err := fromBIP32(bip32Key)
		if err != nil {
			return ExtendedKeys{}, err
		}
		return ExtendedKeys{result}, nil
	}

	count, err := ReadBase128VarInt(buf)
	if err != nil {
		return nil, errors.Wrap(err, "read count")
	}

	result := make(ExtendedKeys, 0, count)
	for i := 0; i < count; i++ {
		var ek ExtendedKey
		if err := ek.read(buf); err != nil {
			return nil, errors.Wrap(err, "read xkey base")
		}
		result = append(result, ek)
	}

	return result, nil
}

// ExtendedKeysFromStr creates a list of keys from a hex string.
func ExtendedKeysFromStr(s string) (ExtendedKeys, error) {
	net, prefix, data, err := BIP0276Decode(s)
	if err != nil {
		// Fall back to BIP-0032 format
		bip32Key, b32err := bip32.B58Deserialize(s)
		if b32err != nil {
			return ExtendedKeys{}, errors.Wrap(err, "decode xkeys hex string")
		}

		result, err := fromBIP32(bip32Key)
		if err != nil {
			return ExtendedKeys{}, err
		}
		return ExtendedKeys{result}, nil
	}

	if prefix != ExtendedKeysURLPrefix {
		return ExtendedKeys{}, fmt.Errorf("Wrong prefix : %s", prefix)
	}

	result, err := ExtendedKeysFromBytes(data)
	if err != nil {
		return ExtendedKeys{}, err
	}

	for i, _ := range result {
		result[i].Network = net
	}

	return result, nil
}

// ExtendedKeysFromStr58 creates a list of keys from a base 58 string.
func ExtendedKeysFromStr58(s string) (ExtendedKeys, error) {
	net, prefix, data, err := BIP0276Decode58(s)
	if err != nil {
		// Fall back to BIP-0032 format
		bip32Key, b32err := bip32.B58Deserialize(s)
		if b32err != nil {
			return ExtendedKeys{}, errors.Wrap(err, "decode xkeys base58 string")
		}

		result, err := fromBIP32(bip32Key)
		if err != nil {
			return ExtendedKeys{}, err
		}
		return ExtendedKeys{result}, nil
	}

	if prefix != ExtendedKeysURLPrefix {
		return ExtendedKeys{}, fmt.Errorf("Wrong prefix : %s", prefix)
	}

	result, err := ExtendedKeysFromBytes(data)
	if err != nil {
		return ExtendedKeys{}, err
	}

	for i, _ := range result {
		result[i].Network = net
	}

	return result, nil
}

// SetBytes decodes the list of keys from bytes.
func (k *ExtendedKeys) SetBytes(b []byte) error {
	nks, err := ExtendedKeysFromBytes(b)
	if err != nil {
		return err
	}

	*k = nks
	return nil
}

// Bytes returns the list of keys data.
func (k ExtendedKeys) Bytes() []byte {
	var buf bytes.Buffer

	if err := buf.WriteByte(ExtendedKeysHeader); err != nil {
		return nil
	}

	if err := WriteBase128VarInt(&buf, len(k)); err != nil {
		return nil
	}

	for _, key := range k {
		if err := key.write(&buf); err != nil {
			return nil
		}
	}

	return buf.Bytes()
}

// String returns the list of keys formatted as hex text.
func (k ExtendedKeys) String() string {
	var net Network
	if len(k) > 0 {
		net = k[0].Network
	}
	return BIP0276Encode(net, ExtendedKeysURLPrefix, k.Bytes())
}

// String58 returns the list of keys formatted as base58 text.
func (k ExtendedKeys) String58() string {
	var net Network
	if len(k) > 0 {
		net = k[0].Network
	}
	return BIP0276Encode58(net, ExtendedKeysURLPrefix, k.Bytes())
}

// SetString decodes a list of keys from hex text.
func (k *ExtendedKeys) SetString(s string) error {
	nk, err := ExtendedKeysFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// SetString58 decodes a list of keys from base 58 text.
func (k *ExtendedKeys) SetString58(s string) error {
	nk, err := ExtendedKeysFromStr58(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// Equal returns true if the other list of keys have the same values
func (k ExtendedKeys) Equal(other ExtendedKeys) bool {
	if len(k) != len(other) {
		return false
	}
	for i, key := range k {
		if !key.Equal(other[i]) {
			return false
		}
	}
	return true
}

// RawAddress returns a raw address for this list of keys.
func (k ExtendedKeys) RawAddress(requiredSigners int) (RawAddress, error) {
	if len(k) == 1 {
		return k[0].RawAddress()
	}

	pkhs := make([][]byte, 0, len(k))
	for _, key := range k {
		pkhs = append(pkhs, Hash160(key.PublicKey().Bytes()))
	}
	return NewRawAddressMultiPKH(requiredSigners, pkhs)
}

// ExtendedPublicKeys returns the public version of this list of keys.
func (k ExtendedKeys) ExtendedPublicKeys() ExtendedKeys {
	result := make(ExtendedKeys, 0, len(k))
	for _, key := range k {
		result = append(result, key.ExtendedPublicKey())
	}
	return result
}

// ChildKeys returns the child keys at the specified index.
func (k ExtendedKeys) ChildKeys(index uint32) (ExtendedKeys, error) {
	result := make(ExtendedKeys, 0, len(k))
	for _, key := range k {
		child, err := key.ChildKey(index)
		if err != nil {
			return result, err
		}
		result = append(result, child)
	}
	return result, nil
}

// ChildKeysForPath returns the child key at the specified index path.
func (k ExtendedKeys) ChildKeysForPath(path []uint32) (ExtendedKeys, error) {
	var err error
	result := k
	for _, index := range path {
		result, err = result.ChildKeys(index)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// MarshalJSON converts to json.
func (k ExtendedKeys) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String58() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *ExtendedKeys) UnmarshalJSON(data []byte) error {
	return k.SetString58(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (k *ExtendedKeys) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("ExtendedKeys db column not bytes")
	}

	c := make([]byte, len(b))
	copy(c, b)
	return k.SetBytes(c)
}

func ReadBase128VarInt(r io.ByteReader) (int, error) {
	value := uint32(0)
	done := false
	bitOffset := uint32(0)
	for !done {
		subValue, err := r.ReadByte()
		if err != nil {
			return int(value), err
		}

		done = (subValue & 0x80) == 0 // High bit not set
		subValue = subValue & 0x7f    // Remove high bit

		value += uint32(subValue) << bitOffset
		bitOffset += 7
	}

	return int(value), nil
}

func WriteBase128VarInt(w io.ByteWriter, value int) error {
	v := uint32(value)
	for {
		if v < 128 {
			return w.WriteByte(byte(v))
		}
		subValue := (byte(v&0x7f) | 0x80) // Get last 7 bits and set high bit
		if err := w.WriteByte(subValue); err != nil {
			return err
		}
		v = v >> 7
	}
}
