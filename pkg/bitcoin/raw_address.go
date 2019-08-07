package bitcoin

import (
	"bytes"
	"encoding/binary"
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
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/scriptHashLength)
		for len(b) >= 0 {
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
	}
	return false
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
	}
	return false
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
	b := make([]byte, 0, 3+(len(a.pkhs)*scriptHashLength))

	b = append(b, byte(scriptTypeMultiPKH))

	// Append required count
	var numBuf bytes.Buffer
	binary.Write(&numBuf, binary.LittleEndian, a.required)
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
	}

	return false
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
	}
	return false
}
