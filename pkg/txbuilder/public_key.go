package txbuilder

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
)

type PublicKey struct {
	publicKey *btcec.PublicKey
}

func NewPublicKey(pkBytes []byte) (*PublicKey, error) {
	pubKey, err := btcec.ParsePubKey(pkBytes, btcec.S256())
	if err != nil {
		return nil, errors.Wrap(err, "parsing pub key")
	}

	newPk := &PublicKey{
		publicKey: pubKey,
	}

	return newPk, nil
}

func (k PublicKey) GetSerialized() []byte {
	if k.publicKey == nil {
		return []byte{}
	}
	return k.publicKey.SerializeCompressed()
}

func (k PublicKey) GetSerializedString() string {
	return fmt.Sprintf("%x", k.GetSerialized())
}

func (k PublicKey) GetAddress() (btcutil.Address, error) {
	return GetAddress(k.GetSerialized())
}
