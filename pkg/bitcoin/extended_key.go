package bitcoin

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	bip32 "github.com/tyler-smith/go-bip32"
)

const (
	Hardened             = uint32(0x80000000) // Hardened child index offset
	ExtendedKeyHeader    = 0x40
	ExtendedKeyURLPrefix = "bitcoin-xkey"
)

var (
	ErrNotExtendedKey = errors.New("Data not an xkey")
)

type ExtendedKey struct {
	Network     Network
	Depth       byte
	FingerPrint [4]byte
	Index       uint32
	ChainCode   [32]byte
	KeyValue    [33]byte
}

// LoadMasterExtendedKey creates a key from a seed.
func LoadMasterExtendedKey(seed []byte) (ExtendedKey, error) {
	var result ExtendedKey

	hmac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	_, err := hmac.Write(seed)
	if err != nil {
		return result, err
	}
	sum := hmac.Sum(nil)

	if err := privateKeyIsValid(sum[:32]); err != nil {
		return result, err
	}

	copy(result.KeyValue[1:], sum[:32])
	copy(result.ChainCode[:], sum[32:])

	return result, nil
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

	copy(result.KeyValue[1:], sum[:32])
	copy(result.ChainCode[:], sum[32:])

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
		// Fall back to BIP-0032 format
		bip32Key, err := bip32.Deserialize(b)
		if err != nil {
			return ExtendedKey{}, ErrNotExtendedKey
		}

		return fromBIP32(bip32Key)
	}

	var result ExtendedKey
	err = result.read(buf)
	return result, err
}

// ExtendedKeyFromBytes creates a key from bytes.
func (k *ExtendedKey) Deserialize(buf *bytes.Reader) error {
	header, err := buf.ReadByte()
	if err != nil {
		return errors.Wrap(err, "read header")
	}
	if header != ExtendedKeyHeader {
		// Fall back to BIP-0032 format
		b := make([]byte, 82)
		if _, err := buf.Read(b); err != nil {
			return err
		}
		bip32Key, err := bip32.Deserialize(b)
		if err != nil {
			return ErrNotExtendedKey
		}

		return k.setFromBIP32(bip32Key)
	}

	return k.read(buf)
}

// ExtendedKeyFromStr creates a key from a hex string.
func ExtendedKeyFromStr(s string) (ExtendedKey, error) {
	net, prefix, data, err := BIP0276Decode(s)
	if err != nil {
		// Fall back to BIP-0032 format
		bip32Key, b32err := bip32.B58Deserialize(s)
		if b32err != nil {
			return ExtendedKey{}, errors.Wrap(err, "decode xkey hex string")
		}

		return fromBIP32(bip32Key)
	}

	if prefix != ExtendedKeyURLPrefix {
		return ExtendedKey{}, fmt.Errorf("Wrong prefix : %s", prefix)
	}

	result, err := ExtendedKeyFromBytes(data)
	if err != nil {
		return ExtendedKey{}, err
	}

	result.Network = net
	return result, nil
}

// ExtendedKeyFromStr58 creates a key from a base 58 string.
func ExtendedKeyFromStr58(s string) (ExtendedKey, error) {
	net, prefix, data, err := BIP0276Decode58(s)
	if err != nil {
		// Fall back to BIP-0032 format
		bip32Key, b32err := bip32.B58Deserialize(s)
		if b32err != nil {
			return ExtendedKey{}, errors.Wrap(err, "decode xkey base58 string")
		}

		return fromBIP32(bip32Key)
	}

	if prefix != ExtendedKeyURLPrefix {
		return ExtendedKey{}, fmt.Errorf("Wrong prefix : %s", prefix)
	}

	result, err := ExtendedKeyFromBytes(data)
	if err != nil {
		return ExtendedKey{}, err
	}

	result.Network = net
	return result, nil
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
	k.Serialize(&buf)
	return buf.Bytes()
}

// Serialize writes the key data.
func (k ExtendedKey) Serialize(buf *bytes.Buffer) error {
	if err := buf.WriteByte(ExtendedKeyHeader); err != nil {
		return err
	}

	if err := k.write(buf); err != nil {
		return err
	}

	return nil
}

// String returns the key formatted as hex text.
func (k ExtendedKey) String() string {
	return BIP0276Encode(k.Network, ExtendedKeyURLPrefix, k.Bytes())
}

// String58 returns the key formatted as base 58 text.
func (k ExtendedKey) String58() string {
	return BIP0276Encode58(k.Network, ExtendedKeyURLPrefix, k.Bytes())
}

// SetString decodes a key from hex text.
func (k *ExtendedKey) SetString(s string) error {
	nk, err := ExtendedKeyFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

// SetString58 decodes a key from base 58 text.
func (k *ExtendedKey) SetString58(s string) error {
	nk, err := ExtendedKeyFromStr(s)
	if err != nil {
		return err
	}

	*k = nk
	return nil
}

func (k *ExtendedKey) SetNetwork(net Network) {
	k.Network = net
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
		return Key{}
	}
	result, _ := KeyFromNumber(k.KeyValue[1:], net) // Skip first zero byte. We just want the 32 byte key value.
	return result
}

// PublicKey returns the public version of this key (xpub).
func (k ExtendedKey) PublicKey() PublicKey {
	if k.IsPrivate() {
		return k.Key(MainNet).PublicKey()
	}
	pub, _ := PublicKeyFromBytes(k.KeyValue[:])
	return pub
}

// RawAddress returns a raw address for this key.
func (k ExtendedKey) RawAddress() (RawAddress, error) {
	return k.PublicKey().RawAddress()
}

// ExtendedPublicKey returns the public version of this key.
func (k ExtendedKey) ExtendedPublicKey() ExtendedKey {
	if !k.IsPrivate() {
		return k
	}

	result := k
	copy(result.KeyValue[:], k.Key(InvalidNet).PublicKey().Bytes())
	return result
}

// ChildKey returns the child key at the specified index.
func (k ExtendedKey) ChildKey(index uint32) (ExtendedKey, error) {
	if index >= Hardened && !k.IsPrivate() {
		return ExtendedKey{}, errors.New("Can't derive hardened child from xpub")
	}

	result := ExtendedKey{
		Network: k.Network,
		Depth:   k.Depth + 1,
		Index:   index,
	}

	// Calculate fingerprint
	var fingerPrint []byte
	if k.IsPrivate() {
		fingerPrint = Hash160(k.PublicKey().Bytes())
	} else {
		fingerPrint = Hash160(k.KeyValue[:])
	}
	copy(result.FingerPrint[:], fingerPrint[:4])

	// Calculate child
	hmac := hmac.New(sha512.New, k.ChainCode[:])
	if index >= Hardened { // Hardened child
		// Write private key with leading zero
		hmac.Write(k.KeyValue[:])
	} else {
		// Write compressed public key
		if k.IsPrivate() {
			hmac.Write(k.PublicKey().Bytes())
		} else {
			hmac.Write(k.KeyValue[:])
		}
	}

	err := binary.Write(hmac, binary.BigEndian, index)
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
		privateKey, err := KeyFromNumber(sum[:32], MainNet)
		if err != nil {
			return result, errors.Wrap(err, "parse child private key")
		}
		publicKey := privateKey.PublicKey().Bytes()

		copy(result.KeyValue[:], addCompressedPublicKeys(publicKey, k.KeyValue[:]))

		if err := compressedPublicKeyIsValid(result.KeyValue[:]); err != nil {
			return result, errors.Wrap(err, "child add public")
		}
	}

	return result, nil
}

func PathIndexToString(index uint32) string {
	if index >= Hardened {
		return fmt.Sprintf("%d'", index-Hardened)
	}
	return fmt.Sprintf("%d", index)
}

func PathToString(values []uint32) string {
	var parts = make([]string, 0, len(values)+1)
	parts = append(parts, "m")

	for _, v := range values {
		parts = append(parts, PathIndexToString(v))
	}

	return strings.Join(parts, "/")
}

func PathIndexFromString(index string) (uint32, error) {
	if len(index) == 0 {
		return 0, errors.New("Empty index value")
	}
	hard := false
	if index[len(index)-1] == '\'' {
		hard = true
		index = index[:len(index)-1]
	}
	if len(index) == 0 {
		return 0, errors.New("Empty index value")
	}
	value, err := strconv.Atoi(index)
	if err != nil {
		return 0, errors.Wrap(err, "path index not integer")
	}
	if hard {
		return uint32(value) + Hardened, nil
	}
	return uint32(value), nil
}

func PathFromString(s string) ([]uint32, error) {
	parts := strings.Split(s, "/")

	if len(parts) == 0 {
		return nil, errors.New("Path empty")
	}

	if parts[0] == "m" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return nil, errors.New("Path empty")
	}

	result := make([]uint32, 0, len(parts))

	for _, n := range parts {
		if len(n) == 0 {
			return nil, fmt.Errorf("Empty path index : %s", s)
		}

		index, err := PathIndexFromString(n)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Invalid path index : %s", n))
		}

		result = append(result, index)
	}

	return result, nil
}

// ChildKeyForPath returns the child key at the specified index path.
func (k ExtendedKey) ChildKeyForPath(path []uint32) (ExtendedKey, error) {
	var err error
	result := k
	for _, index := range path {
		result, err = result.ChildKey(index)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// MarshalJSON converts to json.
func (k ExtendedKey) MarshalJSON() ([]byte, error) {
	return []byte("\"" + k.String58() + "\""), nil
}

// UnmarshalJSON converts from json.
func (k *ExtendedKey) UnmarshalJSON(data []byte) error {
	return k.SetString58(string(data[1 : len(data)-1]))
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

// fromBIP32 creates an extended key from a bip32 key.
func fromBIP32(old *bip32.Key) (ExtendedKey, error) {
	result := ExtendedKey{}
	err := result.setFromBIP32(old)
	return result, err
}

// setFromBIP32 assigns the extended key to the same value as the bip32 key.
func (k *ExtendedKey) setFromBIP32(old *bip32.Key) error {
	k.Network = InvalidNet
	k.Depth = old.Depth
	copy(k.FingerPrint[:], old.FingerPrint)
	k.Index = binary.BigEndian.Uint32(old.ChildNumber)
	copy(k.ChainCode[:], old.ChainCode)
	if old.IsPrivate {
		k.KeyValue[0] = 0
		copy(k.KeyValue[1:], old.Key)
	} else {
		copy(k.KeyValue[:], old.Key)
	}

	return nil
}

// read reads just the basic data of the extended key.
func (k *ExtendedKey) read(buf *bytes.Reader) error {
	var err error

	k.Depth, err = buf.ReadByte()
	if err != nil {
		return errors.Wrap(err, "reading xkey depth")
	}

	_, err = buf.Read(k.FingerPrint[:])
	if err != nil {
		return errors.Wrap(err, "reading xkey fingerprint")
	}

	err = binary.Read(buf, binary.BigEndian, &k.Index)
	if err != nil {
		return errors.Wrap(err, "reading xkey index")
	}

	_, err = buf.Read(k.ChainCode[:])
	if err != nil {
		return errors.Wrap(err, "reading xkey chaincode")
	}

	_, err = buf.Read(k.KeyValue[:])
	if err != nil {
		return errors.Wrap(err, "reading xkey key")
	}

	return nil
}

// write writes just the basic data of the extended key.
func (k ExtendedKey) write(buf *bytes.Buffer) error {
	err := buf.WriteByte(k.Depth)
	if err != nil {
		return errors.Wrap(err, "writing xkey depth")
	}

	_, err = buf.Write(k.FingerPrint[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey fingerprint")
	}

	err = binary.Write(buf, binary.BigEndian, k.Index)
	if err != nil {
		return errors.Wrap(err, "writing xkey index")
	}

	_, err = buf.Write(k.ChainCode[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey chaincode")
	}

	_, err = buf.Write(k.KeyValue[:])
	if err != nil {
		return errors.Wrap(err, "writing xkey key")
	}

	return nil
}
