package bitcoin

import (
	"bytes"
	"errors"
)

// AddressFromUnlockingScript returns the address associated with the specified unlocking script.
func AddressFromUnlockingScript(unlockingScript []byte, net Network) (Address, error) {
	ra, err := RawAddressFromUnlockingScript(unlockingScript)
	if err != nil {
		return Address{}, err
	}
	return NewAddressFromRawAddress(ra, net), nil
}

// RawAddressFromUnlockingScript returns the raw address associated with the specified unlocking
//   script.
func RawAddressFromUnlockingScript(unlockingScript []byte) (RawAddress, error) {
	var result RawAddress

	if len(unlockingScript) < 2 {
		return result, ErrUnknownScriptTemplate
	}

	buf := bytes.NewReader(unlockingScript)

	// TODO Implement checking for multi-pkh

	// First push
	_, firstPush, err := ParsePushDataScript(buf)
	if err != nil {
		return result, err
	}

	if buf.Len() == 0 && isSignature(firstPush) {
		// Can't determine public key for address from signature along. Locking script required.
		return result, ErrNotEnoughData
	}

	if len(firstPush) == 0 {
		return result, ErrUnknownScriptTemplate
	}

	// Second push
	_, secondPush, err := ParsePushDataScript(buf)
	if err != nil {
		return result, err
	}

	if len(secondPush) == 0 {
		return result, ErrUnknownScriptTemplate
	}

	if isSignature(firstPush) && isPublicKey(secondPush) {
		// PKH
		// <Signature> <PublicKey>
		result.SetPKH(Hash160(secondPush))
		return result, nil
	}

	if isPublicKey(firstPush) && isSignature(secondPush) {
		// RPH
		// <PublicKey> <Signature>
		rValue, err := signatureRValue(secondPush)
		if err != nil {
			return result, err
		}
		result.SetRPH(Hash160(rValue))
		return result, nil
	}

	return result, ErrUnknownScriptTemplate
}

// PublicKeyFromUnlockingScript returns the serialized compressed public key from the unlocking
//   script if there is one.
// It only works for P2PKH and P2RPH unlocking scripts.
func PublicKeyFromUnlockingScript(unlockingScript []byte) ([]byte, error) {
	if len(unlockingScript) < 2 {
		return nil, ErrUnknownScriptTemplate
	}

	buf := bytes.NewReader(unlockingScript)

	// First push
	_, firstPush, err := ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	if isPublicKey(firstPush) {
		return firstPush, nil
	}

	if buf.Len() == 0 {
		if isSignature(firstPush) {
			// Can't determine public key for address from signature along. Locking script required.
			return nil, ErrNotEnoughData
		}
		return nil, ErrUnknownScriptTemplate
	}

	// Second push
	_, secondPush, err := ParsePushDataScript(buf)
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
	return len(b) > 40 && b[0] == 0x30 // compound header byte
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
