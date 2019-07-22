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

	// ScriptAddress returns raw address data without type.
	// TODO Remove when protocol fully upgraded for address types.
	ScriptAddress() []byte

	// LockingScript returns the bitcoin output(locking) script for paying to the address.
	LockingScript() []byte
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
		a, err := AddressPKHFromBytes(b[1:])
		return a, MainNet, err
	case typeSH:
		a, err := AddressSHFromBytes(b[1:])
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
		a, err := AddressMultiPKHFromBytes(required, pkhs)
		return a, MainNet, err
	case typeRPH:
		a, err := AddressRPHFromBytes(b[1:])
		return a, MainNet, err
	case typeTestPKH:
		a, err := AddressPKHFromBytes(b[1:])
		return a, TestNet, err
	case typeTestSH:
		a, err := AddressSHFromBytes(b[1:])
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
		a, err := AddressMultiPKHFromBytes(required, pkhs)
		return a, TestNet, err
	case typeTestRPH:
		a, err := AddressRPHFromBytes(b[1:])
		return a, TestNet, err
	}

	return nil, MainNet, ErrBadType
}

/****************************************** PKH ***************************************************/
type AddressPKH struct {
	pkh []byte
}

// AddressPKHFromBytes creates an address from a public key hash.
func AddressPKHFromBytes(pkh []byte) (*AddressPKH, error) {
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

func (a *AddressPKH) ScriptAddress() []byte {
	return a.pkh
}

/******************************************* SH ***************************************************/
type AddressSH struct {
	sh []byte
}

// AddressSHFromBytes creates an address from a script hash.
func AddressSHFromBytes(sh []byte) (*AddressSH, error) {
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

func (a *AddressSH) ScriptAddress() []byte {
	return nil //errors.New("SH addresses don't have script addresses")
}

/**************************************** MultiPKH ************************************************/
type AddressMultiPKH struct {
	pkhs     [][]byte
	required uint16
}

// AddressMultiPKHFromBytes creates an address from a required signature count and some public key hashes.
func AddressMultiPKHFromBytes(required uint16, pkhs [][]byte) (*AddressMultiPKH, error) {
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

func (a *AddressMultiPKH) ScriptAddress() []byte {
	return nil //errors.New("MultiPKH addresses don't have script addresses")
}

/***************************************** RPH ************************************************/
type AddressRPH struct {
	rph []byte
}

// AddressRPHFromBytes creates an address from an R puzzle hash.
func AddressRPHFromBytes(rph []byte) (*AddressRPH, error) {
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

func (a *AddressRPH) ScriptAddress() []byte {
	return nil //errors.New("R Puzzle addresses don't have script addresses")
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
