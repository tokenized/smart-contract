package config

import (
	"github.com/btcsuite/btcutil"
)

type Fee struct {
	Address btcutil.Address
	Value   uint64
}
