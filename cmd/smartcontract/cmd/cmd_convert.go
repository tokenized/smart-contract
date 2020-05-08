package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/protocol"

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

		assetType, assetCode, err := protocol.DecodeAssetID(args[0])
		if err == nil {
			fmt.Printf("Asset %s : %x\n", assetType, assetCode.Bytes())
			return nil
		}

		network := network(c)
		if network == bitcoin.InvalidNet {
			fmt.Printf("Invalid network specified")
			return nil
		}

		key, err := bitcoin.KeyFromStr(args[0])
		if err == nil {
			fmt.Printf("PublicKey : %s\n", key.PublicKey().String())
			ra, _ := key.RawAddress()
			fmt.Printf("Address : %s\n", bitcoin.NewAddressFromRawAddress(ra, network).String())
			return nil
		}

		if len(args[0]) == 66 {
			hash, _ := hex.DecodeString(args[0])
			ra, _ := bitcoin.NewRawAddressPKH(bitcoin.Hash160(hash))
			fmt.Printf("Address : %s\n", bitcoin.NewAddressFromRawAddress(ra, network).String())
			return nil
		}

		address, err := bitcoin.DecodeAddress(args[0])
		if err == nil {
			ra := bitcoin.NewRawAddressFromAddress(address)
			fmt.Printf("Raw Address : %x\n", ra.Bytes())

			if pk, err := ra.GetPublicKey(); err == nil {
				fmt.Printf("Public Key : %s\n", pk.String())

				pkh, _ := bitcoin.NewHash20(bitcoin.Hash160(pk.Bytes()))
				fmt.Printf("Public Key Hash : %s\n", pkh.String())

				pkhRa, err := bitcoin.NewRawAddressPKH(pkh.Bytes())
				if err == nil {
					fmt.Printf("P2PKH address : %s\n",
						bitcoin.NewAddressFromRawAddress(pkhRa, network).String())
				}
			}

			if pkh, err := ra.GetPublicKeyHash(); err == nil {
				fmt.Printf("Public Key Hash : %s\n", pkh.String())
			}

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
