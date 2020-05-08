package bitcoin

import (
	"crypto/sha256"

	"golang.org/x/crypto/ripemd160"
)

// Ripemd160 returns the RIPEMD (RIPE Message Digest) of the input.
//
// This is a wrapper for easy access to a chosen implementation.
//
// See https://en.wikipedia.org/wiki/RIPEMD
func Ripemd160(b []byte) []byte {
	hasher := ripemd160.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}

// Sha256 returns the SHA256 (Secure Hash Algorithm) of the input.
//
// This is a wrapper for easy access to a chosen implementation.
//
// See https://en.wikipedia.org/wiki/SHA-2
func Sha256(b []byte) []byte {
	result := sha256.Sum256(b)
	return result[:]
}

// Hash160 returns the Ripemd160(SHA256(input)) of the input.
//
// This is a wrapper for easy access to a chosen implementation.
func Hash160(b []byte) []byte {
	return Ripemd160(Sha256(b))
}

// DoubleSha256 performs a double Sha256 hash on the bytes.
func DoubleSha256(b []byte) []byte {
	return Sha256(Sha256(b))
}
