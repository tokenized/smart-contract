package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/bootstrap"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdState = &cobra.Command{
	Use:   "state <hex>",
	Short: "Load and print the contract state.",
	Long:  "Load and print the contract state.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Missing hash")
		}

		ctx := bootstrap.NewContextWithDevelopmentLogger()

		cfg := bootstrap.NewConfigFromEnv(ctx)

		params := config.NewChainParams(cfg.Bitcoin.Network)

		pkh, err := btcutil.DecodeAddress(args[0], &params)
		if err != nil {
			return err
		}

		masterDB := bootstrap.NewMasterDB(ctx, cfg)

		return loadContract(ctx, c, masterDB, pkh, &params)
	},
}

func loadContract(ctx context.Context,
	cmd *cobra.Command,
	db *db.DB,
	address btcutil.Address,
	params *chaincfg.Params) error {

	pkh := protocol.PublicKeyHashFromBytes(address.ScriptAddress())

	c, err := contract.Fetch(ctx, db, pkh)
	if err != nil {
		return err
	}

	fmt.Printf("# Contract %s\n\n", address)

	if err := dumpJSON(c); err != nil {
		return err
	}

	for _, assetCode := range c.AssetCodes {
		a, err := asset.Fetch(ctx, db, &c.ID, &assetCode)
		if err != nil {
			return err
		}

		fmt.Printf("## Asset %x\n\n", a.ID.Bytes())

		if err := dumpJSON(a); err != nil {
			return err
		}

		payload := protocol.AssetTypeMapping(a.AssetType)

		if _, err := payload.Write(a.AssetPayload); err != nil {
			return err
		}

		fmt.Printf("### Payload\n\n")

		if err := dumpJSON(payload); err != nil {
			return err
		}

		fmt.Printf("### Holdings\n\n")

		// get the PKH's inside the holders/asset_code directory
		keys, err := holdings.List(ctx, db, pkh, &assetCode)
		if err != nil {
			return nil
		}

		for _, key := range keys {
			// split the key into parts.
			parts := strings.Split(key, "/")

			// the last part of the key is the PKH of the owner, in hex format.
			owner := parts[len(parts)-1]

			// Convert the hex representation to an Address, which is used
			// for display purposes.
			address, err := addressFromHex(owner, params)
			if err != nil {
				return err
			}

			// create the PKH from the bytes
			ownerPKH := protocol.PublicKeyHashFromBytes(address.ScriptAddress())

			// we can get the holding for this owner now
			h, err := holdings.Fetch(ctx, db, pkh, &assetCode, ownerPKH)
			if err != nil {
				return err
			}

			fmt.Printf("#### %s\n\n", address)

			if err := dumpJSON(h); err != nil {
				return err
			}
		}

	}

	return nil
}

// addressFromHex returns a decoded Address from a the hex representation of
// a PKH.
func addressFromHex(s string, params *chaincfg.Params) (btcutil.Address, error) {

	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return btcutil.NewAddressPubKeyHash(b, params)
}
