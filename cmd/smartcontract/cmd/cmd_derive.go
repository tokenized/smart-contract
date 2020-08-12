package cmd

import (
	"fmt"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdDerive = &cobra.Command{
	Use:   "derive xkey path",
	Short: "Derives child keys for an extended key",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Incorrect argument count")
		}

		network := network(c)
		if network == bitcoin.InvalidNet {
			fmt.Printf("Invalid network specified")
			return nil
		}

		xkey, err := bitcoin.ExtendedKeyFromStr(args[0])
		if err != nil {
			fmt.Printf("Failed to parse extended key : %s\n", err)
			return nil
		}

		path, err := bitcoin.PathFromString(args[1])
		if err != nil {
			fmt.Printf("Failed to parse path : %s\n", err)
			return nil
		}

		child, err := xkey.ChildKeyForPath(path)
		if err != nil {
			fmt.Printf("Failed to derive child : %s\n", err)
			return nil
		}

		fmt.Printf("XKey : %s\n", child.String())
		if child.IsPrivate() {
			fmt.Printf("WIF (Private) : %s\n", child.Key(network).String())
		}
		fmt.Printf("Public Key : %s\n", child.PublicKey().String())

		ra, err := child.RawAddress()
		if err != nil {
			fmt.Printf("Failed to create address : %s\n", err)
			return nil
		}
		fmt.Printf("Address : %s\n", bitcoin.NewAddressFromRawAddress(ra, network).String())
		return nil
	},
}
