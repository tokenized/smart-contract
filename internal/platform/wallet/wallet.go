package wallet

/**
 * Wallet Service
 *
 * What is my purpose?
 * - You store keys
 */

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/db"
)

type WalletInterface interface {
	Get(string) (*Key, error)
	GetPKH([]byte) (*Key, error)
	List([]string) ([]*Key, error)
	ListPKH([][]byte) ([]*Key, error)
	ListAll() []*Key
	Add(*Key) error
	Load(context.Context, *db.DB, *chaincfg.Params) error
	Save(context.Context, *db.DB) error
}

type Wallet struct {
	lock     sync.RWMutex
	KeyStore *KeyStore
}

func New() *Wallet {
	return &Wallet{
		KeyStore: NewKeyStore(),
	}
}

func (w Wallet) Add(key *Key) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.KeyStore.Add(key)
}

// Register a private key with the wallet
func (w Wallet) Register(secret string, chainParams *chaincfg.Params) error {
	w.lock.Lock()
	defer w.lock.Unlock()

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

func (w Wallet) List(addrs []string) ([]*Key, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	var rks []*Key

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

func (w Wallet) ListPKH(pkhs [][]byte) ([]*Key, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	var rks []*Key

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

func (w Wallet) ListAll() []*Key {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.KeyStore.GetAll()
}

func (w Wallet) Get(addr string) (*Key, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.KeyStore.Get(addr)
}

func (w Wallet) GetPKH(pkh []byte) (*Key, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.KeyStore.GetPKH(pkh)
}

func (w Wallet) Load(ctx context.Context, masterDB *db.DB, params *chaincfg.Params) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.KeyStore.Load(ctx, masterDB, params)
}

func (w Wallet) Save(ctx context.Context, masterDB *db.DB) error {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.KeyStore.Save(ctx, masterDB)
}
