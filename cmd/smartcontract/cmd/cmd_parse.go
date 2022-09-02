package cmd

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
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
			fmt.Printf("Enter hex or ASM to decode: ")
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
			fmt.Printf("Failed to parse script hex : %s\n", err)

			script, err := bitcoin.StringToScript(input)
			if err == nil {
				if parseScript(c, script) == nil {
					return nil
				}

				return nil
			}

			fmt.Printf("Failed to parse script : %s\n", err)
			return nil
		}

		if parseTx(c, data) == nil {
			return nil
		}

		if parseScript(c, data) == nil {
			return nil
		}

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

	// hexBuf := &bytes.Buffer{}
	// tx.Serialize(hexBuf)
	// fmt.Printf("Hex : %x\n", hexBuf.Bytes())

	fmt.Printf(tx.StringWithAddresses(network(c)))

	for _, txOut := range tx.TxOut {
		parseScript(c, txOut.LockingScript)
	}

	return nil
}

func parseScript(c *cobra.Command, script bitcoin.Script) error {

	isTest := false
	message, err := protocol.Deserialize(script, isTest)
	if err != nil {
		if err == protocol.ErrNotTokenized {
			// Check is test protocol signature
			isTest = true
			message, err = protocol.Deserialize(script, isTest)
			if err != nil {
				if err == protocol.ErrNotTokenized {
					fmt.Printf("Script : %s\n", script)
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
	case *actions.InstrumentDefinition:
		if len(m.InstrumentPayload) == 0 {
			fmt.Printf("Empty instrument payload!\n")
			return nil
		}
		instrument, err := instruments.Deserialize([]byte(m.InstrumentType), m.InstrumentPayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s", err)
		} else {
			if err := instrument.Validate(); err != nil {
				fmt.Printf("Instrument is invalid : %s\n", err)
			} else {
				fmt.Printf("Instrument is valid\n")
			}
			dumpJSON(instrument)
		}
	case *actions.InstrumentCreation:
		if len(m.InstrumentPayload) == 0 {
			fmt.Printf("Empty instrument payload!\n")
			return nil
		}
		instrument, err := instruments.Deserialize([]byte(m.InstrumentType), m.InstrumentPayload)
		if err != nil {
			fmt.Printf("Failed to deserialize payload : %s\n", err)
		} else {
			if err := instrument.Validate(); err != nil {
				fmt.Printf("Instrument is invalid : %s\n", err)
			}
			dumpJSON(instrument)
		}

		hash, err := bitcoin.NewHash20(m.InstrumentCode)
		if err != nil {
			fmt.Printf("Invalid hash : %s\n", err)
			return nil
		}
		fmt.Printf("Instrument ID : %s\n", protocol.InstrumentID(m.InstrumentType, *hash))
	case *actions.Transfer:
		for i, a := range m.Instruments {
			if a.InstrumentType == protocol.BSVInstrumentID {
				fmt.Printf("Instrument ID %d : %s\n", i, protocol.BSVInstrumentID)
				continue
			}

			hash, err := bitcoin.NewHash20(a.InstrumentCode)
			if err != nil {
				fmt.Printf("Instrument code %d is invalid : %s\n", i, err)
			} else {
				fmt.Printf("Instrument ID %d : %s\n", i, protocol.InstrumentID(a.InstrumentType, *hash))
			}
		}
	case *actions.Settlement:
		for i, a := range m.Instruments {
			if a.InstrumentType == protocol.BSVInstrumentID {
				fmt.Printf("Instrument ID %d : %s\n", i, protocol.BSVInstrumentID)
				continue
			}

			hash, err := bitcoin.NewHash20(a.InstrumentCode)
			if err != nil {
				fmt.Printf("Instrument code %d is invalid : %s\n", i, err)
			} else {
				fmt.Printf("Instrument ID %d : %s\n", i, protocol.InstrumentID(a.InstrumentType, *hash))
			}
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
}
