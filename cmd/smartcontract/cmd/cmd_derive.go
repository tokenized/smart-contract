package cmd

import (
	"fmt"
	"strconv"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdDerive = &cobra.Command{
	Use:   "derive xkey path",
	Short: "Derives child keys for an extended key",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 && len(args) != 3 {
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

		if len(args) == 3 {
			count, err := strconv.Atoi(args[2])
			if err != nil {
				fmt.Printf("Failed to parse count : %s\n", err)
				return nil
			}

			for i := uint32(0); i < uint32(count); i++ {
				child, err := xkey.ChildKeyForPath(path)
				if err != nil {
					fmt.Printf("Failed to derive child : %s\n", err)
					return nil
				}

				if child.IsPrivate() {
					key := child.Key(network)
					fmt.Printf("Key (+%d) : %s\n", i, key.String())
				}

				ra, err := child.RawAddress()
				if err != nil {
					fmt.Printf("Failed to create address : %s\n", err)
					return nil
				}
				fmt.Printf("Address (+%d) : %s\n", i,
					bitcoin.NewAddressFromRawAddress(ra, network).String())

				path[len(path)-1] += 1
			}

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
