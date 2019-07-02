package cmd

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/bootstrap"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
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

		// 1JWUgphZHD9HXfWXkDNwiCpNUv7zkuZvUC
		pkh, err := btcutil.DecodeAddress(args[0], &params)
		if err != nil {
			return err
		}

		masterDB := bootstrap.NewMasterDB(ctx, cfg)

		return loadContract(ctx, c, masterDB, pkh)
	},
}

func loadContract(ctx context.Context,
	cmd *cobra.Command,
	db *db.DB,
	address btcutil.Address) error {

	pkh := protocol.PublicKeyHashFromBytes(address.ScriptAddress())

	c, err := contract.Fetch(ctx, db, pkh)
	if err != nil {
		return err
	}

	fmt.Printf("# Contract %x\n\n", c.ID.Bytes())

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
	}

	{
		b := c.ID.Bytes()
		fmt.Printf("len = %v : %v\n", len(b), b)
	}

	contractRoot := fmt.Sprintf("%s", c.ID.Bytes())

	keys, err := db.List(ctx, contractRoot)
	if err != nil {
		return err
	}

	fmt.Printf("keys = %+v\n", keys)

	// holdings := []state.Holding{}

	// h, err := holdings.Fetch(ctx, db, contract.ID, assetCode)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("h = %+v\n", h)

	// fmt.Printf("# Contract\n")
	// fmt.Printf("```\n%#+v\n```\n", contract)

	return nil
}
