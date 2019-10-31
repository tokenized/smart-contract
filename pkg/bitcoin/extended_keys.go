package bitcoin

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	ExtendedKeysHeader = 0x41
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
		return nil, errors.New("Not an extended key list")
	}

	count, err := readBase128VarInt(buf)
	if err != nil {
		return nil, errors.Wrap(err, "read count")
	}

	result := make(ExtendedKeys, 0, count)
	for i := 0; i < count; i++ {
		ek, err := readExtendedKey(buf)
		if err != nil {
			return nil, errors.Wrap(err, "read xkey base")
		}
		result = append(result, ek)
	}

	return result, nil
}

// ExtendedKeysFromStr creates a list of keys from a string.
func ExtendedKeysFromStr(s string) (ExtendedKeys, error) {
	if len(s) < 4 {
		return ExtendedKeys{}, errors.New("Too Short")
	}
	hash := DoubleSha256([]byte(s[:len(s)-4]))
	check := hex.EncodeToString(hash[:4])
	if check != s[len(s)-4:] {
		return ExtendedKeys{}, errors.New("Invalid check hash")
	}

	parts := strings.Split(s, ":")

	if len(parts) > 2 {
		return ExtendedKeys{}, errors.New("To many colons in xkey")
	}

	if len(parts) == 2 {
		if parts[0] != "bitcoin-xkeys" {
			return ExtendedKeys{}, fmt.Errorf("Invalid xkey prefix : %s", parts[0])
		}
		s = parts[1]
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return ExtendedKeys{}, errors.Wrap(err, "decode xkey hex")
	}

	return ExtendedKeysFromBytes(b)
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

	if err := writeBase128VarInt(&buf, len(k)); err != nil {
		return nil
	}

	for _, key := range k {
		if err := writeExtendedKey(key, &buf); err != nil {
			return nil
		}
	}

	return buf.Bytes()
}

// String returns the list of keys formatted as text.
func (k ExtendedKeys) String() string {
	result := "bitcoin-xkeys:0100" + hex.EncodeToString(k.Bytes())

	// Append first 4 bytes of double SHA256 of hash of preceding text
	check := DoubleSha256([]byte(result))
	return result + hex.EncodeToString(check[:4])
}

// SetString decodes a list of keys from text.
func (k *ExtendedKeys) SetString(s string) error {
	nk, err := ExtendedKeysFromStr(s)
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

// ExtendedPublicKey returns the public version of this key.
func (k ExtendedKeys) ExtendedPublicKeys() ExtendedKeys {
	result := make(ExtendedKeys, 0, len(k))
	for _, key := range k {
		result = append(result, key.ExtendedPublicKey())
	}
	return result
}

// MarshalJSON converts to json.
func (k *ExtendedKeys) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *ExtendedKeys) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1 : len(data)-1]))
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

func readBase128VarInt(r io.ByteReader) (int, error) {
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

func writeBase128VarInt(w io.ByteWriter, value int) error {
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
