package bitcoin

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	Hardened          = uint32(0x80000000) // Hardened child index offset
	ExtendedKeyHeader = 0x40
)

type ExtendedKey struct {
	Depth       byte
	FingerPrint [4]byte
	Index       uint32
	ChainCode   [32]byte
	KeyValue    [33]byte
}

// GenerateExtendedKey creates a key from random data.
func GenerateMasterExtendedKey() (ExtendedKey, error) {
	var result ExtendedKey

	seed := make([]byte, 64)
	rand.Read(seed)

	hmac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	_, err := hmac.Write(seed)
	if err != nil {
		return result, err
	}
	sum := hmac.Sum(nil)

	if err := privateKeyIsValid(sum[:32]); err != nil {
		return result, err
	}

	copy(result.KeyValue[:], sum[:32])
	copy(result.ChainCode[1:], sum[32:])

	return result, nil
}

// ExtendedKeyFromBytes creates a key from bytes.
func ExtendedKeyFromBytes(b []byte) (ExtendedKey, error) {
	buf := bytes.NewReader(b)

	header, err := buf.ReadByte()
	if err != nil {
		return ExtendedKey{}, errors.Wrap(err, "read header")
	}
	if header != ExtendedKeyHeader {
		return ExtendedKey{}, errors.New("Not an xkey")
	}

	return readExtendedKey(buf)
}

// ExtendedKeyFromStr creates a key from a string.
func ExtendedKeyFromStr(s string) (ExtendedKey, error) {
	if len(s) < 4 {
		return ExtendedKey{}, errors.New("Too Short")
	}
	hash := DoubleSha256([]byte(s[:len(s)-4]))
	check := hex.EncodeToString(hash[:4])
	if check != s[len(s)-4:] {
		return ExtendedKey{}, errors.New("Invalid check hash")
	}

	parts := strings.Split(s, ":")

	if len(parts) > 2 {
		return ExtendedKey{}, errors.New("To many colons in xkey")
	}

	if len(parts) == 2 {
		if parts[0] != "bitcoin-xkey" {
			return ExtendedKey{}, fmt.Errorf("Invalid xkey prefix : %s", parts[0])
		}
		s = parts[1]
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return ExtendedKey{}, errors.Wrap(err, "decode xkey hex")
	}

	return ExtendedKeyFromBytes(b)
}

// SetBytes decodes the key from bytes.
func (k *ExtendedKey) SetBytes(b []byte) error {
	nk, err := ExtendedKeyFromBytes(b)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// Bytes returns the key data.
func (k ExtendedKey) Bytes() []byte {
	var buf bytes.Buffer

	if err := buf.WriteByte(ExtendedKeyHeader); err != nil {
		return nil
	}

	if err := writeExtendedKey(k, &buf); err != nil {
		return nil
	}

	return buf.Bytes()
}

// String returns the key formatted as text.
func (k ExtendedKey) String() string {
	result := "bitcoin-xkey:0100" + hex.EncodeToString(k.Bytes())

	// Append first 4 bytes of double SHA256 of hash of preceding text
	check := DoubleSha256([]byte(result))
	return result + hex.EncodeToString(check[:4])
}

// SetString decodes a key from text.
func (k *ExtendedKey) SetString(s string) error {
	nk, err := ExtendedKeyFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// Equal returns true if the other key has the same value
func (k ExtendedKey) Equal(other ExtendedKey) bool {
	return bytes.Equal(k.ChainCode[:], other.ChainCode[:]) && bytes.Equal(k.KeyValue[:], other.KeyValue[:])
}

// IsPrivate returns true if the key is a private key.
func (k ExtendedKey) IsPrivate() bool {
	return k.KeyValue[0] == 0
}

// Key returns the (private) key associated with this key.
func (k ExtendedKey) Key(net Network) Key {
	if !k.IsPrivate() {
		return nil
	}
	result, _ := KeyS256FromBytes(k.KeyValue[1:], net) // Skip first zero byte. We just want the 32 byte key value.
	return result
}

// PublicKey returns the public version of this key (xpub).
func (k ExtendedKey) PublicKey() PublicKey {
	if k.IsPrivate() {
		return k.Key(MainNet).PublicKey()
	}
	pub, _ := DecodePublicKeyBytes(k.KeyValue[:])
	return pub
}

// ExtendedPublicKey returns the public version of this key.
func (k ExtendedKey) ExtendedPublicKey() ExtendedKey {
	if !k.IsPrivate() {
		return k
	}

	result := k
	copy(k.KeyValue[:], k.Key(MainNet).PublicKey().Bytes())

	return result
}

// ChildKey returns the child key at the specified index.
func (k ExtendedKey) ChildKey(index uint32) (ExtendedKey, error) {
	if index >= Hardened && !k.IsPrivate() {
		return ExtendedKey{}, errors.New("Can't derive hardened child from xpub")
	}

	result := ExtendedKey{
		Depth: k.Depth + 1,
		Index: index,
	}

	// Calculate fingerprint
	var hash []byte
	if k.IsPrivate() {
		hash = Hash160(k.PublicKey().Bytes())
	} else {
		hash = Hash160(k.KeyValue[:])
	}
	copy(result.FingerPrint[:], hash[:4])

	// Calculate child
	hmac := hmac.New(sha512.New, k.ChainCode[:])
	if index >= Hardened {
		hmac.Write(k.KeyValue[:])
	} else {
		if k.IsPrivate() {
			hmac.Write(k.PublicKey().Bytes())
		} else {
			hmac.Write(k.KeyValue[:])
		}
	}

	err := binary.Write(hmac, binary.LittleEndian, index)
	if err != nil {
		return result, errors.Wrap(err, "write index to hmac")
	}

	sum := hmac.Sum(nil)

	// Set chain code
	copy(result.ChainCode[:], sum[32:])

	// Calculate child
	if k.IsPrivate() {
		copy(result.KeyValue[1:], addPrivateKeys(sum[:32], k.KeyValue[1:]))

		if err := privateKeyIsValid(result.KeyValue[1:]); err != nil {
			return result, errors.Wrap(err, "child add private")
		}
	} else {
		privateKey, err := KeyS256FromBytes(sum[:32], MainNet)
		if err != nil {
			return result, errors.Wrap(err, "parse child private key")
		}

		copy(result.KeyValue[:], addPublicKeys(privateKey.Number(), k.KeyValue[:]))

		if err := publicKeyIsValid(result.KeyValue[:]); err != nil {
			return result, errors.Wrap(err, "child add public")
		}
	}

	return result, nil
}

// MarshalJSON converts to json.
func (k *ExtendedKey) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *ExtendedKey) UnmarshalJSON(data []byte) error {
	return k.SetString(string(data[1 : len(data)-1]))
}

// Scan converts from a database column.
func (k *ExtendedKey) Scan(data interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return errors.New("ExtendedKey db column not bytes")
	}

	c := make([]byte, len(b))
	copy(c, b)
	return k.SetBytes(c)
}

// readExtendedKey reads just the basic data of the extended key.
func readExtendedKey(buf *bytes.Reader) (ExtendedKey, error) {
	var result ExtendedKey
	var err error

	result.Depth, err = buf.ReadByte()
	if err != nil {
		return result, errors.Wrap(err, "reading xkey depth")
	}

	_, err = buf.Read(result.FingerPrint[:])
	if err != nil {
		return result, errors.Wrap(err, "reading xkey fingerprint")
	}

	err = binary.Read(buf, binary.LittleEndian, &result.Index)
	if err != nil {
		return result, errors.Wrap(err, "reading xkey index")
	}

	_, err = buf.Read(result.ChainCode[:])
	if err != nil {
		return result, errors.Wrap(err, "reading xkey chaincode")
	}

	_, err = buf.Read(result.KeyValue[:])
	if err != nil {
		return result, errors.Wrap(err, "reading xkey key")
	}

	return result, nil
}

// writeExtendedKey writes just the basic data of the extended key.
func writeExtendedKey(ek ExtendedKey, buf *bytes.Buffer) error {
	err := buf.WriteByte(ek.Depth)
	if err != nil {
		return errors.Wrap(err, "writing xkey depth")
	}

	_, err = buf.Write(ek.FingerPrint[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey fingerprint")
	}

	err = binary.Write(buf, binary.LittleEndian, ek.Index)
	if err != nil {
		return errors.Wrap(err, "writing xkey index")
	}

	_, err = buf.Write(ek.ChainCode[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey chaincode")
	}

	_, err = buf.Write(ek.KeyValue[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey key")
	}

	return nil
}
