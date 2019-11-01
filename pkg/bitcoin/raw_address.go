package bitcoin

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	ScriptTypePKH      = 0x20 // Public Key Hash
	ScriptTypeSH       = 0x21 // Script Hash
	ScriptTypeMultiPKH = 0x22 // Multi-PKH
	ScriptTypeRPH      = 0x23 // RPH

	ScriptHashLength = 20 // Length of standard public key, script, and R hashes RIPEMD(SHA256())
)

// RawAddress represents a bitcoin address in raw format, with no check sum or encoding.
// It represents a "script template" for common locking and unlocking scripts.
// It enables parsing and creating of common locking and unlocking scripts as well as identifying
//   participants involved in the scripts via public key hashes and other hashes.
type RawAddress struct {
	scriptType byte
	data       []byte
}

// DecodeRawAddress decodes a binary raw address. It returns an error if there was an issue.
func DecodeRawAddress(b []byte) (RawAddress, error) {
	var result RawAddress
	err := result.Decode(b)
	return result, err
}

// Decode decodes a binary raw address. It returns an error if there was an issue.
func (ra *RawAddress) Decode(b []byte) error {
	switch b[0] {
	// Public Key Hash
	case AddressTypeMainPKH:
		fallthrough
	case AddressTypeTestPKH:
		fallthrough
	case ScriptTypePKH:
		return ra.SetPKH(b[1:])

	// Script Hash
	case AddressTypeMainSH:
		fallthrough
	case AddressTypeTestSH:
		fallthrough
	case ScriptTypeSH:
		return ra.SetSH(b[1:])

	// Multiple Public Key Hash
	case AddressTypeMainMultiPKH:
		fallthrough
	case AddressTypeTestMultiPKH:
		fallthrough
	case ScriptTypeMultiPKH:
		ra.scriptType = ScriptTypeMultiPKH
		ra.data = b[1:]

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
		return nil

	// R Puzzle Hash
	case AddressTypeMainRPH:
		fallthrough
	case AddressTypeTestRPH:
		fallthrough
	case ScriptTypeRPH:
		return ra.SetRPH(b[1:])
	}

	return ErrBadType
}

// Deserialize reads a binary raw address. It returns an error if there was an issue.
func (ra *RawAddress) Deserialize(buf *bytes.Reader) error {
	t, err := buf.ReadByte()
	if err != nil {
		return err
	}

	switch t {
	// Public Key Hash
	case AddressTypeMainPKH:
		fallthrough
	case AddressTypeTestPKH:
		fallthrough
	case ScriptTypePKH:
		pkh := make([]byte, ScriptHashLength)
		if _, err := buf.Read(pkh); err != nil {
			return err
		}
		return ra.SetPKH(pkh)

	// Script Hash
	case AddressTypeMainSH:
		fallthrough
	case AddressTypeTestSH:
		fallthrough
	case ScriptTypeSH:
		sh := make([]byte, ScriptHashLength)
		if _, err := buf.Read(sh); err != nil {
			return err
		}
		return ra.SetSH(sh)

	// Multiple Public Key Hash
	case AddressTypeMainMultiPKH:
		fallthrough
	case AddressTypeTestMultiPKH:
		fallthrough
	case ScriptTypeMultiPKH:
		// Parse required count
		var required uint16
		if err := binary.Read(buf, binary.LittleEndian, &required); err != nil {
			return err
		}
		// Parse hash count
		var count uint16
		if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
			return err
		}
		pkhs := make([][]byte, 0, count)
		for i := uint16(0); i < count; i++ {
			pkh := make([]byte, ScriptHashLength)
			if _, err := buf.Read(pkh); err != nil {
				return err
			}
			pkhs = append(pkhs, pkh)
		}
		return ra.SetMultiPKH(required, pkhs)

	// R Puzzle Hash
	case AddressTypeMainRPH:
		fallthrough
	case AddressTypeTestRPH:
		fallthrough
	case ScriptTypeRPH:
		rph := make([]byte, ScriptHashLength)
		if _, err := buf.Read(rph); err != nil {
			return err
		}
		return ra.SetRPH(rph)
	}

	return ErrBadType
}

// NewRawAddressFromAddress creates a RawAddress from an Address.
func NewRawAddressFromAddress(a Address) RawAddress {
	result := RawAddress{data: a.data}

	switch a.addressType {
	case AddressTypeMainPKH:
		fallthrough
	case AddressTypeTestPKH:
		result.scriptType = ScriptTypePKH
	case AddressTypeMainSH:
		fallthrough
	case AddressTypeTestSH:
		result.scriptType = ScriptTypeSH
	case AddressTypeMainMultiPKH:
		fallthrough
	case AddressTypeTestMultiPKH:
		result.scriptType = ScriptTypeMultiPKH
	case AddressTypeMainRPH:
		fallthrough
	case AddressTypeTestRPH:
		result.scriptType = ScriptTypeRPH
	}

	return result
}

/****************************************** PKH ***************************************************/

// NewRawAddressPKH creates an address from a public key hash.
func NewRawAddressPKH(pkh []byte) (RawAddress, error) {
	var result RawAddress
	err := result.SetPKH(pkh)
	return result, err
}

// SetPKH sets the type as ScriptTypePKH and sets the data to the specified Public Key Hash.
func (ra *RawAddress) SetPKH(pkh []byte) error {
	if len(pkh) != ScriptHashLength {
		return ErrBadScriptHashLength
	}

	ra.scriptType = ScriptTypePKH
	ra.data = pkh
	return nil
}

/******************************************* SH ***************************************************/

// NewRawAddressSH creates an address from a script hash.
func NewRawAddressSH(sh []byte) (RawAddress, error) {
	var result RawAddress
	err := result.SetSH(sh)
	return result, err
}

// SetSH sets the type as ScriptTypeSH and sets the data to the specified Script Hash.
func (ra *RawAddress) SetSH(sh []byte) error {
	if len(sh) != ScriptHashLength {
		return ErrBadScriptHashLength
	}

	ra.scriptType = ScriptTypeSH
	ra.data = sh
	return nil
}

/**************************************** MultiPKH ************************************************/

// NewRawAddressMultiPKH creates an address from multiple public key hashes.
func NewRawAddressMultiPKH(required uint16, pkhs [][]byte) (RawAddress, error) {
	var result RawAddress
	err := result.SetMultiPKH(required, pkhs)
	return result, err
}

// SetMultiPKH sets the type as ScriptTypeMultiPKH and puts the required count and Public Key Hashes into data.
func (ra *RawAddress) SetMultiPKH(required uint16, pkhs [][]byte) error {
	ra.scriptType = ScriptTypeMultiPKH
	buf := bytes.NewBuffer(make([]byte, 0, 4+(len(pkhs)*ScriptHashLength)))
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
	ra.data = buf.Bytes()
	return nil
}

/******************************************** RPH *************************************************/

// NewRawAddressRPH creates an address from a R puzzle hash.
func NewRawAddressRPH(rph []byte) (RawAddress, error) {
	var result RawAddress
	err := result.SetRPH(rph)
	return result, err
}

// SetRPH sets the type as ScriptTypeRPH and sets the data to the specified R Puzzle Hash.
func (ra *RawAddress) SetRPH(rph []byte) error {
	if len(rph) != ScriptHashLength {
		return ErrBadScriptHashLength
	}
	ra.scriptType = ScriptTypeRPH
	ra.data = rph
	return nil
}

/***************************************** Common *************************************************/

// Type returns the script type of the address.
func (ra RawAddress) Type() byte {
	return ra.scriptType
}

// IsSpendable returns true if the address produces a locking script that can be unlocked.
func (ra RawAddress) IsSpendable() bool {
	// TODO Full locking and unlocking support only available for P2PKH.
	return !ra.IsEmpty() && (ra.scriptType == ScriptTypePKH)
}

// Bytes returns the byte encoded format of the address.
func (ra RawAddress) Bytes() []byte {
	if len(ra.data) == 0 {
		return nil
	}
	return append([]byte{ra.scriptType}, ra.data...)
}

func (ra RawAddress) Equal(other RawAddress) bool {
	return ra.scriptType == other.scriptType && bytes.Equal(ra.data, other.data)
}

// IsEmpty returns true if the address does not have a value set.
func (ra RawAddress) IsEmpty() bool {
	return len(ra.data) == 0
}

func (ra RawAddress) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(ra.scriptType); err != nil {
		return err
	}
	if _, err := buf.Write(ra.data); err != nil {
		return err
	}
	return nil
}

// Hash returns the hash corresponding to the address.
func (ra *RawAddress) Hash() (*Hash20, error) {
	switch ra.scriptType {
	case ScriptTypePKH:
		return NewHash20(ra.data)
	case ScriptTypeSH:
		return NewHash20(ra.data)
	case ScriptTypeMultiPKH:
		return NewHash20(Hash160(ra.Bytes()))
	case ScriptTypeRPH:
		return NewHash20(ra.data)
	}
	return nil, ErrUnknownScriptTemplate
}

// MarshalJSON converts to json.
func (ra *RawAddress) MarshalJSON() ([]byte, error) {
	if len(ra.data) == 0 {
		return []byte("\"\""), nil
	}
	return []byte("\"" + hex.EncodeToString(ra.Bytes()) + "\""), nil
}

// UnmarshalJSON converts from json.
func (ra *RawAddress) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("Too short for RawAddress hex data : %d", len(data))
	}

	if len(data) == 2 {
		// Empty raw address
		ra.scriptType = 0
		ra.data = nil
		return nil
	}

	// Decode hex and remove double quotes.
	raw, err := hex.DecodeString(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}

	// Decode into raw address
	return ra.Decode(raw)
}

// Scan converts from a database column.
func (ra *RawAddress) Scan(data interface{}) error {
	if data == nil {
		// Empty raw address
		ra.scriptType = 0
		ra.data = nil
		return nil
	}

	b, ok := data.([]byte)
	if !ok {
		return errors.New("RawAddress db column not bytes")
	}

	if len(b) == 0 {
		// Empty raw address
		ra.scriptType = 0
		ra.data = nil
		return nil
	}

	// Copy byte slice because it will be wiped out by the database after this call.
	c := make([]byte, len(b))
	copy(c, b)

	// Decode into raw address
	return ra.Decode(c)
}
