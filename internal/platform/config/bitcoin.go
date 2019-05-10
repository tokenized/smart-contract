package config

import (
	"github.com/btcsuite/btcd/chaincfg"
)

// NewChainParams returns chain configuration parameters
// based on the supplied string.
//
// - mainnet = Bitcoin SV main network
// - testnet = Bitcoin SV test network
//
func NewChainParams(network string) chaincfg.Params {
	var cfg chaincfg.Params

	switch network {
	default:
	case "mainnet":
		cfg = chaincfg.MainNetParams
		cfg.Net = 0xe8f3e1e3 // BCH MainNet Magic bytes
	case "testnet":
		cfg = chaincfg.TestNet3Params
		cfg.Net = 0xf4f3e5f4 // BCH TestNet Magic bytes
	case "stn":
		cfg = chaincfg.TestNet3Params
		cfg.Net = 0xfbcec4f9 // Scaling TestNet Magic bytes
	}

	return cfg
}
