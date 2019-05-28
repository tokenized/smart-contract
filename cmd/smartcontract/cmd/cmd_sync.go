package cmd

import (
	"github.com/tokenized/smart-contract/cmd/smartcontract/client"

	"github.com/spf13/cobra"
)

const (
	FlagDebugMode = "debug"
	FlagNoStop    = "nostop"
)

var cmdSync = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize with the Bitcoin network.",
	Long:  "Synchronize with the Bitcoin network. This is required after any txs effect the wallet are posted to update UTXOs so that valid/spendable txs can be created.",
	RunE: func(c *cobra.Command, args []string) error {
		ctx := client.Context()
		if ctx == nil {
			return nil
		}
		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			return err
		}

		dontStopOnSync, _ := c.Flags().GetBool(FlagNoStop)
		return theClient.RunSpyNode(ctx, !dontStopOnSync)
	},
}

func init() {
	cmdSync.Flags().Bool(FlagDebugMode, false, "Debug mode")
	cmdSync.Flags().Bool(FlagNoStop, false, "Don't stop on sync")
}
