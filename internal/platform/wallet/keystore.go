package wallet

import (
	"errors"

	"github.com/btcsuite/btcd/btcec"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
)

type RootKey struct {
	PublicAddress string
	PrivateKey    *btcec.PrivateKey
	PublicKey     *btcec.PublicKey
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
	k.Keys[pkh] = &RootKey{
		PublicAddress: pkh,
		PrivateKey:    privKey,
		PublicKey:     pubKey,
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
