package txbuilder

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// PayAddress is an address with an intended payment
type PayAddress struct {
	Address btcutil.Address
	Value   uint64
}

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

// AddressSet is used to create a unique set of addresses from a slice of
// btcutil.Address, where the order of the values in the original slice will
// be preserved.
type AddressSet struct {
	Addresses []btcutil.Address
}

// Set returns a set of btcutil.Addresses.
//
// Being a set, the returned slice contains unique values. The order of the
// values are also preserved.
func (a AddressSet) Set() []btcutil.Address {
	unique := []btcutil.Address{}

	for _, addr := range a.Addresses {
		if !a.in(unique, addr) {
			unique = append(unique, addr)
		}
	}

	return unique
}

// in returns true if the btcutil.Address exists in the []btcutil.Address,
// false otherwise.
func (a AddressSet) in(addresses []btcutil.Address, addr btcutil.Address) bool {
	s := addr.String()

	for _, address := range addresses {
		// compare the string values
		if address.String() == s {
			return true
		}
	}

	// no duplicate was found
	return false
}
