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

type ScriptTemplate interface {
	// Bytes returns the non-network specific type followed by the address data.
	Bytes() []byte

	// LockingScript returns the bitcoin output(locking) script for paying to the address.
	LockingScript() []byte

	// Equal returns true if the address parameter has the same value.
	Equal(ScriptTemplate) bool
}

// DecodeScriptTemplate decodes a binary bitcoin script template. It returns the script
//   template, and an error if there was an issue.
func DecodeScriptTemplate(b []byte) (ScriptTemplate, error) {
	switch b[0] {
	case scriptTypePKH:
		return NewScriptTemplatePKH(b[1:])
	case scriptTypeSH:
		return NewScriptTemplateSH(b[1:])
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
				return nil, ErrBadHashLength
			}
			pkhs = append(pkhs, b[:scriptHashLength])
			b = b[scriptHashLength:]
		}
		return NewScriptTemplateMultiPKH(required, pkhs)
	case scriptTypeRPH:
		return NewScriptTemplateRPH(b[1:])
	}

	return nil, ErrBadType
}

/****************************************** PKH ***************************************************/
type ScriptTemplatePKH struct {
	pkh []byte
}

// NewScriptTemplatePKH creates an address from a public key hash.
func NewScriptTemplatePKH(pkh []byte) (*ScriptTemplatePKH, error) {
	if len(pkh) != scriptHashLength {
		return nil, ErrBadHashLength
	}
	return &ScriptTemplatePKH{pkh: pkh}, nil
}

func (a *ScriptTemplatePKH) PKH() []byte {
	return a.pkh
}

func (a *ScriptTemplatePKH) Bytes() []byte {
	return append([]byte{scriptTypePKH}, a.pkh...)
}

func (a *ScriptTemplatePKH) Equal(other ScriptTemplate) bool {
	if other == nil {
		return false
	}
	otherPKH, ok := other.(*ScriptTemplatePKH)
	if !ok {
		return false
	}
	return bytes.Equal(a.pkh, otherPKH.pkh)
}

/******************************************* SH ***************************************************/
type ScriptTemplateSH struct {
	sh []byte
}

// NewScriptTemplateSH creates an address from a script hash.
func NewScriptTemplateSH(sh []byte) (*ScriptTemplateSH, error) {
	if len(sh) != scriptHashLength {
		return nil, ErrBadHashLength
	}
	return &ScriptTemplateSH{sh: sh}, nil
}

func (a *ScriptTemplateSH) SH() []byte {
	return a.sh
}

func (a *ScriptTemplateSH) Bytes() []byte {
	return append([]byte{scriptTypeSH}, a.sh...)
}

func (a *ScriptTemplateSH) Equal(other ScriptTemplate) bool {
	if other == nil {
		return false
	}
	otherSH, ok := other.(*ScriptTemplateSH)
	if !ok {
		return false
	}
	return bytes.Equal(a.sh, otherSH.sh)
}

/**************************************** MultiPKH ************************************************/
type ScriptTemplateMultiPKH struct {
	pkhs     [][]byte
	required uint16
}

// NewScriptTemplateMultiPKH creates an address from a required signature count and some public key hashes.
func NewScriptTemplateMultiPKH(required uint16, pkhs [][]byte) (*ScriptTemplateMultiPKH, error) {
	for _, pkh := range pkhs {
		if len(pkh) != scriptHashLength {
			return nil, ErrBadHashLength
		}
	}
	return &ScriptTemplateMultiPKH{pkhs: pkhs, required: required}, nil
}

func (a *ScriptTemplateMultiPKH) PKHs() []byte {
	b := make([]byte, 0, len(a.pkhs)*scriptHashLength)
	for _, pkh := range a.pkhs {
		b = append(b, pkh...)
	}
	return b
}

func (a *ScriptTemplateMultiPKH) Bytes() []byte {
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

func (a *ScriptTemplateMultiPKH) Equal(other ScriptTemplate) bool {
	if other == nil {
		return false
	}
	otherMultiPKH, ok := other.(*ScriptTemplateMultiPKH)
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
type ScriptTemplateRPH struct {
	rph []byte
}

// NewScriptTemplateRPH creates an address from an R puzzle hash.
func NewScriptTemplateRPH(rph []byte) (*ScriptTemplateRPH, error) {
	if len(rph) != scriptHashLength {
		return nil, ErrBadHashLength
	}
	return &ScriptTemplateRPH{rph: rph}, nil
}

func (a *ScriptTemplateRPH) RPH() []byte {
	return a.rph
}

func (a *ScriptTemplateRPH) Bytes() []byte {
	return append([]byte{scriptTypeRPH}, a.rph...)
}

func (a *ScriptTemplateRPH) Equal(other ScriptTemplate) bool {
	if other == nil {
		return false
	}
	otherRPH, ok := other.(*ScriptTemplateRPH)
	if !ok {
		return false
	}
	return bytes.Equal(a.rph, otherRPH.rph)
}
