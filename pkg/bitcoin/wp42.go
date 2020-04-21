package bitcoin

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func GenerateSeedMessage() (Hash32, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return Hash32{}, errors.Wrap(err, "random")
	}

	var val big.Int
	val.SetBytes(b)

	// For added randomness
	now := time.Now()
	addVal := big.NewInt(now.UnixNano())

	val.Add(&val, addVal)

	// Extend to 32 bytes
	b = val.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}

	result, err := NewHash32(b)
	if err != nil {
		return Hash32{}, errors.Wrap(err, "new hash")
	}
	return *result, nil
}

// NextKey implements the WP42 method of deriving a private key from a private key and a hash.
func NextKey(key Key, hash Hash32) (Key, error) {
	// Add hash to key value
	b := addPrivateKeys(key.value.Bytes(), hash.Bytes())

	return KeyFromNumber(b, key.Network())
}

// NextPublicKey implements the WP42 method of deriving a public key from a public key and a hash.
func NextPublicKey(key PublicKey, hash Hash32) (PublicKey, error) {
	var result PublicKey

	// Multiply hash by G
	x, y := curveS256.ScalarBaseMult(hash.Bytes())

	// Add to public key
	x, y = curveS256.Add(&key.X, &key.Y, x, y)

	// Check validity
	if x.Sign() == 0 || y.Sign() == 0 {
		return result, ErrOutOfRangeKey
	}

	result.X.Set(x)
	result.Y.Set(y)

	return result, nil
}
