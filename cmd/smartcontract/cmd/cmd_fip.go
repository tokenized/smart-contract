package cmd

import (
	"fmt"
	"strconv"

	"github.com/tokenized/specification/dist/golang/permissions"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdFIP = &cobra.Command{
	Use:   "fip index ...",
	Short: "Convert protocol field index path to hex",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("Incorrect argument count")
		}

		indexes := make(permissions.FieldIndexPath, 0, len(args))
		for _, arg := range args {
			index, err := strconv.Atoi(arg)
			if err != nil {
				fmt.Printf("Invalid index : %s\n", arg)
				return nil
			}
			indexes = append(indexes, uint32(index))
		}

		b, _ := indexes.Bytes()
		fmt.Printf("Field index path bytes : %x\n", b)
		return nil
	},
}

func init() {
}
