package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdParse = &cobra.Command{
	Use:   "parse <hex encoded op return>",
	Short: "Parse hex op return script into text.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Incorrect parameter count")
		}

		data, err := hex.DecodeString(args[0])
		if err != nil {
			panic(err)
		}

		message, err := protocol.Deserialize(data, true)
		if err != nil {
			panic(err)
		}

		fmt.Println(message.String())

		return nil
	},
}
