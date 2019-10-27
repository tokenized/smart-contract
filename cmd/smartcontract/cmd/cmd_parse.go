package cmd

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
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
	Use:   "parse <hex>",
	Short: "Parse a hexadecimal representation of a TX or OP_RETURN script, and output the result.",
	Long:  "Parse a hexadecimal representation of a TX or OP_RETURN script, and output the result.",
	RunE: func(c *cobra.Command, args []string) error {

		var input string
		var err error
		if len(args) == 1 {
			input = args[0]
		} else if len(args) > 1 {
			fmt.Printf("Too many arguments\n")
			return nil
		} else {
			fmt.Printf("Enter hex to decode: ")
			reader := bufio.NewReader(os.Stdin)
			input, err = reader.ReadString('\n') // Get input from stdin
			if err != nil {
				fmt.Printf("Failed to read user input : %s\n", err)
				return nil
			}
		}

		input = strings.TrimSpace(input)

		data, err := hex.DecodeString(input)
		if err != nil {
			fmt.Printf("Failed to decode hex : %s\n", err)
			return nil
		}

		if parseTx(c, data) == nil {
			return nil
		}

		parseScript(c, data)

		return nil
	},
}

func parseTx(c *cobra.Command, rawtx []byte) error {

	tx := wire.MsgTx{}
	buf := bytes.NewReader(rawtx)

	if err := tx.Deserialize(buf); err != nil {
		return errors.Wrap(err, "decode tx")
	}

	fmt.Printf("\nTx (%d bytes) : %s\n", tx.SerializeSize(), tx.TxHash().String())
	dumpJSON(tx)

	for _, txOut := range tx.TxOut {
		if parseScript(c, txOut.PkScript) == nil {
			return nil
		}
	}

	return nil
}

func parseScript(c *cobra.Command, script []byte) error {

	isTest := false
	message, err := protocol.Deserialize(script, isTest)
	if err != nil {
		if err == protocol.ErrNotTokenized {
			// Check is test protocol signature
			isTest = true
			message, err = protocol.Deserialize(script, isTest)
			if err != nil {
				return errors.Wrap(err, "decode op return")
			}
		} else {
			return errors.Wrap(err, "decode op return")
		}
	}

	fmt.Printf("\n")

	if isTest {
		fmt.Printf("Uses Test Protocol Signature\n")
	}

	fmt.Printf("Action type : %s\n\n", message.Code())

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
