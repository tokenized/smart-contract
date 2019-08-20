package bitcoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

type Hash32 [32]byte

func NewHash32(b []byte) (*Hash32, error) {
	if len(b) != 32 {
		return nil, errors.New("Wrong byte length")
	}
	result := Hash32{}
	copy(result[:], b)
	return &result, nil
}

// Bytes returns the data for the hash.
func (h *Hash32) Bytes() []byte {
	return h[:]
}

// Equal returns true if the parameter has the same value.
func (h *Hash32) Equal(o *Hash32) bool {
	return bytes.Equal(h[:], o[:])
}

// Serialize writes the hash into a buffer.
func (h *Hash32) Serialize(buf *bytes.Buffer) error {
	_, err := buf.Write(h[:])
	return err
}

// MarshalJSON converts to json.
func (h *Hash32) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%x\"", h[:])), nil
}

// UnmarshalJSON converts from json.
func (h *Hash32) UnmarshalJSON(data []byte) error {
	if len(data) != 34 {
		return fmt.Errorf("Wrong size hex data for Hash32 : %d", len(data)-2)
	}

	_, err := hex.Decode(h[:], data[1:len(data)-1])
	return err
}
