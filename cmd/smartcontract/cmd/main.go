package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
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
	scCmd.AddCommand(cmdDoubleSpend)
	scCmd.AddCommand(cmdParse)
	scCmd.AddCommand(cmdState)
	scCmd.AddCommand(cmdJSON)
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
	network := os.Getenv("BITCOIN_CHAIN")
	if len(network) == 0 {
		fmt.Printf("WARNING!! No Bitcoin network specified. Set environment value BITCOIN_CHAIN=testnet\n")
	}
	return network
}

// dumpJSON pretty prints a JSON representation of a struct.
func dumpJSON(o interface{}) error {
	js, err := json.MarshalIndent(o, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("```\n%s\n```\n\n", js)

	return nil
}
