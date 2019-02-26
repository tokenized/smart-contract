package cmd

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/internal/platform/node"
)

var scCmd = &cobra.Command{
	Use:   "smartcontract",
	Short: "Smart Contract CLI",
}

func Execute() {
	scCmd.AddCommand(cmdSync)
	scCmd.Execute()
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := node.Values{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}

	return context.WithValue(context.Background(), node.KeyValues, &values)
}
