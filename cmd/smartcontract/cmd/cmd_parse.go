package cmd

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontract/client"
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

	send, _ := c.Flags().GetBool(FlagSend)
	if send {
		fmt.Printf("Sending to network\n")
		ctx := client.Context()
		if ctx == nil {
			fmt.Printf("Failed to create client context\n")
			return nil
		}

		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			fmt.Printf("Failed to create client : %s\n", err)
			return nil
		}

		if err := theClient.BroadcastTx(ctx, &tx); err != nil {
			fmt.Printf("Failed to send tx : %s\n", err)
		}
	}

	fmt.Printf("\nTx (%d bytes) : %s\n", tx.SerializeSize(), tx.TxHash().String())
	fmt.Printf(tx.StringWithAddresses(network(c)))

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
				if err == protocol.ErrNotTokenized {
					r := bytes.NewReader(script)
					for i := 0; i < 100; i++ {
						_, pushdata, err := bitcoin.ParsePushDataScript(r)
						if err == nil {
							fmt.Printf("OP %02d %x\n", i, pushdata)
							continue
						}

						if err == io.EOF { // finished parsing script
							return nil
						}
						if err != bitcoin.ErrNotPushOp { // ignore non push op codes
							return errors.Wrap(err, "decode bitcoin script")
						}
					}
				}
				return errors.Wrap(err, "decode op return")
			}
		} else {
			return errors.Wrap(err, "decode op return")
		}
	}

	fmt.Printf("\n")

	if isTest {
		fmt.Printf("Uses Test Protocol Signature\n")
	} else {
		fmt.Printf("Uses Production Protocol Signature\n")
	}

	fmt.Printf("Action type : %s\n\n", message.Code())

	if err := message.Validate(); err != nil {
		fmt.Printf("Action is invalid : %s\n", err)
	} else {
		fmt.Printf("Action is valid\n")
	}

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
			if err := asset.Validate(); err != nil {
				fmt.Printf("Asset is invalid : %s\n", err)
			} else {
				fmt.Printf("Asset is valid\n")
			}
			dumpJSON(asset)
		}
	case *actions.AssetCreation:
		if len(m.AssetPayload) == 0 {
			fmt.Printf("Empty asset payload!\n")
			return nil
		}
		asset, err := assets.Deserialize([]byte(m.AssetType), m.AssetPayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s\n", err)
		} else {
			if err := asset.Validate(); err != nil {
				fmt.Printf("Asset is invalid : %s\n", err)
			}
			dumpJSON(asset)
		}

		hash, err := bitcoin.NewHash20(m.AssetCode)
		if err != nil {
			fmt.Printf("Invalid hash : %s\n", err)
			return nil
		}
		fmt.Printf("Asset ID : %s\n", protocol.AssetID(m.AssetType, *hash))
	case *actions.Transfer:
		for i, a := range m.Assets {
			hash, err := bitcoin.NewHash20(a.AssetCode)
			if err != nil {
				fmt.Printf("Invalid hash : %s\n", err)
				return nil
			}
			fmt.Printf("Asset ID %d : %s\n", i, protocol.AssetID(a.AssetType, *hash))
		}
	case *actions.Settlement:
		for i, a := range m.Assets {
			hash, err := bitcoin.NewHash20(a.AssetCode)
			if err != nil {
				fmt.Printf("Invalid hash : %s\n", err)
				return nil
			}
			fmt.Printf("Asset ID %d : %s\n", i, protocol.AssetID(a.AssetType, *hash))
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

			switch p := msg.(type) {
			case *messages.Offer:
				fmt.Printf("\nEmbedded offer tx:\n")
				parseTx(c, p.Payload)
			case *messages.SignatureRequest:
				fmt.Printf("\nEmbedded signature request tx:\n")
				parseTx(c, p.Payload)
			case *messages.RevertedTx:
				fmt.Printf("\nEmbedded Reverted tx:\n")
				parseTx(c, p.Transaction)
			case *messages.SettlementRequest:
				fmt.Printf("\nEmbedded settlement:\n")

				action, err := protocol.Deserialize(p.Settlement, isTest)
				if err != nil {
					fmt.Printf("Failed to deserialize settlement from settlement request : %s\n",
						err)
					return nil
				}

				settlement, ok := action.(*actions.Settlement)
				if !ok {
					fmt.Printf("Settlement Request payload not a settlement\n")
					return nil
				}

				dumpJSON(settlement)
			}
		}
	}

	return nil
}

func init() {
	cmdParse.Flags().Bool(FlagSend, false, "send to network")
}
