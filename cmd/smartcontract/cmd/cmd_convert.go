package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

var cmdConvert = &cobra.Command{
	Use:   "convert [address/hash]",
	Short: "Convert bitcoin addresses to hashes and vice versa",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Incorrect argument count")
		}

		network := network(c)
		if network == bitcoin.InvalidNet {
			fmt.Printf("Invalid network specified")
			return nil
		}

		if len(args[0]) == 66 {
			hash, _ := hex.DecodeString(args[0])
			ra, _ := bitcoin.NewRawAddressPKH(bitcoin.Hash160(hash))
			fmt.Printf("Address : %s\n", bitcoin.NewAddressFromRawAddress(ra, bitcoin.MainNet).String())
			return nil
		}

		address, err := bitcoin.DecodeAddress(args[0])
		if err == nil {
			h, err := address.Hash()
			if err != nil {
				fmt.Printf("%s\n", err)
				return nil
			}
			fmt.Printf("Hex : %x\n", h[:])
			return nil
		}

		if len(args[0]) > 42 {
			fmt.Printf("Invalid hash size : %s\n", err)
			return nil
		}

		hash := make([]byte, 21)
		n, err := hex.Decode(hash, []byte(args[0]))
		if err != nil {
			fmt.Printf("Invalid hash : %s\n", err)
			return nil
		}
		if n == 20 {
			address, err = bitcoin.NewAddressPKH(hash[:20], network)
			if err != nil {
				fmt.Printf("Invalid hash : %s\n", err)
				return nil
			}
			fmt.Printf("Address : %s\n", address.String())
			return nil
		}
		if n == 21 {
			rawAddress, err := bitcoin.DecodeRawAddress(hash)
			if err != nil {
				fmt.Printf("Invalid hash : %s\n", err)
				return nil
			}
			address := bitcoin.NewAddressFromRawAddress(rawAddress, network)
			fmt.Printf("Address : %s\n", address.String())
			return nil
		}

		fmt.Printf("Invalid hash size : %d\n", n)
		return nil
	},
}

func init() {
}
