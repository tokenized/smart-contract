package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdConvert = &cobra.Command{
	Use:   "convert [address/hash]",
	Short: "Convert bitcoin addresses to hashes and vice versa",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 2 {
			b, err := hex.DecodeString(args[1])
			if err == nil {
				assetID := protocol.AssetID(args[0], *protocol.AssetCodeFromBytes(b))
				fmt.Printf("Asset ID : %s\n", assetID)
				return nil
			}

		}

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

		b, err := hex.DecodeString(args[0])
		if err == nil {
			if len(b) == 32 {
				// Reverse hash
				fmt.Printf("Reverse hash : %x\n", reverse32(b))
				return nil
			} else if len(b) == 20 {
				// Reverse hash
				fmt.Printf("Reverse hash : %x\n", reverse20(b))
				return nil
			}

			xkey, err := bitcoin.ExtendedKeyFromBytes(b)
			if err == nil {
				if xkey.IsPrivate() {
					fmt.Printf("Private extended key : ")
				} else {
					fmt.Printf("Public extended key : ")
				}
				fmt.Printf("%s\n", xkey.String())

				fmt.Printf("Plural : %s\n", bitcoin.ExtendedKeys{xkey}.String())
				return nil
			}

			xkeys, err := bitcoin.ExtendedKeysFromBytes(b)
			if err == nil {
				if xkeys[0].IsPrivate() {
					fmt.Printf("Private extended key : ")
				} else {
					fmt.Printf("Public extended key : ")
				}
				fmt.Printf("%s\n", xkeys.String())
				return nil
			}
		}

		key, err := bitcoin.KeyFromStr(args[0])
		if err == nil {
			fmt.Printf("Public key : %s\n", key.PublicKey().String())
			ra, _ := key.RawAddress()
			fmt.Printf("Address : %s\n", bitcoin.NewAddressFromRawAddress(ra, network).String())
			return nil
		}

		xkey, err := bitcoin.ExtendedKeyFromStr(args[0])
		if err == nil {
			fmt.Printf("Extended public key : %s\n", xkey.ExtendedPublicKey().String())

			fmt.Printf("Plural : %s\n", bitcoin.ExtendedKeys{xkey}.String())
			return nil
		}

		xkeys, err := bitcoin.ExtendedKeysFromStr(args[0])
		if err == nil {
			fmt.Printf("Extended public key : %s\n", xkeys.ExtendedPublicKeys().String())
			fmt.Printf("Raw : %x\n", xkeys.Bytes())
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

func reverse32(h []byte) []byte {
	r := make([]byte, 32)
	i := 31
	for _, b := range h[:] {
		r[i] = b
		i--
	}
	return r
}

func reverse20(h []byte) []byte {
	r := make([]byte, 20)
	i := 19
	for _, b := range h[:] {
		r[i] = b
		i--
	}
	return r
}
