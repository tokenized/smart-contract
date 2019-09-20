package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tokenized/smart-contract/pkg/wire"
	
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/messages"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdParse = &cobra.Command{
	Use:   "parse <hex> [isTest boolean optional]",
	Short: "Parse a hexadecimal representation of a TX or OP_RETURN script, and output the result.",
	Long:  "Parse a hexadecimal representation of a TX or OP_RETURN script, and output the result.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Missing hex input")
		}

		input := strings.Trim(string(args[0]), "\n ")

		// optional isTest argument
		isTest := false
		if len(args) == 2 && args[1] == "true" {
			fmt.Printf("Using test\n")
			isTest = true
		}

		data, err := hex.DecodeString(input)
		if err != nil {
			fmt.Printf("Failed to decode hex : %s\n", err)
		}

		if parseTx(c, data, isTest) == nil {
			return nil
		}

		parseScript(c, data, isTest)

		return nil
	},
}

func parseTx(c *cobra.Command, rawtx []byte, isTest bool) error {

	tx := wire.MsgTx{}
	buf := bytes.NewReader(rawtx)

	if err := tx.Deserialize(buf); err != nil {
		return errors.Wrap(err, "decode tx")
	}

	for _, txOut := range tx.TxOut {
		if parseScript(c, txOut.PkScript, isTest) == nil {
			return nil
		}
	}

	return nil
}

func parseScript(c *cobra.Command, script []byte, isTest bool) error {

	message, err := protocol.Deserialize(script, isTest)
	if err != nil {
		return errors.Wrap(err, "decode op return")
	}

	fmt.Printf("type : %s\n\n", message.Code())

	if err := dumpJSON(message); err != nil {
		return err
	}

	switch m := message.(type) {
	case *actions.AssetDefinition:
		if len(m.AssetPayload) == 0 {
			fmt.Printf("Empty asset payload!\n")
			return nil
		}
		asset, err := assets.Deserialize([]byte(m.AssetType), m.AssetPayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s", err)
		} else {
			dumpJSON(asset)
		}
	case *actions.AssetCreation:
		if len(m.AssetPayload) == 0 {
			fmt.Printf("Empty asset payload!\n")
			return nil
		}
		asset, err := assets.Deserialize([]byte(m.AssetType), m.AssetPayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s", err)
		} else {
			dumpJSON(asset)
		}
	case *actions.Message:
		if len(m.MessagePayload) == 0 {
			fmt.Printf("Empty message payload!\n")
			return nil
		}
		msg, err := messages.Deserialize(m.MessageCode, m.MessagePayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s", err)
		} else {
			dumpJSON(msg)
		}
	}

	return nil
}
