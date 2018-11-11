package txbuilder

import (
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/btcec"
)

func GetPublicKey(pkBytes []byte) PublicKey {
	pubKey, err := btcec.ParsePubKey(pkBytes, btcec.S256())
	if err != nil {
		//jerr.Get("error parsing pub key", err).Print()
	}
	return PublicKey{
		publicKey: pubKey,
	}
}

type PublicKey struct {
	publicKey *btcec.PublicKey
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
