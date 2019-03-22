package cmd

import (
	"github.com/tokenized/smart-contract/cmd/smartcontract/client"

	"github.com/spf13/cobra"
)

const (
	FlagDebugMode = "debug"
)

var cmdSync = &cobra.Command{
	Use:   "sync",
	Short: "Syncronize contract state with the network",
	RunE: func(c *cobra.Command, args []string) error {
		ctx := client.Context()
		theClient, err := client.NewClient(ctx)
		if err != nil {
			return err
		}

		return theClient.RunSpyNode(ctx)
	},
}

func init() {
	cmdSync.Flags().Bool(FlagDebugMode, false, "Debug mode")
}
