package bitcoin

import (
	"bytes"
	"encoding/binary"
	"errors"
)

var (
	ErrBadScriptHashLength   = errors.New("Script hash has invalid length")
	ErrBadCheckSum           = errors.New("Address has bad checksum")
	ErrBadType               = errors.New("Address type unknown")
	ErrUnknownScriptTemplate = errors.New("Unknown script template")
)

const (
	addressTypeMainPKH      = 0x00 // Public Key Hash
	addressTypeMainSH       = 0x05 // Script Hash
	addressTypeMainMultiPKH = 0x10 // Multi-PKH - Experimental value. Not standard
	addressTypeMainRPH      = 0x20 // RPH - Experimental value. Not standard

	addressTypeTestPKH      = 0x6f // Testnet Public Key Hash
	addressTypeTestSH       = 0xc4 // Testnet Script Hash
	addressTypeTestMultiPKH = 0xd0 // Multi-PKH - Experimental value. Not standard
	addressTypeTestRPH      = 0xe0 // RPH - Experimental value. Not standard
)

type Address interface {
	// String returns the type and address data followed by a checksum encoded with Base58.
	String() string

	// Network returns the network id for the address.
	Network() Network

	RawAddress
}

// DecodeAddress decodes a base58 text bitcoin address. It returns the address, and an error
//   if there was an issue.
func DecodeAddress(address string) (Address, error) {
	b, err := decodeAddress(address)
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case addressTypeMainPKH:
		return NewAddressPKH(b[1:], MainNet)
	case addressTypeMainSH:
		return NewAddressSH(b[1:], MainNet)
	case addressTypeMainMultiPKH:
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
		return NewAddressMultiPKH(required, pkhs, MainNet)
	case addressTypeMainRPH:
		return NewAddressRPH(b[1:], MainNet)
	case addressTypeTestPKH:
		return NewAddressPKH(b[1:], TestNet)
	case addressTypeTestSH:
		return NewAddressSH(b[1:], TestNet)
	case addressTypeTestMultiPKH:
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
		return NewAddressMultiPKH(required, pkhs, TestNet)
	case addressTypeTestRPH:
		return NewAddressRPH(b[1:], TestNet)
	}

	return nil, ErrBadType
}

// DecodeNetMatches returns true if the decoded network id matches the specified network id.
// All test network ids decode as TestNet.
func DecodeNetMatches(decoded Network, desired Network) bool {
	switch decoded {
	case MainNet:
		return desired == MainNet
	case TestNet:
		return desired != MainNet
	}

	return false
}

func NewAddressFromRawAddress(st RawAddress, net Network) Address {
	switch t := st.(type) {
	case *RawAddressPKH:
		return &AddressPKH{t, net}
	case *RawAddressSH:
		return &AddressSH{t, net}
	case *RawAddressMultiPKH:
		return &AddressMultiPKH{t, net}
	case *RawAddressRPH:
		return &AddressRPH{t, net}
	case *AddressPKH:
		return t
	case *AddressSH:
		return t
	case *AddressMultiPKH:
		return t
	case *AddressRPH:
		return t
	case *ConcreteRawAddress:
		return NewAddressFromRawAddress(t.RawAddress(), net)
	}

	return nil
}

// PKH is a helper function that returns the PKH for a RawAddress or Address. It returns false
//   if there is no PKH.
func PKH(st RawAddress) ([]byte, bool) {
	switch a := st.(type) {
	case *RawAddressPKH:
		return a.PKH(), true
	case *AddressPKH:
		return a.PKH(), true
	case *ConcreteRawAddress:
		return PKH(a.RawAddress())
	}

	return nil, false
}

// SH is a helper function that returns the SH for a RawAddress or Address. It returns false
//   if there is no SH.
func SH(st RawAddress) ([]byte, bool) {
	switch a := st.(type) {
	case *RawAddressSH:
		return a.SH(), true
	case *AddressSH:
		return a.SH(), true
	case *ConcreteRawAddress:
		return SH(a.RawAddress())
	}

	return nil, false
}

// PKHs is a helper function that returns the PKHs for a RawAddress or Address. It returns false
//   if there is no PKHs.
func PKHs(st RawAddress) ([]byte, bool) {
	switch a := st.(type) {
	case *RawAddressMultiPKH:
		return a.PKHs(), true
	case *AddressMultiPKH:
		return a.PKHs(), true
	case *ConcreteRawAddress:
		return PKHs(a.RawAddress())
	}

	return nil, false
}

// RPH is a helper function that returns the RPH for a RawAddress or Address. It returns false
//   if there is no RPH.
func RPH(st RawAddress) ([]byte, bool) {
	switch a := st.(type) {
	case *RawAddressRPH:
		return a.RPH(), true
	case *AddressRPH:
		return a.RPH(), true
	case *ConcreteRawAddress:
		return RPH(a.RawAddress())
	}

	return nil, false
}

/****************************************** PKH ***************************************************/
type AddressPKH struct {
	*RawAddressPKH
	net Network
}

// NewAddressPKH creates an address from a public key hash.
func NewAddressPKH(pkh []byte, net Network) (*AddressPKH, error) {
	st, err := NewRawAddressPKH(pkh)
	if err != nil {
		return nil, err
	}
	return &AddressPKH{st, net}, nil
}

// String returns the type and address data followed by a checksum encoded with Base58.
func (a *AddressPKH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = addressTypeMainPKH
	default:
		addressType = addressTypeTestPKH
	}
	return encodeAddress(append([]byte{addressType}, a.pkh...))
}

// Network returns the network id for the address.
func (a *AddressPKH) Network() Network {
	return a.net
}

/******************************************* SH ***************************************************/
type AddressSH struct {
	*RawAddressSH
	net Network
}

// NewAddressSH creates an address from a script hash.
func NewAddressSH(sh []byte, net Network) (*AddressSH, error) {
	st, err := NewRawAddressSH(sh)
	if err != nil {
		return nil, err
	}
	return &AddressSH{st, net}, nil
}

// String returns the type and address data followed by a checksum encoded with Base58.
func (a *AddressSH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = addressTypeMainSH
	default:
		addressType = addressTypeTestSH
	}
	return encodeAddress(append([]byte{addressType}, a.sh...))
}

// Network returns the network id for the address.
func (a *AddressSH) Network() Network {
	return a.net
}

/**************************************** MultiPKH ************************************************/
type AddressMultiPKH struct {
	*RawAddressMultiPKH
	net Network
}

// NewAddressMultiPKH creates an address from a required signature count and some public key hashes.
func NewAddressMultiPKH(required uint16, pkhs [][]byte, net Network) (*AddressMultiPKH, error) {
	st, err := NewRawAddressMultiPKH(required, pkhs)
	if err != nil {
		return nil, err
	}
	return &AddressMultiPKH{st, net}, nil
}

// String returns the type and address data followed by a checksum encoded with Base58.
func (a *AddressMultiPKH) String() string {
	b := make([]byte, 0, 3+(len(a.pkhs)*scriptHashLength))

	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = addressTypeMainMultiPKH
	default:
		addressType = addressTypeTestMultiPKH
	}
	b = append(b, byte(addressType))

	// Append required count
	var numBuf bytes.Buffer
	binary.Write(&numBuf, binary.LittleEndian, a.required)
	b = append(b, numBuf.Bytes()...)

	// Append all pkhs
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}

	return encodeAddress(b)
}

// Network returns the network id for the address.
func (a *AddressMultiPKH) Network() Network {
	return a.net
}

/***************************************** RPH ************************************************/
type AddressRPH struct {
	*RawAddressRPH
	net Network
}

// NewAddressRPH creates an address from an R puzzle hash.
func NewAddressRPH(rph []byte, net Network) (*AddressRPH, error) {
	st, err := NewRawAddressRPH(rph)
	if err != nil {
		return nil, err
	}
	return &AddressRPH{st, net}, nil
}

// String returns the type and address data followed by a checksum encoded with Base58.
func (a *AddressRPH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = addressTypeMainRPH
	default:
		addressType = addressTypeTestRPH
	}
	return encodeAddress(append([]byte{addressType}, a.rph...))
}

// Network returns the network id for the address.
func (a *AddressRPH) Network() Network {
	return a.net
}

func encodeAddress(b []byte) string {
	// Perform Double SHA-256 hash
	checksum := DoubleSha256(b)

	// Append the first 4 checksum bytes
	address := append(b, checksum[:4]...)

	// Convert the result from a byte string into a base58 string using
	// Base58 encoding. This is the most commonly used Bitcoin Address
	// format
	return Base58(address)
}

func decodeAddress(address string) ([]byte, error) {
	b := Base58Decode(address)

	if len(b) < 5 {
		return nil, ErrBadCheckSum
	}

	// Verify checksum
	checksum := DoubleSha256(b[:len(b)-4])
	if !bytes.Equal(checksum[:4], b[len(b)-4:]) {
		return nil, ErrBadCheckSum
	}

	return b[:len(b)-4], nil
}
