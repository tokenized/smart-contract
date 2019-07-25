package wallet

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	walletKey = "wallet"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
)

type KeyStore struct {
	Keys map[[20]byte]*Key
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		Keys: make(map[[20]byte]*Key),
	}
}

func (k KeyStore) Add(key *Key) error {
	var pkh [20]byte
	addressPKH, ok := key.Address.(*bitcoin.AddressPKH)
	if !ok {
		return errors.New("Key address not PKH")
	}
	copy(pkh[:], addressPKH.PKH())
	k.Keys[pkh] = key
	return nil
}

func (k KeyStore) Remove(key *Key) error {
	addressPKH, ok := key.Address.(*bitcoin.AddressPKH)
	if !ok {
		return errors.New("Key address not PKH")
	}

	var pkh [20]byte
	copy(pkh[:], addressPKH.PKH())
	delete(k.Keys, pkh)
	return nil
}

// Get returns the key corresponding to the specified address.
func (k KeyStore) Get(address bitcoin.Address) (*Key, error) {
	addressPKH, ok := address.(*bitcoin.AddressPKH)
	if !ok {
		return nil, errors.New("Wrong address type")
	}
	var spkh [20]byte
	copy(spkh[:], addressPKH.PKH())
	key, ok := k.Keys[spkh]

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

func (k KeyStore) GetAddresses() []bitcoin.Address {
	result := make([]bitcoin.Address, 0, len(k.Keys))
	for _, key := range k.Keys {
		result = append(result, key.Address)
	}
	return result
}

func (k KeyStore) GetAll() []*Key {
	result := make([]*Key, 0, len(k.Keys))
	for _, key := range k.Keys {
		result = append(result, key)
	}
	return result
}

func (k *KeyStore) Load(ctx context.Context, masterDB *db.DB, net wire.BitcoinNet) error {
	k.Keys = make(map[[20]byte]*Key)

	data, err := masterDB.Fetch(ctx, walletKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil // No keys yet
		}
		return err
	}

	buf := bytes.NewBuffer(data)

	var count uint32
	if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	var spkh [20]byte
	for i := uint32(0); i < count; i++ {
		var newKey Key
		if err := newKey.Read(buf, net); err != nil {
			return err
		}

		copy(spkh[:], bitcoin.Hash160(newKey.Key.PublicKey().Bytes()))
		k.Keys[spkh] = &newKey
	}

	return nil
}

func (k *KeyStore) Save(ctx context.Context, masterDB *db.DB) error {
	var buf bytes.Buffer

	count := uint32(len(k.Keys))
	if err := binary.Write(&buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	for _, key := range k.Keys {
		if err := key.Write(&buf); err != nil {
			return err
		}
	}

	return masterDB.Put(ctx, walletKey, buf.Bytes())
}
