package txbuilder

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func GetAddress(pubKey []byte) (btcutil.Address, error) {
	if len(pubKey) == 0 {
		return nil, errors.New("Empty pubkey")
	}

	addr, err := btcutil.NewAddressPubKey(pubKey, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	address, err := btcutil.DecodeAddress(addr.EncodeAddress(), &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	return address, nil
}

func GetAddressFromString(addressString string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(addressString, &chaincfg.MainNetParams)
}
