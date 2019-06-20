package wallet

import (
	"bytes"
	"crypto/sha256"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"golang.org/x/crypto/ripemd160"
)

type Key struct {
	Address    btcutil.Address
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
}

func NewKey(key *btcec.PrivateKey, params *chaincfg.Params) *Key {
	result := Key{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}

	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(result.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	result.Address, _ = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), params)
	return &result
}

func (rk *Key) Read(buf *bytes.Buffer, params *chaincfg.Params) error {
	data := make([]byte, btcec.PrivKeyBytesLen)
	if _, err := buf.Read(data); err != nil {
		return err
	}
	rk.PrivateKey, rk.PublicKey = btcec.PrivKeyFromBytes(btcec.S256(), data)

	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(rk.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	var err error
	rk.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), params)
	return err
}

func (rk *Key) Write(buf *bytes.Buffer) error {
	_, err := buf.Write(rk.PrivateKey.Serialize())
	return err
}
