package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdCode = &cobra.Command{
	Use:   "code",
	Short: "Run random code",
	RunE: func(c *cobra.Command, args []string) error {
		jsonText := "{ \"name\" : \"KYC Oracle\", \"public_key\" : \"AtoGHJyVGRyHTxu+0qCikrIe97D1O+Oa5RRCOKA43TJF\" }"

		oracle := protocol.Oracle{}
		if err := json.Unmarshal([]byte(jsonText), &oracle); err != nil {
			fmt.Printf("Failed to unmarshal json : %s\n", err)
			return nil
		}

		fmt.Printf("Oracle %+v\n", oracle)

		data, err := oracle.Serialize()
		if err != nil {
			fmt.Printf("Failed to serialize oracle : %s\n", err)
			return nil
		}
		fmt.Printf("Data : %s\n", base64.StdEncoding.EncodeToString(data))
		return nil
	},
}

func init() {
}
