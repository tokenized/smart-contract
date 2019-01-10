package txbuilder

import (
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// UTXOs is a wrapper for a []UTXO.
type UTXOs []UTXO

// NewUTXOs returns a new UTXOs.
func NewUTXOs(tx *wire.MsgTx) UTXOs {
	utxos := UTXOs{}

	for _, txIn := range tx.TxIn {
		pop := txIn.PreviousOutPoint

		u := NewUTXOFromTX(*tx, pop.Index)
		utxos = append(utxos, u)
	}

	return utxos
}

// Value returns the total value of the set of UTXO's.
func (u UTXOs) Value() uint64 {
	v := uint64(0)

	for _, utxo := range u {
		v += utxo.Value
	}

	return v
}

// Addresses returns all addresses from the UTXO's, in order.
func (u UTXOs) Addresses() ([]btcutil.Address, error) {
	addresses := []btcutil.Address{}

	for _, utxo := range u {
		a, err := utxo.PublicAddress(&chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, a)
	}

	return addresses, nil
}

// UniqueAddresses returns the unique addresses from the UTXO's, in order.
func (u UTXOs) UniqueAddresses() ([]btcutil.Address, error) {
	addresses, err := u.Addresses()
	if err != nil {
		return nil, err
	}

	set := AddressSet{
		Addresses: addresses,
	}
	return set.Set(), nil
}

// ForAddress returns UTXOs that match the given Address.
func (u UTXOs) ForAddress(address btcutil.Address) (UTXOs, error) {
	filtered := UTXOs{}

	s := address.String()

	for _, utxo := range u {
		publicAddress, err := utxo.PublicAddress(&chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}

		if publicAddress.String() != s {
			continue
		}

		filtered = append(filtered, utxo)
	}

	return filtered, nil
}
