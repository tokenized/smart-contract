package cmd

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	FlagNetMode = "network"
)

var scCmd = &cobra.Command{
	Use:   "smartcontract",
	Short: "Smart Contract CLI",
}

func Execute() {
	scCmd.AddCommand(cmdSync)
	scCmd.AddCommand(cmdScan)
	scCmd.AddCommand(cmdBuild)
	scCmd.AddCommand(cmdAuth)
	scCmd.AddCommand(cmdConvert)
	scCmd.AddCommand(cmdSign)
	scCmd.AddCommand(cmdCode)
	scCmd.AddCommand(cmdGen)
	scCmd.AddCommand(cmdBench)
	scCmd.Flags().StringP(FlagNetMode, FlagNetMode[:1], "testnet", "Network, i.e. testnet/mainnet")
	scCmd.Execute()
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := node.Values{
		TraceID: uuid.New().String(),
		Now:     protocol.CurrentTimestamp(),
	}

	return context.WithValue(context.Background(), node.KeyValues, &values)
}

// network returns the network string. It is necessary because cobra default values don't seem to work.
func network(c *cobra.Command) string {
	network, err := c.Flags().GetString(FlagNetMode)
	if err != nil {
		return "testnet"
	}
	return network
}
