package wallet

import (
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
)

type RootKey struct {
	Address    btcutil.Address
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
}

type KeyStore struct {
	Keys map[string]*RootKey
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		Keys: make(map[string]*RootKey),
	}
}

func (k KeyStore) Put(pkh string, privKey *btcec.PrivateKey, pubKey *btcec.PublicKey) error {
	addr, _ := btcutil.DecodeAddress(pkh, &chaincfg.MainNetParams)

	k.Keys[pkh] = &RootKey{
		Address:    addr,
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}

	return nil
}

func (k KeyStore) Get(address string) (*RootKey, error) {
	key, ok := k.Keys[address]

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// Returns pub key hashes in raw byte format
func (k KeyStore) GetRawPubKeyHashes() ([][]byte, error) {
	result := make([][]byte, 0, len(k.Keys))
	for _, rootKey := range k.Keys {
		result = append(result, rootKey.Address.ScriptAddress())
	}
	return result, nil
}
