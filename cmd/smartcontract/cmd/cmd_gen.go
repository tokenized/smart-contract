package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdGen = &cobra.Command{
	Use:   "gen",
	Short: "Generates a bitcoin private key in WIF",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 0 {
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

		key, err := bitcoin.GenerateKeyS256()
		if err != nil {
			fmt.Printf("Failed to generate key : %s\n", err)
			return nil
		}

		address, err := bitcoin.NewAddressPKH(bitcoin.Hash160(key.PublicKey().Bytes()))
		if err != nil {
			fmt.Printf("Failed to generate address : %s\n", err)
			return nil
		}

		fmt.Printf("WIF : %s\n", key.String(wire.BitcoinNet(params.Net)))
		fmt.Printf("PubKey : %s\n", base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()))
		fmt.Printf("Addr : %s\n", address.String(wire.BitcoinNet(params.Net)))
		return nil
	},
}

func init() {
}
