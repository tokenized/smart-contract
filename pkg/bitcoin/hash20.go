package bitcoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

type Hash20 [20]byte

func NewHash20(b []byte) (*Hash20, error) {
	if len(b) != 20 {
		return nil, errors.New("Wrong byte length")
	}
	result := Hash20{}
	copy(result[:], b)
	return &result, nil
}

// Bytes returns the data for the hash.
func (h *Hash20) Bytes() []byte {
	return h[:]
}

// Equal returns true if the parameter has the same value.
func (h *Hash20) Equal(o *Hash20) bool {
	return bytes.Equal(h[:], o[:])
}

// Serialize writes the hash into a buffer.
func (h *Hash20) Serialize(buf *bytes.Buffer) error {
	_, err := buf.Write(h[:])
	return err
}

// MarshalJSON converts to json.
func (h *Hash20) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%x\"", h[:])), nil
}

// UnmarshalJSON converts from json.
func (h *Hash20) UnmarshalJSON(data []byte) error {
	if len(data) != 22 {
		return fmt.Errorf("Wrong size hex data for Hash22 : %d", len(data)-2)
	}

	_, err := hex.Decode(h[:], data[1:len(data)-1])
	return err
}
