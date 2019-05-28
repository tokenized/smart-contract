package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ripemd160"
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

		hash160 := ripemd160.New()
		hash256 := sha256.Sum256(key.PubKey().SerializeCompressed())
		hash160.Write(hash256[:])
		address, err := btcutil.NewAddressPubKeyHash(hash160.Sum(nil), params)
		if err != nil {
			fmt.Printf("Failed to generate address : %s\n", err)
			return nil
		}

		fmt.Printf("WIF : %s\n", wif.String())
		fmt.Printf("PubKey : %s\n", base64.StdEncoding.EncodeToString(key.PubKey().SerializeCompressed()))
		fmt.Printf("Addr : %s\n", address.String())
		return nil
	},
}

func init() {
}
