package bitcoin

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
)

var (
	ErrCheckHashInvalid = errors.New("Check Hash Invalid")
	ErrInvalidVersion   = errors.New("Invalid Version")
	ErrInvalidNetwork   = errors.New("Invalid Network")
)

// Base64 returns the Bas64 encoding of the input.
//
// See https://en.wikipedia.org/wiki/Base64
func Base64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// Base64Decode returns base 64 decodes the argument and returns the result.
func Base64Decode(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Base58 return the Base58 encoding of the input.
//
// See https://en.wikipedia.org/wiki/Base58
func Base58(b []byte) string {
	return base58.Encode(b)
}

// Base58Decode returns base 58 decodes the argument and returns the result.
func Base58Decode(s string) []byte {
	return base58.Decode(s)
}

// BIP0276Encode encodes a value with a specified prefix into a hex string.
func BIP0276Encode(net Network, prefix string, data []byte) string {
	result := prefix + ":"

	result += "01" // BIP-0276 Version

	// BIP-0276 Network
	switch net {
	case InvalidNet:
		result += "00"
	case MainNet:
		result += "01"
	case TestNet:
		result += "02"
	}

	result += hex.EncodeToString(data)

	// BIP-0276 Check Hash - Append first 4 bytes of double SHA256 of hash of preceding text
	check := DoubleSha256([]byte(result))
	return result + hex.EncodeToString(check[:4])
}

// BIP0276Decode decodes a value into the prefix and data into a hex string.
func BIP0276Decode(url string) (Network, string, []byte, error) {
	url = strings.TrimSpace(url)

	// Check Hash
	if len(url) <= 8 {
		return InvalidNet, "", nil, errors.New("Too Short")
	}
	hash := DoubleSha256([]byte(url[:len(url)-8]))
	check := hex.EncodeToString(hash[:4])
	if check != url[len(url)-8:] {
		return InvalidNet, "", nil, ErrCheckHashInvalid
	}

	parts := strings.Split(url, ":")

	if len(parts) != 2 {
		return InvalidNet, "", nil, errors.New("To many colons in xkey")
	}

	b, err := hex.DecodeString(parts[1])
	if err != nil {
		return InvalidNet, "", nil, errors.Wrap(err, "decode xkey hex")
	}

	// BIP-0276 Version
	if b[0] != 1 {
		return InvalidNet, "", nil,
			errors.Wrap(ErrInvalidVersion, fmt.Sprintf("Invalid BIP-0276 version : %x", b[0]))
	}
	b = b[1:] // Drop version

	// BIP-0276 Network
	var net Network
	switch b[0] {
	case 0:
		net = InvalidNet
	case 1:
		net = MainNet
	case 2:
		net = TestNet
	default:
		return InvalidNet, "", nil,
			errors.Wrap(ErrInvalidVersion, fmt.Sprintf("Invalid BIP-0276 network : %x", b[0]))
	}
	b = b[1:] // Drop network

	return net, parts[0], b, nil
}

// BIP0276Encode58 encodes a value with a specified prefix into a base58 string.
func BIP0276Encode58(net Network, prefix string, data []byte) string {
	fullData := make([]byte, 0, (len(data)*2)+6)

	fullData = append(fullData, 0x01) // BIP-0276 Version

	// BIP-0276 Network
	switch net {
	case InvalidNet:
		fullData = append(fullData, 0x00)
	case MainNet:
		fullData = append(fullData, 0x01)
	case TestNet:
		fullData = append(fullData, 0x02)
	}

	// BIP-0276 Data
	fullData = append(fullData, data...)

	// BIP-0276 Check Hash
	hexValue := BIP0276Encode(net, prefix, data)
	check, _ := hex.DecodeString(hexValue[len(hexValue)-8:])
	fullData = append(fullData, check...)

	return prefix + ":" + Base58(fullData)
}

// BIP0276Decode58 decodes a value into the prefix and data into a base58 string.
func BIP0276Decode58(url string) (Network, string, []byte, error) {
	url = strings.TrimSpace(url)

	parts := strings.Split(url, ":")

	if len(parts) != 2 {
		return InvalidNet, "", nil, errors.New("To many colons in xkey")
	}

	b := Base58Decode(parts[1])
	if len(b) == 0 {
		return InvalidNet, "", nil, errors.New("Failed to decode xkey base58")
	}

	// Check hash
	checkValue := parts[0] + ":" + hex.EncodeToString(b)
	if len(checkValue) <= 8 {
		return InvalidNet, "", nil, errors.New("Too Short")
	}
	hash := DoubleSha256([]byte(checkValue[:len(checkValue)-8]))
	check := hex.EncodeToString(hash[:4])
	if check != checkValue[len(checkValue)-8:] {
		return InvalidNet, "", nil, ErrCheckHashInvalid
	}

	// BIP-0276 Version
	if b[0] != 1 {
		return InvalidNet, "", nil,
			errors.Wrap(ErrInvalidVersion, fmt.Sprintf("Invalid BIP-0276 version : %x", b[0]))
	}
	b = b[1:] // Drop version

	// BIP-0276 Network
	var net Network
	switch b[0] {
	case 0:
		net = InvalidNet
	case 1:
		net = MainNet
	case 2:
		net = TestNet
	default:
		return InvalidNet, "", nil,
			errors.Wrap(ErrInvalidVersion, fmt.Sprintf("Invalid BIP-0276 network : %x", b[0]))
	}
	b = b[1:] // Drop network

	return net, parts[0], b, nil
}
