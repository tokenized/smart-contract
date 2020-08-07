package cmd

import (
	"github.com/spf13/cobra"
)

var cmdCode = &cobra.Command{
	Use:   "code",
	Short: "Run random code",
	RunE: func(c *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
}
