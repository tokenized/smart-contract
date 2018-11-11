package wallet

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
)

type KeyStore struct {
	Keys map[string]*btcec.PrivateKey
}

func NewKeyStore(privKey *btcec.PrivateKey) (*KeyStore, error) {
	pub := privKey.PubKey()

	h := hex.EncodeToString(pub.SerializeCompressed())

	pubhash, err := btcutil.DecodeAddress(h, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	address := pubhash.EncodeAddress()

	store := KeyStore{
		Keys: map[string]*btcec.PrivateKey{
			address: privKey,
		},
	}

	return &store, nil
}

func (k KeyStore) Get(address string) (*btcec.PrivateKey, error) {
	key, ok := k.Keys[address]

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}
