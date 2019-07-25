package bitcoin

import (
	"encoding/base64"

	"github.com/btcsuite/btcutil/base58"
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
