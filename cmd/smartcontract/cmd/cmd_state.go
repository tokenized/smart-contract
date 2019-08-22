package cmd

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/bootstrap"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/assets"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

		address, err := bitcoin.DecodeAddress(args[0])
		if err != nil {
			return err
		}

		masterDB := bootstrap.NewMasterDB(ctx, cfg)

		return loadContract(ctx, c, masterDB, address,
			bitcoin.NetworkFromString(cfg.Bitcoin.Network))
	},
}

func loadContract(ctx context.Context,
	cmd *cobra.Command,
	db *db.DB,
	address bitcoin.Address,
	net bitcoin.Network) error {

	c, err := contract.Fetch(ctx, db, address)
	if err != nil {
		return err
	}

	fmt.Printf("# Contract %s\n\n", address.String())

	if err := dumpJSON(c); err != nil {
		return err
	}

	for _, assetCode := range c.AssetCodes {
		a, err := asset.Fetch(ctx, db, c.Address, assetCode)
		if err != nil {
			return err
		}

		fmt.Printf("## Asset %x\n\n", a.Code.Bytes())

		if err := dumpJSON(a); err != nil {
			return err
		}

		asset, err := assets.Deserialize([]byte(a.AssetType), a.AssetPayload)
		if err != nil {
			return err
		}

		fmt.Printf("### Payload\n\n")

		if err := dumpJSON(asset); err != nil {
			return err
		}

		fmt.Printf("### Holdings\n\n")

		// get the PKH's inside the holders/asset_code directory
		holdings, err := holdings.FetchAll(ctx, db, address, assetCode)
		if err != nil {
			return nil
		}

		for _, holding := range holdings {
			address = bitcoin.NewAddressFromRawAddress(holding.Address, net)
			fmt.Printf("#### Holding for %s\n\n", address.String())

			if err := dumpHoldingJSON(holding); err != nil {
				return err
			}
		}
	}

	return nil
}

// dumpHoldingJSON dumps a Holding to JSON.
//
// As the json package requires map keys to be strings, this special function
// handles key converstion.
func dumpHoldingJSON(h *state.Holding) error {
	holdingStatuses := h.HoldingStatuses
	h.HoldingStatuses = nil

	if err := dumpJSON(h); err != nil {
		return err
	}

	if len(holdingStatuses) == 0 {
		return nil
	}

	// deal with key conversion.
	statuses := map[string]state.HoldingStatus{}

	for _, s := range holdingStatuses {
		k := fmt.Sprintf("%x", s.TxId.Bytes())
		statuses[k] = *s
	}

	fmt.Printf("#### Holding Statuses\n\n")

	return dumpJSON(statuses)
}
