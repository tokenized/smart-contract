package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

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

		network := network(c)
		if network == bitcoin.InvalidNet {
			fmt.Printf("Invalid network specified")
			return nil
		}

		key, err := bitcoin.GenerateKeyS256(network)
		if err != nil {
			fmt.Printf("Failed to generate key : %s\n", err)
			return nil
		}

		address, err := bitcoin.NewAddressPKH(bitcoin.Hash160(key.PublicKey().Bytes()), network)
		if err != nil {
			fmt.Printf("Failed to generate address : %s\n", err)
			return nil
		}

		fmt.Printf("WIF : %s\n", key.String())
		fmt.Printf("PubKey : %s\n", base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()))
		fmt.Printf("Addr : %s\n", address.String())
		return nil
	},
}

func init() {
}
