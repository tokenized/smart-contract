package bitcoin

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	btcdwire "github.com/btcsuite/btcd/wire"
)

type Network uint32

const (
	MainNet       Network = 0xe8f3e1e3
	TestNet       Network = 0xf4f3e5f4
	StressTestNet Network = 0xf9c4cefb
	InvalidNet    Network = 0x00000000
)

var (
	// MainNetParams defines the network parameters for the BSV Main Network.
	MainNetParams chaincfg.Params

	// TestNetParams defines the network parameters for the BSV Test Network.
	TestNetParams chaincfg.Params

	// StressTestNetParams defines the network parameters for the BSV Stress Test Network.
	StressTestNetParams chaincfg.Params
)

func NetworkFromString(name string) Network {
	switch name {
	case "mainnet":
		return MainNet
	case "testnet":
		return TestNet
	case "stn":
		return StressTestNet
	}

	return InvalidNet
}

func NetworkName(net Network) string {
	switch net {
	case MainNet:
		return "mainnet"
	case TestNet:
		return "testnet"
	case StressTestNet:
		return "stn"
	}

	return "testnet"
}

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
