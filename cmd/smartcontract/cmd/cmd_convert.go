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
		if len(network) == 0 {
			return nil
		}
		net := bitcoin.NetworkFromString(network)

		address, err := bitcoin.DecodeAddress(args[0])
		if err == nil {
			fmt.Printf("Hex : %x\n", address.Bytes())
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

		address, err = bitcoin.NewAddressPKH(hash, net)
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
