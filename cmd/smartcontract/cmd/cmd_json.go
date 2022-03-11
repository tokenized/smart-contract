package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdJSON = &cobra.Command{
	Use:     "json message_type args",
	Short:   "Generate JSON to be used as a payload for the build command.",
	Long:    "Generate JSON to be used as a payload for the build command.",
	Example: "smartcontract json T1 SHC 6259cbd4e0522d8c6539f0a291bfcf4cdad9a5275925571ba1ccbdbe5ac0188d 1GtQEoDE7us5udLWuNCmbngYuwjs12EnwP 90001 # create T1 payload",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Missing type of payload to create.")
		}

		messageType := strings.ToLower(args[0])

		var b []byte
		var err error

		switch messageType {
		case "t1":
			b, err = buildT1(c, args)
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
	args []string) ([]byte, error) {

	if (len(args) > 1 && args[1] == "help") || len(args) < 5 {
		fmt.Printf("Usage:\n  smartcontract json messagetype instrument_type instrument_code recipient_address quantity\n\nExample:\n  smartcontract json T1 SHC 6259cbd4e0522d8c6539f0a291bfcf4cdad9a5275925571ba1ccbdbe5ac0188d 1GtQEoDE7us5udLWuNCmbngYuwjs12EnwP 90000")
		return nil, nil
	}

	instrumentType := args[1]
	instrumentCode := args[2]
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

	type instrument struct {
		Type      string     `json:"instrument_type"`
		Code      string     `json:"instrument_code"`
		Senders   []sender   `json:"instrument_senders"`
		Receivers []receiver `json:"instrument_receivers"`
	}

	type wrapper struct {
		Instruments []instrument `json:"instruments"`
	}

	// Get the address. Later we will convert it to a hex encoded
	// representation for the JSON message
	address, err := bitcoin.DecodeAddress(recipient)
	if err != nil {
		return nil, err
	}

	a := instrument{
		Type: instrumentType,
		Code: instrumentCode,
		Senders: []sender{
			{
				Index:    0,
				Quantity: qty,
			},
		},
		Receivers: []receiver{
			{
				Address:  address.String(),
				Quantity: qty,
			},
		},
	}

	w := wrapper{
		Instruments: []instrument{
			a,
		},
	}

	return json.Marshal(w)
}
