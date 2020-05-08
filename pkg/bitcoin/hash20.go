package bitcoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const Hash20Size = 20

// Hash20 is a 20 byte integer in little endian format.
type Hash20 [Hash20Size]byte

func NewHash20(b []byte) (*Hash20, error) {
	if len(b) != Hash20Size {
		return nil, errors.New("Wrong byte length")
	}
	result := Hash20{}
	copy(result[:], b)
	return &result, nil
}

// NewHash20FromStr creates a little endian hash from a big endian string.
func NewHash20FromStr(s string) (*Hash20, error) {
	if len(s) != 2*Hash20Size {
		return nil, fmt.Errorf("Wrong size hex for Hash20 : %d", len(s))
	}

	b := make([]byte, Hash20Size)
	_, err := hex.Decode(b, []byte(s[:]))
	if err != nil {
		return nil, err
	}

	result := Hash20{}
	reverse20(result[:], b)
	return &result, nil
}

// NewHash20FromData creates a Hash20 by hashing the data with a Ripemd160(Sha256(b))
func NewHash20FromData(b []byte) (*Hash20, error) {
	return NewHash20(Ripemd160(Sha256(b)))
}

// Bytes returns the data for the hash.
func (h Hash20) Bytes() []byte {
	return h[:]
}

// SetBytes sets the value of the hash.
func (h *Hash20) SetBytes(b []byte) error {
	if len(b) != Hash20Size {
		return errors.New("Wrong byte length")
	}
	copy(h[:], b)
	return nil
}

// String returns the hex for the hash.
func (h *Hash20) String() string {
	var r [Hash20Size]byte
	reverse20(r[:], h[:])
	return fmt.Sprintf("%x", r[:])
}

// Equal returns true if the parameter has the same value.
func (h *Hash20) Equal(o *Hash20) bool {
	if h == nil {
		return o == nil
	}
	if o == nil {
		return false
	}
	return bytes.Equal(h[:], o[:])
}

// Serialize writes the hash into a writer.
func (h Hash20) Serialize(w io.Writer) error {
	_, err := w.Write(h[:])
	return err
}

func (h *Hash20) Deserialize(buf *bytes.Reader) error {
	if _, err := buf.Read(h[:]); err != nil {
		return err
	}
	return nil
}

// Deserialize reads a hash from a reader.
func DeserializeHash20(r io.Reader) (*Hash20, error) {
	result := Hash20{}
	_, err := r.Read(result[:])
	if err != nil {
		return nil, err
	}

	return &result, err
}

// MarshalJSON converts to json.
func (h Hash20) MarshalJSON() ([]byte, error) {
	var r [Hash20Size]byte
	reverse20(r[:], h[:])
	return []byte(fmt.Sprintf("\"%x\"", r[:])), nil
}

// UnmarshalJSON converts from json.
func (h *Hash20) UnmarshalJSON(data []byte) error {
	if len(data) != (2*Hash20Size)+2 {
		return fmt.Errorf("Wrong size hex for Hash20 : %d", len(data)-2)
	}

	b := make([]byte, Hash20Size)
	_, err := hex.Decode(b, data[1:len(data)-1])
	if err != nil {
		return err
	}
	reverse20(h[:], b)
	return nil
}

// Scan converts from a database column.
func (h *Hash20) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("Hash20 db column not bytes")
	}

	return h.SetBytes(b)
}

func reverse20(h, rh []byte) {
	i := Hash20Size - 1
	for _, b := range rh[:] {
		h[i] = b
		i--
	}
}
