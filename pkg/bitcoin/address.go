package bitcoin

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	ErrBadHashLength         = errors.New("Hash has invalid length")
	ErrBadCheckSum           = errors.New("Address has bad checksum")
	ErrBadType               = errors.New("Address type unknown")
	ErrUnknownScriptTemplate = errors.New("Unknown script template")
)

const (
	typePKH      = 0x00 // Public Key Hash
	typeSH       = 0x05 // Script Hash
	typeMultiPKH = 0x10 // Experimental value. Not standard
	typeRPH      = 0x20 // Experimental value. Not standard

	typeTestPKH      = 0x6f // Testnet Public Key Hash
	typeTestSH       = 0xc4 // Testnet Script Hash
	typeTestMultiPKH = 0xd0 // Experimental value. Not standard
	typeTestRPH      = 0xe0 // Experimental value. Not standard

	hashLength = 20 // Length of public key, script, and R hashes
)

type Address interface {
	// String returns the type and address data followed by a checksum encoded with Base58.
	String(network wire.BitcoinNet) string

	// Bytes returns the type followed by the address data.
	Bytes(network wire.BitcoinNet) []byte

	// LockingScript returns the bitcoin output(locking) script for paying to the address.
	LockingScript() []byte

	// Equal returns true if the address parameter has the same value.
	Equal(Address) bool
}

// DecodeAddressString decodes a base58 text bitcoin address. It returns the address, the network,
//   and an error if there was an issue.
func DecodeAddressString(address string) (Address, wire.BitcoinNet, error) {
	b, err := decodeAddress(address)
	if err != nil {
		return nil, MainNet, err
	}

	return DecodeAddressBytes(b)
}

// DecodeAddressBytes decodes a binary bitcoin address. It returns the address, the network, and an
//   error if there was an issue.
func DecodeAddressBytes(b []byte) (Address, wire.BitcoinNet, error) {
	switch b[0] {
	case typePKH:
		a, err := NewAddressPKH(b[1:])
		return a, MainNet, err
	case typeSH:
		a, err := NewAddressSH(b[1:])
		return a, MainNet, err
	case typeMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, MainNet, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/hashLength)
		for len(b) >= 0 {
			if len(b) < hashLength {
				return nil, MainNet, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:hashLength])
			b = b[hashLength:]
		}
		a, err := NewAddressMultiPKH(required, pkhs)
		return a, MainNet, err
	case typeRPH:
		a, err := NewAddressRPH(b[1:])
		return a, MainNet, err
	case typeTestPKH:
		a, err := NewAddressPKH(b[1:])
		return a, TestNet, err
	case typeTestSH:
		a, err := NewAddressSH(b[1:])
		return a, TestNet, err
	case typeTestMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, TestNet, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/hashLength)
		for len(b) >= 0 {
			if len(b) < hashLength {
				return nil, TestNet, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:hashLength])
			b = b[hashLength:]
		}
		a, err := NewAddressMultiPKH(required, pkhs)
		return a, TestNet, err
	case typeTestRPH:
		a, err := NewAddressRPH(b[1:])
		return a, TestNet, err
	}

	return nil, MainNet, ErrBadType
}

// DecodeNetMatches returns true if the decoded network id matches the specified network id.
// All test network ids decode as TestNet.
func DecodeNetMatches(decoded wire.BitcoinNet, desired wire.BitcoinNet) bool {
	switch decoded {
	case MainNet:
		return desired == MainNet
	case TestNet:
		return desired != MainNet
	}

	return false
}

/****************************************** PKH ***************************************************/
type AddressPKH struct {
	pkh []byte
}

// NewAddressPKH creates an address from a public key hash.
func NewAddressPKH(pkh []byte) (*AddressPKH, error) {
	if len(pkh) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressPKH{pkh: pkh}, nil
}

func (a *AddressPKH) PKH() []byte {
	return a.pkh
}

func (a *AddressPKH) String(network wire.BitcoinNet) string {
	return encodeAddress(a.Bytes(network))
}

func (a *AddressPKH) Bytes(network wire.BitcoinNet) []byte {
	var addressType byte

	// Add address type byte in front
	switch network {
	case MainNet:
		addressType = typePKH
	default:
		addressType = typeTestPKH
	}
	return append([]byte{addressType}, a.pkh...)
}

func (a *AddressPKH) Equal(other Address) bool {
	if other == nil {
		return false
	}
	otherPKH, ok := other.(*AddressPKH)
	if !ok {
		return false
	}
	return bytes.Equal(a.pkh, otherPKH.pkh)
}

/******************************************* SH ***************************************************/
type AddressSH struct {
	sh []byte
}

// NewAddressSH creates an address from a script hash.
func NewAddressSH(sh []byte) (*AddressSH, error) {
	if len(sh) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressSH{sh: sh}, nil
}

func (a *AddressSH) SH() []byte {
	return a.sh
}

func (a *AddressSH) String(network wire.BitcoinNet) string {
	return encodeAddress(a.Bytes(network))
}

func (a *AddressSH) Bytes(network wire.BitcoinNet) []byte {
	var addressType byte

	// Add address type byte in front
	switch network {
	case MainNet:
		addressType = typeSH
	default:
		addressType = typeTestSH
	}
	return append([]byte{addressType}, a.sh...)
}

func (a *AddressSH) Equal(other Address) bool {
	if other == nil {
		return false
	}
	otherSH, ok := other.(*AddressSH)
	if !ok {
		return false
	}
	return bytes.Equal(a.sh, otherSH.sh)
}

/**************************************** MultiPKH ************************************************/
type AddressMultiPKH struct {
	pkhs     [][]byte
	required uint16
}

// NewAddressMultiPKH creates an address from a required signature count and some public key hashes.
func NewAddressMultiPKH(required uint16, pkhs [][]byte) (*AddressMultiPKH, error) {
	for _, pkh := range pkhs {
		if len(pkh) != hashLength {
			return nil, ErrBadHashLength
		}
	}
	return &AddressMultiPKH{pkhs: pkhs, required: required}, nil
}

func (a *AddressMultiPKH) PKHs() []byte {
	b := make([]byte, 0, len(a.pkhs)*hashLength)
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}
	return b
}

func (a *AddressMultiPKH) String(network wire.BitcoinNet) string {
	return encodeAddress(a.Bytes(network))
}

func (a *AddressMultiPKH) Bytes(network wire.BitcoinNet) []byte {
	b := make([]byte, 0, 3+(len(a.pkhs)*hashLength))

	var addressType byte

	// Add address type byte in front
	switch network {
	case MainNet:
		addressType = typeMultiPKH
	default:
		addressType = typeTestMultiPKH
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

	return b
}

func (a *AddressMultiPKH) Equal(other Address) bool {
	if other == nil {
		return false
	}
	otherMultiPKH, ok := other.(*AddressMultiPKH)
	if !ok {
		return false
	}

	if len(a.pkhs) != len(otherMultiPKH.pkhs) {
		return false
	}

	for i, pkh := range a.pkhs {
		if !bytes.Equal(pkh, otherMultiPKH.pkhs[i]) {
			return false
		}
	}
	return true
}

/***************************************** RPH ************************************************/
type AddressRPH struct {
	rph []byte
}

// NewAddressRPH creates an address from an R puzzle hash.
func NewAddressRPH(rph []byte) (*AddressRPH, error) {
	if len(rph) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressRPH{rph: rph}, nil
}

func (a *AddressRPH) RPH() []byte {
	return a.rph
}

func (a *AddressRPH) String(network wire.BitcoinNet) string {
	return encodeAddress(a.Bytes(network))
}

func (a *AddressRPH) Bytes(network wire.BitcoinNet) []byte {
	var addressType byte

	// Add address type byte in front
	switch network {
	case MainNet:
		addressType = typeRPH
	default:
		addressType = typeTestRPH
	}
	return append([]byte{addressType}, a.rph...)
}

func (a *AddressRPH) Equal(other Address) bool {
	if other == nil {
		return false
	}
	otherRPH, ok := other.(*AddressRPH)
	if !ok {
		return false
	}
	return bytes.Equal(a.rph, otherRPH.rph)
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
