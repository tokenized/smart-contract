package wallet

/**
 * Wallet Service
 *
 * What is my purpose?
 * - You store keys
 */

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

type WalletInterface interface {
	Get(string) (*RootKey, error)
	GetPKH([]byte) (*RootKey, error)
	List([]string) ([]*RootKey, error)
	ListPKH([][]byte) ([]*RootKey, error)
	ListAll() []*RootKey
}

type Wallet struct {
	KeyStore *KeyStore
}

func New() *Wallet {
	return &Wallet{
		KeyStore: NewKeyStore(),
	}
}

func (w Wallet) Add(key *RootKey) error {
	return w.KeyStore.Add(key)
}

// Register a private key with the wallet
func (w Wallet) Register(secret string, chainParams *chaincfg.Params) error {
	if len(secret) == 0 {
		return errors.New("Create wallet failed: missing secret")
	}

	// load the WIF if we have one
	wif, err := btcutil.DecodeWIF(secret)
	if err != nil {
		return err
	}
	if !wif.IsForNet(chainParams) {
		return errors.New("WIF for wrong net")
	}

	// Private / Public Keys
	priv := wif.PrivKey
	pub := priv.PubKey()

	// Public Address (PKH)
	h := hex.EncodeToString(pub.SerializeCompressed())
	pubhash, err := btcutil.DecodeAddress(h, chainParams)
	if err != nil {
		return err
	}
	pubaddr := pubhash.EncodeAddress()

	// Put in key store
	w.KeyStore.Put(pubaddr, priv, pub, chainParams)
	return nil
}

func (w Wallet) List(addrs []string) ([]*RootKey, error) {
	var rks []*RootKey

	for _, addr := range addrs {
		rk, err := w.Get(addr)
		if err != nil {
			if err == ErrKeyNotFound {
				continue
			}
			return nil, err
		}

		rks = append(rks, rk)
	}

	return rks, nil
}

func (w Wallet) ListPKH(pkhs [][]byte) ([]*RootKey, error) {
	var rks []*RootKey

	for _, pkh := range pkhs {
		rk, err := w.GetPKH(pkh)
		if err != nil {
			if err == ErrKeyNotFound {
				continue
			}
			return nil, err
		}

		rks = append(rks, rk)
	}

	return rks, nil
}

func (w Wallet) ListAll() []*RootKey {
	return w.KeyStore.GetAll()
}

func (w Wallet) Get(addr string) (*RootKey, error) {
	return w.KeyStore.Get(addr)
}

func (w Wallet) GetPKH(pkh []byte) (*RootKey, error) {
	return w.KeyStore.GetPKH(pkh)
}
