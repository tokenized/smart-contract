package txbuilder

import (
	"github.com/btcsuite/btcutil"
)

type PayAddress struct {
	Address btcutil.Address
	Value   uint64
}

func NewPayAddress(address btcutil.Address, value uint64) PayAddress {
	return PayAddress{
		Address: address,
		Value:   value,
	}
}
