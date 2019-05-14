package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdConvert = &cobra.Command{
	Use:   "convert [address/hash]",
	Short: "Convert bitcoin addresses to hashes and vice versa",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Incorrect argument count")
		}

		var params *chaincfg.Params
		network := network(c)
		if len(network) == 0 {
			return nil
		}
		if network == "testnet" {
			params = &chaincfg.TestNet3Params
		} else if network == "mainnet" {
			params = &chaincfg.MainNetParams
		} else {
			fmt.Printf("Unknown network : %s\n", network)
			return nil
		}

		address, err := btcutil.DecodeAddress(args[0], params)
		if err == nil {
			fmt.Printf("Hash : %x\n", address.ScriptAddress())
			return nil
		}

		hash := make([]byte, 20)
		n, err := hex.Decode(hash, []byte(args[0]))
		if err != nil {
			fmt.Printf("Invalid hash : %s\n", err)
			return nil
		}
		if n != 20 {
			fmt.Printf("Invalid hash size : %d\n", n)
			return nil
		}

		address, err = btcutil.NewAddressPubKeyHash(hash, params)
		if err != nil {
			fmt.Printf("Invalid hash : %s\n", err)
			return nil
		}
		fmt.Printf("Address : %s\n", address.String())

		return nil
	},
}

func init() {
}
