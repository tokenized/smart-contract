package cmd

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
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
		testnet, _ := c.Flags().GetBool(FlagTestNetMode)
		if testnet {
			params = &chaincfg.TestNet3Params
		} else {
			params = &chaincfg.MainNetParams
		}

		key, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			fmt.Printf("Failed to generate key : %s\n", err)
			return nil
		}

		wif, err := btcutil.NewWIF(key, params, true)
		if err != nil {
			fmt.Printf("Failed to generate WIF : %s\n", err)
			return nil
		}

		fmt.Printf("WIF : %s\n", wif.String())
		return nil
	},
}

func init() {
	cmdGen.Flags().Bool(FlagTestNetMode, false, "Testnet mode")
}
