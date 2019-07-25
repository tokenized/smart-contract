package bitcoin

import (
	"bytes"
	"errors"
)

// AddressFromUnlockingScript returns the address associated with the specified unlocking script.
func AddressFromUnlockingScript(unlockingScript []byte) (Address, error) {
	if len(unlockingScript) < 2 {
		return nil, ErrUnknownScriptTemplate
	}

	buf := bytes.NewBuffer(unlockingScript)

	// First push
	pushSize, err := ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	firstPush := make([]byte, pushSize)
	_, err = buf.Read(firstPush)
	if err != nil {
		return nil, err
	}

	if buf.Len() == 0 {
		return nil, ErrUnknownScriptTemplate
	}

	// Second push
	pushSize, err = ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	secondPush := make([]byte, pushSize)
	_, err = buf.Read(firstPush)
	if err != nil {
		return nil, err
	}

	if isSignature(firstPush) && isPublicKey(secondPush) {
		// PKH
		// <Signature> <PublicKey>
		return NewAddressPKH(Hash160(secondPush))
	}

	if isPublicKey(firstPush) && isSignature(secondPush) {
		// RPH
		// <PublicKey> <Signature>
		rValue, err := signatureRValue(secondPush)
		if err != nil {
			return nil, err
		}
		return NewAddressPKH(Hash160(rValue))
	}

	// TODO MultiPKH
	return nil, ErrUnknownScriptTemplate
}

// PublicKeyFromUnlockingScript returns the serialized compressed public key from the unlocking
//   script if there is one.
// It only works for P2PKH and P2RPH unlocking scripts.
func PublicKeyFromUnlockingScript(unlockingScript []byte) ([]byte, error) {
	if len(unlockingScript) < 2 {
		return nil, ErrUnknownScriptTemplate
	}

	buf := bytes.NewBuffer(unlockingScript)

	// First push
	pushSize, err := ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	firstPush := make([]byte, pushSize)
	_, err = buf.Read(firstPush)
	if err != nil {
		return nil, err
	}

	if isPublicKey(firstPush) {
		return firstPush, nil
	}

	if buf.Len() == 0 {
		return nil, ErrUnknownScriptTemplate
	}

	// Second push
	pushSize, err = ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	secondPush := make([]byte, pushSize)
	_, err = buf.Read(firstPush)
	if err != nil {
		return nil, err
	}

	if isPublicKey(secondPush) {
		return secondPush, nil
	}

	return nil, ErrUnknownScriptTemplate
}

// isSignature returns true if the data is an encoded signature.
func isSignature(b []byte) bool {
	return len(b) > 40 && b[1] == 0x30 // compound header byte
}

// isPublicKey returns true if the data is an encoded and compressed public key.
func isPublicKey(b []byte) bool {
	return len(b) == 33 && (b[0] == 0x02 || b[0] == 0x03)
}

// signatureRValue returns the r value of the signature.
func signatureRValue(b []byte) ([]byte, error) {
	if len(b) < 40 {
		return nil, errors.New("Invalid signature length")
	}
	length := b[0]
	header := b[1]
	intHeader := b[2]
	rLength := b[3]

	if length > 4+rLength && header == 0x30 && intHeader == 0x02 && len(b) > int(4+rLength) {
		return b[4 : 4+rLength], nil
	}

	return nil, errors.New("Invalid signature encoding")
}
