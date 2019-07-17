package wallet

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/db"
)

const (
	walletKey = "wallet"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
)

type KeyStore struct {
	Keys      map[string]*Key
	KeysByPKH map[[20]byte]*Key
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		Keys:      make(map[string]*Key),
		KeysByPKH: make(map[[20]byte]*Key),
	}
}

func (k KeyStore) Add(key *Key) error {
	k.Keys[key.Address.EncodeAddress()] = key
	var pkh [20]byte
	copy(pkh[:], key.Address.ScriptAddress())
	k.KeysByPKH[pkh] = key
	return nil
}

func (k KeyStore) Remove(key *Key) error {
	k.Keys[key.Address.EncodeAddress()] = key
	var pkh [20]byte
	copy(pkh[:], key.Address.ScriptAddress())
	k.KeysByPKH[pkh] = key
	return nil
}

func (k KeyStore) Put(pkh string, privKey *btcec.PrivateKey, pubKey *btcec.PublicKey, chainParams *chaincfg.Params) error {
	addr, _ := btcutil.DecodeAddress(pkh, chainParams)

	k.Keys[pkh] = &Key{
		Address:    addr,
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}

	return nil
}

func (k KeyStore) Get(address string) (*Key, error) {
	key, ok := k.Keys[address]

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

func (k KeyStore) GetPKH(pkh []byte) (*Key, error) {
	var spkh [20]byte
	copy(spkh[:], pkh)
	key, ok := k.KeysByPKH[spkh]

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// Returns pub key hashes in raw byte format
func (k KeyStore) GetRawPubKeyHashes() ([][]byte, error) {
	result := make([][]byte, 0, len(k.Keys))
	for _, walletKey := range k.Keys {
		result = append(result, walletKey.Address.ScriptAddress())
	}
	return result, nil
}

func (k KeyStore) GetAddresses() []btcutil.Address {
	result := make([]btcutil.Address, 0, len(k.Keys))
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

func (k *KeyStore) Load(ctx context.Context, masterDB *db.DB, params *chaincfg.Params) error {
	k.Keys = make(map[string]*Key)
	k.KeysByPKH = make(map[[20]byte]*Key)

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
		if err := newKey.Read(buf, params); err != nil {
			return err
		}

		copy(spkh[:], newKey.Address.ScriptAddress())
		k.KeysByPKH[spkh] = &newKey
		k.Keys[newKey.Address.EncodeAddress()] = &newKey
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
