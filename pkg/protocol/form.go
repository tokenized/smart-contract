package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrUnknownMessage is returned when there is no matching type for the
	// given code.
	ErrUnknownMessage = errors.New("Unknown message")

	// ErrLengthExceeded is returned when a string is too long for the
	// target type.
	ErrLengthExceeded = errors.New("Length exceeded")
)

// Form is the interface for JSON protocol messages.
type Form interface {
	io.Writer
	Validate() error
	BuildMessage() (OpReturnMessage, error)
}

// PayloadForm is the interface for payload sub-forms, such as required for
// the Asset messages.
type PayloadForm interface {
	PayloadMessage([]byte) (PayloadMessage, error)
	Bytes() ([]byte, error)
}

// NewPayloadFormByCode returns a new PayloadForm by the given code.
//
// Example codes : SHC, COU, etc
func NewPayloadFormByCode(code string) (PayloadForm, error) {
	if code == CodeAssetTypeCoupon {
		return NewAssetTypeCouponForm(), nil
	}

	if code == CodeAssetTypeMovieTicket {
		return NewAssetTypeMovieTicketForm(), nil
	}

	if code == CodeAssetTypeShareCommon {
		return NewAssetTypeShareCommonForm(), nil
	}

	if code == CodeAssetTypeTicketAdmission {
		return NewAssetTypeTicketAdmissionForm(), nil
	}

	return nil, fmt.Errorf("No matching form for code %s", code)
}

// BaseForm is the common struct for all ProtocolForm's.
type BaseForm struct {
}

// read fills the value with the appropriate number of bytes from the buffer.
//
// This is useful for fixed size types such as int, float etc.
func (f BaseForm) read(buf *bytes.Buffer, v interface{}) error {
	return binary.Read(buf, binary.BigEndian, v)
}

// readLen reads the number of bytes from the buffer to fill the slice of
// []byte.
func (f BaseForm) readLen(buf *bytes.Buffer, b []byte) error {
	_, err := io.ReadFull(buf, b)
	return err
}

// asByte enforces the conversion of a single character string to a byte.
func (f BaseForm) asByte(s string) (byte, error) {
	if len(s) > 1 {
		return byte(0), ErrLengthExceeded
	}

	return []byte(s)[:1][0], nil
}

// pad returns a []byte of length l, from the string.
//
// An error will be returned if the string length exceeds l.
func (f BaseForm) pad(s string, l int) ([]byte, error) {
	b := []byte(s)

	if len(b) > l {
		return nil, ErrLengthExceeded
	}

	if len(b) == l {
		return b, nil
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...), nil
}

// ensureByte
func (f BaseForm) ensureByte(s string) (byte, error) {
	b, err := f.pad(s, 1)

	if err != nil {
		return byte(0x0), err
	}

	return b[0], err
}
