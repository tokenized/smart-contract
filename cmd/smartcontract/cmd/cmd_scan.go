package cmd

import (
	"strconv"

	"github.com/tokenized/smart-contract/cmd/smartcontract/client"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const ()

var cmdScan = &cobra.Command{
	Use:   "scan connections",
	Short: "Scan Bitcoin network for peers.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Incorrect argument count")
		}

		count, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}

		ctx := client.Context()
		if ctx == nil {
			return nil
		}
		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			return err
		}

		return theClient.Scan(ctx, count)
	},
}

func init() {
}
