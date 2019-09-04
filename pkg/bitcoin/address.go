package bitcoin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrBadScriptHashLength   = errors.New("Script hash has invalid length")
	ErrBadCheckSum           = errors.New("Address has bad checksum")
	ErrBadType               = errors.New("Address type unknown")
	ErrUnknownScriptTemplate = errors.New("Unknown script template")
)

const (
	AddressTypeMainPKH      = 0x00 // Public Key Hash
	AddressTypeMainSH       = 0x05 // Script Hash
	AddressTypeMainMultiPKH = 0x10 // Multi-PKH - Experimental value. Not standard
	AddressTypeMainRPH      = 0x20 // RPH - Experimental value. Not standard

	AddressTypeTestPKH      = 0x6f // Testnet Public Key Hash
	AddressTypeTestSH       = 0xc4 // Testnet Script Hash
	AddressTypeTestMultiPKH = 0xd0 // Multi-PKH - Experimental value. Not standard
	AddressTypeTestRPH      = 0xe0 // RPH - Experimental value. Not standard
)

type Address struct {
	addressType byte
	data        []byte
}

// DecodeAddress decodes a base58 text bitcoin address. It returns an error if there was an issue.
func DecodeAddress(address string) (Address, error) {
	var result Address
	err := result.Decode(address)
	return result, err
}

// Decode decodes a base58 text bitcoin address. It returns an error if there was an issue.
func (a *Address) Decode(address string) error {
	b, err := decodeAddress(address)
	if err != nil {
		return err
	}

	return a.decodeBytes(b)
}

// decodeAddressBytes decodes a binary address. It returns an error if there was an issue.
func (a *Address) decodeBytes(b []byte) error {
	if len(b) < 2 {
		return ErrBadType
	}

	switch b[0] {

	// MainNet
	case AddressTypeMainPKH:
		return a.SetPKH(b[1:], MainNet)
	case AddressTypeMainSH:
		return a.SetSH(b[1:], MainNet)
	case AddressTypeMainMultiPKH:
		a.data = b[1:]

		// Validate data
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:4])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return err
		}
		// Parse hash count
		var count uint16
		if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
			return err
		}
		b = b[4:] // remove counts
		for i := uint16(0); i < count; i++ {
			if len(b) < ScriptHashLength {
				return ErrBadScriptHashLength
			}
			b = b[ScriptHashLength:]
		}
		a.addressType = AddressTypeMainMultiPKH
		return nil
	case AddressTypeMainRPH:
		return a.SetRPH(b[1:], MainNet)

	// TestNet
	case AddressTypeTestPKH:
		return a.SetPKH(b[1:], TestNet)
	case AddressTypeTestSH:
		return a.SetSH(b[1:], TestNet)
	case AddressTypeTestMultiPKH:
		a.data = b[1:]

		// Validate data
		b = b[1:] // remove type
		// Parse required count
		buf := bytes.NewBuffer(b[:4])
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return err
		}
		// Parse hash count
		var count uint16
		if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
			return err
		}
		b = b[4:] // remove counts
		for i := uint16(0); i < count; i++ {
			if len(b) < ScriptHashLength {
				return ErrBadScriptHashLength
			}
			b = b[ScriptHashLength:]
		}
		a.addressType = AddressTypeTestMultiPKH
		return nil
	case AddressTypeTestRPH:
		return a.SetRPH(b[1:], TestNet)
	}

	return ErrBadType
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

// NewAddressFromRawAddress creates an Address from a RawAddress and a network.
func NewAddressFromRawAddress(ra RawAddress, net Network) Address {
	result := Address{data: ra.data}

	switch ra.scriptType {
	case ScriptTypePKH:
		if net == MainNet {
			result.addressType = AddressTypeMainPKH
		} else {
			result.addressType = AddressTypeTestPKH
		}
	case ScriptTypeSH:
		if net == MainNet {
			result.addressType = AddressTypeMainSH
		} else {
			result.addressType = AddressTypeTestSH
		}
	case ScriptTypeMultiPKH:
		if net == MainNet {
			result.addressType = AddressTypeMainMultiPKH
		} else {
			result.addressType = AddressTypeTestMultiPKH
		}
	case ScriptTypeRPH:
		if net == MainNet {
			result.addressType = AddressTypeMainRPH
		} else {
			result.addressType = AddressTypeTestRPH
		}
	}

	return result
}

/****************************************** PKH ***************************************************/

// NewAddressPKH creates an address from a public key hash.
func NewAddressPKH(pkh []byte, net Network) (Address, error) {
	var result Address
	err := result.SetPKH(pkh, net)
	return result, err
}

// SetPKH sets the Public Key Hash and script type of the address.
func (a *Address) SetPKH(pkh []byte, net Network) error {
	if len(pkh) != ScriptHashLength {
		return ErrBadScriptHashLength
	}

	if net == MainNet {
		a.addressType = AddressTypeMainPKH
	} else {
		a.addressType = AddressTypeTestPKH
	}

	a.data = pkh
	return nil
}

/****************************************** SH ***************************************************/

// NewAddressSH creates an address from a script hash.
func NewAddressSH(sh []byte, net Network) (Address, error) {
	var result Address
	err := result.SetSH(sh, net)
	return result, err
}

// SetSH sets the Script Hash and script type of the address.
func (a *Address) SetSH(sh []byte, net Network) error {
	if len(sh) != ScriptHashLength {
		return ErrBadScriptHashLength
	}

	if net == MainNet {
		a.addressType = AddressTypeMainSH
	} else {
		a.addressType = AddressTypeTestSH
	}

	a.data = sh
	return nil
}

/**************************************** MultiPKH ************************************************/

// NewAddressMultiPKH creates an address from multiple public key hashes.
func NewAddressMultiPKH(required uint16, pkhs [][]byte, net Network) (Address, error) {
	var result Address
	err := result.SetMultiPKH(required, pkhs, net)
	return result, err
}

// SetMultiPKH sets the Public Key Hashes and script type of the address.
func (a *Address) SetMultiPKH(required uint16, pkhs [][]byte, net Network) error {
	if net == MainNet {
		a.addressType = AddressTypeMainMultiPKH
	} else {
		a.addressType = AddressTypeTestMultiPKH
	}

	buf := bytes.NewBuffer(make([]byte, 0, 2+(len(pkhs)*ScriptHashLength)))
	if err := binary.Write(buf, binary.LittleEndian, required); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(pkhs))); err != nil {
		return err
	}
	for _, pkh := range pkhs {
		n, err := buf.Write(pkh)
		if err != nil {
			return err
		}
		if n != ScriptHashLength {
			return ErrBadScriptHashLength
		}
	}
	a.data = buf.Bytes()
	return nil
}

/****************************************** RPH ***************************************************/

// NewAddressRPH creates an address from a R puzzle hash.
func NewAddressRPH(rph []byte, net Network) (Address, error) {
	var result Address
	err := result.SetRPH(rph, net)
	return result, err
}

// SetRPH sets the R Puzzle Hash and script type of the address.
func (a *Address) SetRPH(rph []byte, net Network) error {
	if len(rph) != ScriptHashLength {
		return ErrBadScriptHashLength
	}

	if net == MainNet {
		a.addressType = AddressTypeMainRPH
	} else {
		a.addressType = AddressTypeTestRPH
	}

	a.data = rph
	return nil
}

/***************************************** Common *************************************************/

// String returns the type and address data followed by a checksum encoded with Base58.
func (a Address) String() string {
	return encodeAddress(append([]byte{a.addressType}, a.data...))
}

// Network returns the network id for the address.
func (a Address) Network() Network {
	switch a.addressType {
	case AddressTypeMainPKH:
	case AddressTypeMainSH:
	case AddressTypeMainMultiPKH:
	case AddressTypeMainRPH:
		return MainNet
	}
	return TestNet
}

// IsEmpty returns true if the address does not have a value set.
func (a Address) IsEmpty() bool {
	return len(a.data) == 0
}

// Hash returns the hash corresponding to the address.
func (a *Address) Hash() (*Hash20, error) {
	switch a.addressType {
	case AddressTypeMainPKH:
	case AddressTypeTestPKH:
	case AddressTypeMainSH:
	case AddressTypeTestSH:
	case AddressTypeMainRPH:
	case AddressTypeTestRPH:
		return NewHash20(a.data)
	case AddressTypeMainMultiPKH:
	case AddressTypeTestMultiPKH:
		return NewHash20(Hash160(a.data))
	}
	return nil, ErrUnknownScriptTemplate
}

// MarshalJSON converts to json.
func (a Address) MarshalJSON() ([]byte, error) {
	if len(a.data) == 0 {
		return []byte("\"\""), nil
	}
	return []byte("\"" + a.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (a *Address) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("Too short for Address data : %d", len(data))
	}

	if len(data) == 2 {
		// Empty address
		a.addressType = AddressTypeMainPKH
		a.data = nil
		return nil
	}

	return a.Decode(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (a *Address) Scan(data interface{}) error {
	s, ok := data.(string)
	if !ok {
		return errors.New("Address db column not bytes")
	}

	if len(s) == 0 {
		// Empty address
		a.addressType = AddressTypeMainPKH
		a.data = nil
		return nil
	}

	// Decode address
	return a.Decode(s)
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
