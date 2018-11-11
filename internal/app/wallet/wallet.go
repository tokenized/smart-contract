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

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

type Wallet struct {
	KeyStore      *KeyStore
	PublicAddress string
	PrivateKey    *btcec.PrivateKey
	PublicKey     *btcec.PublicKey
}

func NewWallet(secret string) (*Wallet, error) {
	if len(secret) == 0 {
		return nil, errors.New("Create wallet failed: missing secret")
	}

	// load the WIF if we have one
	wif, err := btcutil.DecodeWIF(secret)
	if err != nil {
		return nil, err
	}

	// Private / Public Keys
	priv := wif.PrivKey
	pub := priv.PubKey()

	// Public Address (PKH)
	h := hex.EncodeToString(pub.SerializeCompressed())
	pubhash, err := btcutil.DecodeAddress(h, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	pubaddr := pubhash.EncodeAddress()

	// Key Store
	keystore, err := NewKeyStore(priv)
	if err != nil {
		return nil, err
	}

	w := Wallet{
		KeyStore:      keystore,
		PublicAddress: pubaddr,
		PrivateKey:    priv,
		PublicKey:     pub,
	}

	return &w, nil
}

func (w Wallet) Get(address string) (*btcec.PrivateKey, error) {
	return w.KeyStore.Get(address)
}

func (w Wallet) BuildTX(key *btcec.PrivateKey,
	utxos txbuilder.UTXOs,
	outs []txbuilder.TxOutput,
	sender btcutil.Address,
	changeAddress btcutil.Address,
	m protocol.OpReturnMessage) (*wire.MsgTx, error) {

	outputs := w.buildOutputs(outs)

	payload := make([]byte, m.Len(), m.Len())
	if _, err := m.Read(payload); err != nil {
		return nil, err
	}

	builder := txbuilder.NewTxBuilder(key)

	return builder.Build(utxos, outputs, changeAddress, payload)
}

func (w Wallet) buildOutputs(outs []txbuilder.TxOutput) []txbuilder.PayAddress {

	// TODO is there a better place to do this? Do I need to do this?

	// The outputs do not need to include the change for the contract
	// address. Change will be calculated by the bch package.
	outputs := []txbuilder.PayAddress{}

	// create any other payments required. There may be >= 0 payments here.
	for _, out := range outs {
		outAddress, err := txbuilder.GetAddressFromString(out.Address.EncodeAddress())
		if err != nil {
			continue
		}

		payment := txbuilder.NewPayAddress(outAddress, out.Value)
		outputs = append(outputs, payment)
	}

	return outputs
}
