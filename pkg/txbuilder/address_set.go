package txbuilder

import (
	"github.com/btcsuite/btcutil"
)

// AddressSet is used to create a unique set of addresses from a slice of
// btcutil.Address, where the order of the values in the original slice will
// be preserved.
type AddressSet struct {
	Addresses []btcutil.Address
}

// NewAddressSet returns a new AddressSet with the slice of btcutil.Address.
func NewAddressSet(addresses []btcutil.Address) AddressSet {
	return AddressSet{
		Addresses: addresses,
	}
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
