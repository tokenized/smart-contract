package bitcoin

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	scriptTypePKH      = 0x30 // Public Key Hash
	scriptTypeSH       = 0x31 // Script Hash
	scriptTypeMultiPKH = 0x32 // Multi-PKH
	scriptTypeRPH      = 0x33 // RPH

	scriptHashLength = 20 // Length of standard public key, script, and R hashes RIPEMD(SHA256())
)

// RawAddress represents a bitcoin address in raw format, with no check sum or encoding.
// It represents a "script template" for common locking and unlocking scripts.
// It enables parsing and creating of common locking and unlocking scripts as well as identifying
//   participants involved in the scripts via public key hashes and other hashes.
type RawAddress interface {
	// Bytes returns the non-network specific type followed by the address data.
	Bytes() []byte

	// LockingScript returns the bitcoin output(locking) script for paying to the address.
	LockingScript() []byte

	// Equal returns true if the address parameter has the same value.
	Equal(RawAddress) bool

	// Serialize writes the address into a buffer.
	Serialize(*bytes.Buffer) error

	// Hash returns the hash corresponding to the address.
	Hash() (*Hash20, error)
}

// DecodeRawAddress decodes a binary bitcoin script template. It returns the script
//   template, and an error if there was an issue.
func DecodeRawAddress(b []byte) (RawAddress, error) {
	switch b[0] {
	case scriptTypePKH:
		return NewRawAddressPKH(b[1:])
	case scriptTypeSH:
		return NewRawAddressSH(b[1:])
	case scriptTypeMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:4])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		// Parse hash count
		var count uint16
		if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
			return nil, err
		}
		b = b[4:] // remove counts
		pkhs := make([][]byte, 0, count)
		for i := uint16(0); i < count; i++ {
			if len(b) < scriptHashLength {
				return nil, ErrBadScriptHashLength
			}
			pkhs = append(pkhs, b[:scriptHashLength])
			b = b[scriptHashLength:]
		}
		return NewRawAddressMultiPKH(required, pkhs)
	case scriptTypeRPH:
		return NewRawAddressRPH(b[1:])
	}

	return nil, ErrBadType
}

func DeserializeRawAddress(buf *bytes.Reader) (RawAddress, error) {
	t, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	switch t {
	case scriptTypePKH:
		pkh := make([]byte, scriptHashLength)
		if _, err := buf.Read(pkh); err != nil {
			return nil, err
		}
		return NewRawAddressPKH(pkh)
	case scriptTypeSH:
		sh := make([]byte, scriptHashLength)
		if _, err := buf.Read(sh); err != nil {
			return nil, err
		}
		return NewRawAddressSH(sh)
	case scriptTypeMultiPKH:
		// Parse required count
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		// Parse hash count
		var count uint16
		if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
			return nil, err
		}
		pkhs := make([][]byte, 0, count)
		for i := uint16(0); i < count; i++ {
			pkh := make([]byte, scriptHashLength)
			if _, err := buf.Read(pkh); err != nil {
				return nil, err
			}
			pkhs = append(pkhs, pkh)
		}
		return NewRawAddressMultiPKH(required, pkhs)
	case scriptTypeRPH:
		rph := make([]byte, scriptHashLength)
		if _, err := buf.Read(rph); err != nil {
			return nil, err
		}
		return NewRawAddressRPH(rph)
	}

	return nil, ErrBadType
}

/****************************************** PKH ***************************************************/
type RawAddressPKH struct {
	pkh []byte
}

// NewRawAddressPKH creates an address from a public key hash.
func NewRawAddressPKH(pkh []byte) (*RawAddressPKH, error) {
	if len(pkh) != scriptHashLength {
		return nil, ErrBadScriptHashLength
	}
	return &RawAddressPKH{pkh: pkh}, nil
}

func (a *RawAddressPKH) PKH() []byte {
	return a.pkh
}

func (a *RawAddressPKH) Bytes() []byte {
	return append([]byte{scriptTypePKH}, a.pkh...)
}

func (a *RawAddressPKH) Equal(other RawAddress) bool {
	if other == nil {
		return false
	}
	switch o := other.(type) {
	case *RawAddressPKH:
		return bytes.Equal(a.pkh, o.pkh)
	case *AddressPKH:
		return bytes.Equal(a.pkh, o.pkh)
	case *ConcreteRawAddress:
		return a.Equal(o.RawAddress())
	}
	return false
}

func (a *RawAddressPKH) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(scriptTypePKH); err != nil {
		return err
	}
	if _, err := buf.Write(a.pkh); err != nil {
		return err
	}
	return nil
}

// Hash returns the hash corresponding to the address.
func (a *RawAddressPKH) Hash() (*Hash20, error) {
	return NewHash20(a.pkh)
}

/******************************************* SH ***************************************************/
type RawAddressSH struct {
	sh []byte
}

// NewRawAddressSH creates an address from a script hash.
func NewRawAddressSH(sh []byte) (*RawAddressSH, error) {
	if len(sh) != scriptHashLength {
		return nil, ErrBadScriptHashLength
	}
	return &RawAddressSH{sh: sh}, nil
}

func (a *RawAddressSH) SH() []byte {
	return a.sh
}

func (a *RawAddressSH) Bytes() []byte {
	return append([]byte{scriptTypeSH}, a.sh...)
}

func (a *RawAddressSH) Equal(other RawAddress) bool {
	if other == nil {
		return false
	}
	switch o := other.(type) {
	case *RawAddressSH:
		return bytes.Equal(a.sh, o.sh)
	case *AddressSH:
		return bytes.Equal(a.sh, o.sh)
	case *ConcreteRawAddress:
		return a.Equal(o.RawAddress())
	}
	return false
}

func (a *RawAddressSH) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(scriptTypeSH); err != nil {
		return err
	}
	if _, err := buf.Write(a.sh); err != nil {
		return err
	}
	return nil
}

// Hash returns the hash corresponding to the address.
func (a *RawAddressSH) Hash() (*Hash20, error) {
	return NewHash20(a.sh)
}

/**************************************** MultiPKH ************************************************/
type RawAddressMultiPKH struct {
	pkhs     [][]byte
	required uint16
}

// NewRawAddressMultiPKH creates an address from a required signature count and some public key hashes.
func NewRawAddressMultiPKH(required uint16, pkhs [][]byte) (*RawAddressMultiPKH, error) {
	for _, pkh := range pkhs {
		if len(pkh) != scriptHashLength {
			return nil, ErrBadScriptHashLength
		}
	}
	return &RawAddressMultiPKH{pkhs: pkhs, required: required}, nil
}

func (a *RawAddressMultiPKH) PKHs() []byte {
	b := make([]byte, 0, len(a.pkhs)*scriptHashLength)
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}
	return b
}

func (a *RawAddressMultiPKH) Bytes() []byte {
	b := make([]byte, 0, 5+(len(a.pkhs)*scriptHashLength))

	b = append(b, byte(scriptTypeMultiPKH))

	// Append required and hash counts
	var numBuf bytes.Buffer
	binary.Write(&numBuf, binary.LittleEndian, a.required)
	binary.Write(&numBuf, binary.LittleEndian, uint16(len(a.pkhs)))
	b = append(b, numBuf.Bytes()...)

	// Append all pkhs
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}

	return b
}

func (a *RawAddressMultiPKH) Equal(other RawAddress) bool {
	if other == nil {
		return false
	}

	switch o := other.(type) {
	case *RawAddressMultiPKH:
		if len(a.pkhs) != len(o.pkhs) {
			return false
		}

		for i, pkh := range a.pkhs {
			if !bytes.Equal(pkh, o.pkhs[i]) {
				return false
			}
		}
		return true
	case *AddressMultiPKH:
		if len(a.pkhs) != len(o.pkhs) {
			return false
		}

		for i, pkh := range a.pkhs {
			if !bytes.Equal(pkh, o.pkhs[i]) {
				return false
			}
		}
		return true
	case *ConcreteRawAddress:
		return a.Equal(o.RawAddress())
	}

	return false
}

func (a *RawAddressMultiPKH) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(scriptTypeMultiPKH); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, a.required); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(a.pkhs))); err != nil {
		return err
	}

	// Write all pkhs
	for _, pkh := range a.pkhs {
		if _, err := buf.Write(pkh); err != nil {
			return err
		}
	}

	return nil
}

// Hash returns the hash corresponding to the address.
func (a *RawAddressMultiPKH) Hash() (*Hash20, error) {
	return NewHash20(Hash160(a.Bytes()))
}

/***************************************** RPH ************************************************/
type RawAddressRPH struct {
	rph []byte
}

// NewRawAddressRPH creates an address from an R puzzle hash.
func NewRawAddressRPH(rph []byte) (*RawAddressRPH, error) {
	if len(rph) != scriptHashLength {
		return nil, ErrBadScriptHashLength
	}
	return &RawAddressRPH{rph: rph}, nil
}

func (a *RawAddressRPH) RPH() []byte {
	return a.rph
}

func (a *RawAddressRPH) Bytes() []byte {
	return append([]byte{scriptTypeRPH}, a.rph...)
}

func (a *RawAddressRPH) Equal(other RawAddress) bool {
	if other == nil {
		return false
	}
	switch o := other.(type) {
	case *RawAddressRPH:
		return bytes.Equal(a.rph, o.rph)
	case *AddressRPH:
		return bytes.Equal(a.rph, o.rph)
	case *ConcreteRawAddress:
		return a.Equal(o.RawAddress())
	}
	return false
}

func (a *RawAddressRPH) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(scriptTypeRPH); err != nil {
		return err
	}
	if _, err := buf.Write(a.rph); err != nil {
		return err
	}
	return nil
}

// Hash returns the hash corresponding to the address.
func (a *RawAddressRPH) Hash() (*Hash20, error) {
	return NewHash20(a.rph)
}

// ConcreteRawAddress is a concrete form of RawAddress.
// It does things not possible with an interface.
// It implements marshal and unmarshal to/from JSON.
// It also Scan for converting from a database column.
type ConcreteRawAddress struct {
	ra RawAddress
}

func NewConcreteRawAddress(ra RawAddress) *ConcreteRawAddress {
	return &ConcreteRawAddress{ra}
}

func (a *ConcreteRawAddress) RawAddress() RawAddress {
	return a.ra
}

// Bytes returns the non-network specific type followed by the address data.
func (a *ConcreteRawAddress) Bytes() []byte {
	return a.ra.Bytes()
}

// LockingScript returns the bitcoin output(locking) script for paying to the address.
func (a *ConcreteRawAddress) LockingScript() []byte {
	return a.ra.LockingScript()
}

// Equal returns true if the address parameter has the same value.
func (a *ConcreteRawAddress) Equal(other RawAddress) bool {
	return a.ra.Equal(other)
}

// Serialize writes the address into a buffer.
func (a *ConcreteRawAddress) Serialize(buf *bytes.Buffer) error {
	return a.ra.Serialize(buf)
}

// Hash returns the hash corresponding to the address.
func (a *ConcreteRawAddress) Hash() (*Hash20, error) {
	if a.ra == nil {
		return nil, errors.New("Empty JSON Raw Address")
	}
	return a.ra.Hash()
}

// MarshalJSON converts to json.
func (a *ConcreteRawAddress) MarshalJSON() ([]byte, error) {
	if a.ra == nil {
		return []byte("\"\""), nil
	}
	return []byte("\"" + base64.StdEncoding.EncodeToString(a.ra.Bytes()) + "\""), nil
}

// UnmarshalJSON converts from json.
func (a *ConcreteRawAddress) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("Too short for RawAddress hex data : %d", len(data))
	}

	if len(data) == 2 {
		a.ra = nil // empty
		return nil
	}

	raw, err := base64.StdEncoding.DecodeString(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	a.ra, err = DecodeRawAddress(raw)
	return err
}

// Scan converts from a database column.
func (a *ConcreteRawAddress) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("ConcreteRawAddress db column not bytes")
	}

	var err error
	c := make([]byte, len(b))
	copy(c, b)
	a.ra, err = DecodeRawAddress(c)
	return err
}
