package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdJSON = &cobra.Command{
	Use:     "json message_type args",
	Short:   "Generate JSON to be used a payload for the build command.",
	Long:    "Generate JSON to be used a payload for the build command.",
	Example: "smartcontract json T1 SHC 6259cbd4e0522d8c6539f0a291bfcf4cdad9a5275925571ba1ccbdbe5ac0188d 1GtQEoDE7us5udLWuNCmbngYuwjs12EnwP 90001 # create T1 payload",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Missing type of payload to create.")
		}

		params := networkParams()

		messageType := strings.ToLower(args[0])

		var b []byte
		var err error

		switch messageType {
		case "t1":
			b, err = buildT1(c, args, params)
		default:
			err = fmt.Errorf("Message type not supported yet : %v", messageType)
		}

		if err != nil {
			return err
		}

		fmt.Printf("%s\n", b)

		return nil
	},
}

func buildT1(cmd *cobra.Command,
	args []string,
	params *chaincfg.Params) ([]byte, error) {

	if (len(args) > 1 && args[1] == "help") || len(args) < 5 {
		fmt.Printf("Usage:\n  smartcontract json messagetype asset_type asset_code recipient_address quantity\n\nExample:\n  smartcontract json T1 SHC 6259cbd4e0522d8c6539f0a291bfcf4cdad9a5275925571ba1ccbdbe5ac0188d 1GtQEoDE7us5udLWuNCmbngYuwjs12EnwP 90000")
		return nil, nil
	}

	assetType := args[1]
	assetCode := args[2]
	recipient := args[3]

	qty, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return nil, err
	}

	// A few wrapper structs to make creating the JSON document clearer.
	type sender struct {
		Index    int   `json:"index"`
		Quantity int64 `json:"quantity"`
	}

	type receiver struct {
		Address  string `json:"address"`
		Quantity int64  `json:"quantity"`
	}

	type asset struct {
		Type      string     `json:"asset_type"`
		Code      string     `json:"asset_code"`
		Senders   []sender   `json:"asset_senders"`
		Receivers []receiver `json:"asset_receivers"`
	}

	type wrapper struct {
		Assets []asset `json:"assets"`
	}

	// Get the address. Later we will convert it to a hex encoded
	// representation for the JSON message
	address, err := btcutil.DecodeAddress(recipient, params)
	if err != nil {
		return nil, err
	}

	a := asset{
		Type: assetType,
		Code: assetCode,
		Senders: []sender{
			{
				Index:    0,
				Quantity: qty,
			},
		},
		Receivers: []receiver{
			{
				Address:  fmt.Sprintf("%x", address.ScriptAddress()),
				Quantity: qty,
			},
		},
	}

	w := wrapper{
		Assets: []asset{
			a,
		},
	}

	return json.Marshal(w)
}
