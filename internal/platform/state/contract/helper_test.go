package contract

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func decodeAddress(address string) btcutil.Address {
	a, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}

	return a
}
