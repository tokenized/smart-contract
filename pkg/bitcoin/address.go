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
	typeMainPKH      = 0x00 // Public Key Hash
	typeMainSH       = 0x05 // Script Hash
	typeMainMultiPKH = 0x10 // Multi-PKH - Experimental value. Not standard
	typeMainRPH      = 0x20 // RPH - Experimental value. Not standard

	typeTestPKH      = 0x6f // Testnet Public Key Hash
	typeTestSH       = 0xc4 // Testnet Script Hash
	typeTestMultiPKH = 0xd0 // Multi-PKH - Experimental value. Not standard
	typeTestRPH      = 0xe0 // RPH - Experimental value. Not standard

	typeIntPKH      = 0x30 // Public Key Hash
	typeIntSH       = 0x31 // Script Hash
	typeIntMultiPKH = 0x32 // Multi-PKH
	typeIntRPH      = 0x33 // RPH

	hashLength = 20 // Length of public key, script, and R hashes
)

type Address interface {
	// String returns the type and address data followed by a checksum encoded with Base58.
	// Panics if called for address that was created with IntNet.
	String() string

	// Network returns the network id for the address.
	Network() wire.BitcoinNet

	// SetNetwork changes the network of the address.
	// This should only be used to change from IntNet to the correct network.
	SetNetwork(net wire.BitcoinNet)

	// Bytes returns the non-network specific type followed by the address data.
	Bytes() []byte

	// LockingScript returns the bitcoin output(locking) script for paying to the address.
	LockingScript() []byte

	// Equal returns true if the address parameter has the same value.
	Equal(Address) bool
}

// DecodeAddressString decodes a base58 text bitcoin address. It returns the address, and an error
//   if there was an issue.
func DecodeAddressString(address string) (Address, error) {
	b, err := decodeAddress(address)
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case typeMainPKH:
		return NewAddressPKH(b[1:], MainNet)
	case typeMainSH:
		return NewAddressSH(b[1:], MainNet)
	case typeMainMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/hashLength)
		for len(b) >= 0 {
			if len(b) < hashLength {
				return nil, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:hashLength])
			b = b[hashLength:]
		}
		return NewAddressMultiPKH(required, pkhs, MainNet)
	case typeMainRPH:
		return NewAddressRPH(b[1:], MainNet)
	case typeTestPKH:
		return NewAddressPKH(b[1:], TestNet)
	case typeTestSH:
		return NewAddressSH(b[1:], TestNet)
	case typeTestMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/hashLength)
		for len(b) >= 0 {
			if len(b) < hashLength {
				return nil, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:hashLength])
			b = b[hashLength:]
		}
		return NewAddressMultiPKH(required, pkhs, TestNet)
	case typeTestRPH:
		return NewAddressRPH(b[1:], TestNet)
	}

	return nil, ErrBadType
}

// DecodeAddressBytes decodes a binary bitcoin address. It returns the address, and an error if
//   there was an issue.
func DecodeAddressBytes(b []byte, net wire.BitcoinNet) (Address, error) {
	switch b[0] {
	case typeIntPKH:
		return NewAddressPKH(b[1:], net)
	case typeIntSH:
		return NewAddressSH(b[1:], net)
	case typeIntMultiPKH:
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:2])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return nil, err
		}
		b = b[2:] // remove required
		pkhs := make([][]byte, 0, len(b)/hashLength)
		for len(b) >= 0 {
			if len(b) < hashLength {
				return nil, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:hashLength])
			b = b[hashLength:]
		}
		return NewAddressMultiPKH(required, pkhs, net)
	case typeIntRPH:
		return NewAddressRPH(b[1:], net)
	}

	return nil, ErrBadType
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
	net wire.BitcoinNet
}

// NewAddressPKH creates an address from a public key hash.
func NewAddressPKH(pkh []byte, net wire.BitcoinNet) (*AddressPKH, error) {
	if len(pkh) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressPKH{pkh: pkh, net: net}, nil
}

func (a *AddressPKH) PKH() []byte {
	return a.pkh
}

// String returns the type and address data followed by a checksum encoded with Base58.
// Panics if called for address that was created with IntNet.
func (a *AddressPKH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = typeMainPKH
	case IntNet:
		panic("Internal address type")
	default:
		addressType = typeTestPKH
	}
	return encodeAddress(append([]byte{addressType}, a.pkh...))
}

// Network returns the network id for the address.
func (a *AddressPKH) Network() wire.BitcoinNet {
	return a.net
}

// SetNetwork changes the network of the address.
// This should only be used to change from IntNet to the correct network.
func (a *AddressPKH) SetNetwork(net wire.BitcoinNet) {
	a.net = net
}

func (a *AddressPKH) Bytes() []byte {
	return append([]byte{typeIntPKH}, a.pkh...)
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
	sh  []byte
	net wire.BitcoinNet
}

// NewAddressSH creates an address from a script hash.
func NewAddressSH(sh []byte, net wire.BitcoinNet) (*AddressSH, error) {
	if len(sh) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressSH{sh: sh, net: net}, nil
}

func (a *AddressSH) SH() []byte {
	return a.sh
}

// String returns the type and address data followed by a checksum encoded with Base58.
// Panics if called for address that was created with IntNet.
func (a *AddressSH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = typeMainSH
	case IntNet:
		panic("Internal address type")
	default:
		addressType = typeTestSH
	}
	return encodeAddress(append([]byte{addressType}, a.sh...))
}

// Network returns the network id for the address.
func (a *AddressSH) Network() wire.BitcoinNet {
	return a.net
}

// SetNetwork changes the network of the address.
// This should only be used to change from IntNet to the correct network.
func (a *AddressSH) SetNetwork(net wire.BitcoinNet) {
	a.net = net
}

func (a *AddressSH) Bytes() []byte {
	return append([]byte{typeIntSH}, a.sh...)
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
	net      wire.BitcoinNet
}

// NewAddressMultiPKH creates an address from a required signature count and some public key hashes.
func NewAddressMultiPKH(required uint16, pkhs [][]byte, net wire.BitcoinNet) (*AddressMultiPKH, error) {
	for _, pkh := range pkhs {
		if len(pkh) != hashLength {
			return nil, ErrBadHashLength
		}
	}
	return &AddressMultiPKH{pkhs: pkhs, required: required, net: net}, nil
}

func (a *AddressMultiPKH) PKHs() []byte {
	b := make([]byte, 0, len(a.pkhs)*hashLength)
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}
	return b
}

// String returns the type and address data followed by a checksum encoded with Base58.
// Panics if called for address that was created with IntNet.
func (a *AddressMultiPKH) String() string {
	b := make([]byte, 0, 3+(len(a.pkhs)*hashLength))

	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = typeMainMultiPKH
	case IntNet:
		panic("Internal address type")
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

	return encodeAddress(b)
}

// Network returns the network id for the address.
func (a *AddressMultiPKH) Network() wire.BitcoinNet {
	return a.net
}

// SetNetwork changes the network of the address.
// This should only be used to change from IntNet to the correct network.
func (a *AddressMultiPKH) SetNetwork(net wire.BitcoinNet) {
	a.net = net
}

func (a *AddressMultiPKH) Bytes() []byte {
	b := make([]byte, 0, 3+(len(a.pkhs)*hashLength))

	b = append(b, byte(typeIntMultiPKH))

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
	net wire.BitcoinNet
}

// NewAddressRPH creates an address from an R puzzle hash.
func NewAddressRPH(rph []byte, net wire.BitcoinNet) (*AddressRPH, error) {
	if len(rph) != hashLength {
		return nil, ErrBadHashLength
	}
	return &AddressRPH{rph: rph, net: net}, nil
}

func (a *AddressRPH) RPH() []byte {
	return a.rph
}

// String returns the type and address data followed by a checksum encoded with Base58.
// Panics if called for address that was created with IntNet.
func (a *AddressRPH) String() string {
	var addressType byte

	// Add address type byte in front
	switch a.net {
	case MainNet:
		addressType = typeMainRPH
	case IntNet:
		panic("Internal address type")
	default:
		addressType = typeTestRPH
	}
	return encodeAddress(append([]byte{addressType}, a.rph...))
}

// Network returns the network id for the address.
func (a *AddressRPH) Network() wire.BitcoinNet {
	return a.net
}

// SetNetwork changes the network of the address.
// This should only be used to change from IntNet to the correct network.
func (a *AddressRPH) SetNetwork(net wire.BitcoinNet) {
	a.net = net
}

func (a *AddressRPH) Bytes() []byte {
	return append([]byte{typeMainRPH}, a.rph...)
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
