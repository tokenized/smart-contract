package wallet

/**
 * Wallet Service
 *
 * What is my purpose?
 * - You sign messages
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
	List([]string) ([]*RootKey, error)
}

type Wallet struct {
	KeyStore *KeyStore
}

func New() *Wallet {
	return &Wallet{
		KeyStore: NewKeyStore(),
	}
}

// Register a private key with the wallet
func (w Wallet) Register(secret string) error {
	if len(secret) == 0 {
		return errors.New("Create wallet failed: missing secret")
	}

	// load the WIF if we have one
	wif, err := btcutil.DecodeWIF(secret)
	if err != nil {
		return err
	}

	// Private / Public Keys
	priv := wif.PrivKey
	pub := priv.PubKey()

	// Public Address (PKH)
	h := hex.EncodeToString(pub.SerializeCompressed())
	pubhash, err := btcutil.DecodeAddress(h, &chaincfg.MainNetParams)
	if err != nil {
		return err
	}
	pubaddr := pubhash.EncodeAddress()

	// Put in key store
	w.KeyStore.Put(pubaddr, priv, pub)
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

func (w Wallet) Get(addr string) (*RootKey, error) {
	return w.KeyStore.Get(addr)
}

// func BuildTX(key *RootKey, iutxos inspector.UTXOs, outs []txbuilder.TxOutput, changeAddress btcutil.Address, m protocol.OpReturnMessage) (*wire.MsgTx, error) {

// // Convert inspector to txbuilder
// utxos := txbuilder.UTXOs{}
// for _, iutxo := range iutxos {
// utxo := txbuilder.UTXO{
// Hash:     iutxo.Hash,
// PkScript: iutxo.PkScript,
// Index:    iutxo.Index,
// Value:    uint64(iutxo.Value),
// }
// utxos = append(utxos, utxo)
// }

// outputs := buildOutputs(outs)

// payload, err := protocol.Serialize(m)
// if err != nil {
// return nil, err
// }

// builder := txbuilder.NewTxBuilder(key.PrivateKey)

// return builder.Build(utxos, outputs, changeAddress, payload)
// }

// func buildOutputs(outs []txbuilder.TxOutput) []txbuilder.PayAddress {

// // TODO is there a better place to do this? Do I need to do this?

// // The outputs do not need to include the change for the contract
// // address. Change will be calculated by the bch package.
// outputs := []txbuilder.PayAddress{}

// // create any other payments required. There may be >= 0 payments here.
// for _, out := range outs {
// outAddress, err := txbuilder.GetAddressFromString(out.Address.EncodeAddress())
// if err != nil {
// continue
// }

// payment := txbuilder.PayAddress{
// Address: outAddress,
// Value:   out.Value,
// }

// outputs = append(outputs, payment)
// }

// return outputs
// }
