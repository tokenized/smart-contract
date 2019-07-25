package bitcoin

import (
	"fmt"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	btcdwire "github.com/btcsuite/btcd/wire"
)

const (
	MainNet       wire.BitcoinNet = 0xe8f3e1e3
	TestNet       wire.BitcoinNet = 0xf4f3e5f4
	StressTestNet wire.BitcoinNet = 0xfbcec4f9
	IntNet        wire.BitcoinNet = 0xffffffff // Internal use only. Panics if String is called.
)

var (
	// MainNetParams defines the network parameters for the BSV Main Network.
	MainNetParams chaincfg.Params

	// TestNetParams defines the network parameters for the BSV Test Network.
	TestNetParams chaincfg.Params

	// StressTestNetParams defines the network parameters for the BSV Stress Test Network.
	StressTestNetParams chaincfg.Params
)

func NewChainParams(network string) *chaincfg.Params {
	switch network {
	default:
	case "mainnet":
		return &MainNetParams
	case "testnet":
		return &TestNetParams
	case "stn":
		return &StressTestNetParams
	}

	return nil
}

func init() {
	// setup the MainNet params
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Name = "mainnet"
	MainNetParams.Net = btcdwire.BitcoinNet(MainNet)

	// the params need to be registed to use them.
	if err := chaincfg.Register(&MainNetParams); err != nil {
		fmt.Printf("WARNING failed to register MainNetParams")
	}

	// setup the TestNet params
	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Name = "testnet"
	TestNetParams.Net = btcdwire.BitcoinNet(TestNet)

	// the params need to be registed to use them.
	if err := chaincfg.Register(&TestNetParams); err != nil {
		fmt.Printf("WARNING failed to register TestNetParams")
	}

	// setup the STN params
	StressTestNetParams = chaincfg.TestNet3Params
	StressTestNetParams.Name = "stn"
	StressTestNetParams.Net = btcdwire.BitcoinNet(StressTestNet)

	// the params need to be registed to use them.
	if err := chaincfg.Register(&StressTestNetParams); err != nil {
		fmt.Printf("WARNING failed to register StressTestNetParams")
	}
}
