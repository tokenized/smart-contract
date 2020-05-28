package wallet

/**
 * Wallet Service
 *
 * What is my purpose?
 * - You store keys
 */

import (
	"bytes"
	"errors"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
)

type WalletInterface interface {
	Get(bitcoin.RawAddress) (*Key, error)
	List([]bitcoin.RawAddress) ([]*Key, error)
	ListAll() []*Key
	Remove(*Key) error
	Serialize(*bytes.Buffer) error
	Deserialize(*bytes.Reader) error
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

func (w Wallet) Remove(key *Key) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.KeyStore.Remove(key)
}

// Register a private key with the wallet
func (w Wallet) Register(wif string, net bitcoin.Network) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(wif) == 0 {
		return errors.New("Create wallet failed: missing secret")
	}

	// load the WIF if we have one
	key, err := bitcoin.KeyFromStr(wif)
	if err != nil {
		return err
	}

	// Put in key store
	newKey := NewKey(key)
	w.KeyStore.Add(newKey)
	return nil
}

func (w Wallet) List(addrs []bitcoin.RawAddress) ([]*Key, error) {
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

func (w Wallet) ListAll() []*Key {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.KeyStore.GetAll()
}

func (w Wallet) Get(address bitcoin.RawAddress) (*Key, error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.KeyStore.Get(address)
}

func (w Wallet) Serialize(buf *bytes.Buffer) error {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.KeyStore.Serialize(buf)
}

func (w Wallet) Deserialize(buf *bytes.Reader) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.KeyStore.Deserialize(buf)
}
