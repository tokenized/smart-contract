package cmd

import (
	"github.com/spf13/cobra"
)

var scCmd = &cobra.Command{
	Use:   "smartcontract",
	Short: "Smart Contract CLI",
}

func Execute() {
	scCmd.AddCommand(cmdSync)
	scCmd.Execute()
}
