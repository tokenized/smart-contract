package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	FlagX  = "x"
	FlagXs = "xs"
)

var cmdGen = &cobra.Command{
	Use:   "gen",
	Short: "Generates a bitcoin private key in WIF",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 0 {
			return errors.New("Incorrect argument count")
		}

		extended, _ := c.Flags().GetBool(FlagX)
		if extended {
			key, err := bitcoin.GenerateMasterExtendedKey()
			if err != nil {
				fmt.Printf("Failed to generate extended key : %s\n", err)
				return nil
			}

			fmt.Printf("Private XKey : %s\n", key.String())
			fmt.Printf("Public XKey : %s\n", key.ExtendedPublicKey().String())
			return nil
		}

		extendedMulti, _ := c.Flags().GetBool(FlagXs)
		if extendedMulti {
			key, err := bitcoin.GenerateMasterExtendedKey()
			if err != nil {
				fmt.Printf("Failed to generate extended key : %s\n", err)
				return nil
			}

			keys := bitcoin.ExtendedKeys{key}

			fmt.Printf("Private XKeys : %s\n", keys.String())
			fmt.Printf("Public XKeys : %s\n", keys.ExtendedPublicKeys().String())
			return nil
		}

		network := network(c)
		if network == bitcoin.InvalidNet {
			fmt.Printf("Invalid network specified")
			return nil
		}

		key, err := bitcoin.GenerateKey(network)
		if err != nil {
			fmt.Printf("Failed to generate key : %s\n", err)
			return nil
		}

		address, err := bitcoin.NewAddressPKH(bitcoin.Hash160(key.PublicKey().Bytes()), network)
		if err != nil {
			fmt.Printf("Failed to generate address : %s\n", err)
			return nil
		}

		fmt.Printf("WIF (Private) : %s\n", key.String())
		fmt.Printf("Public Key : %s\n", hex.EncodeToString(key.PublicKey().Bytes()))
		fmt.Printf("Address : %s\n", address.String())
		return nil
	},
}

func init() {
	cmdGen.Flags().Bool(FlagX, false, "generate an extended key")
	cmdGen.Flags().Bool(FlagXs, false, "generate an multi-extended key")
}
