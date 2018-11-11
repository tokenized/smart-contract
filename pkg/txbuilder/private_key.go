package txbuilder

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
)

func GetPrivateKey(secretHex string) PrivateKey {
	secret, _ := hex.DecodeString(secretHex)
	return PrivateKey{
		Secret: secret,
	}
}

func GeneratePrivateKey() PrivateKey {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 32)
	rand.Read(b)
	return PrivateKey{
		Secret: b,
	}
}

func ImportPrivateKey(wifString string) (PrivateKey, error) {
	wif, err := btcutil.DecodeWIF(wifString)
	if err != nil {
		return PrivateKey{}, err
	}
	return PrivateKey{
		Secret: wif.PrivKey.Serialize(),
	}, nil
}

type PrivateKey struct {
	Secret []byte
}

func (k PrivateKey) GetBinaryString() string {
	var binaryKey string
	for _, n := range k.Secret {
		binaryKey += fmt.Sprintf("%b", n)
	}
	return binaryKey
}

func (k PrivateKey) GetBase58() string {
	return base58.CheckEncode(k.Secret, 128)
}

func (k PrivateKey) GetBase58Compressed() string {
	return base58.CheckEncode(append(k.Secret, 0x01), 128)
}

func (k PrivateKey) GetHex() string {
	return fmt.Sprintf("%x", k.Secret)
}

func (k PrivateKey) GetHexCompressed() string {
	return fmt.Sprintf("%x", append(k.Secret, 0x01))
}

func (k PrivateKey) GetPublicKey() PublicKey {
	_, pub := btcec.PrivKeyFromBytes(btcec.S256(), k.Secret)
	return PublicKey{
		publicKey: pub,
	}
}

func (k PrivateKey) GetBtcEcPrivateKey() *btcec.PrivateKey {
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), k.Secret)
	return priv
}
