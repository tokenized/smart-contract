package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdParse = &cobra.Command{
	Use:   "parse <hex> [isTest boolean optional]",
	Short: "Parse a hexadecimal representation of a TX, and output the result.",
	Long:  "Parse a hexadecimal representation of a TX, and output the result.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Missing hex input")
		}

		input := args[0]

		// optional isTest argument
		isTest := false
		if len(args) == 2 && args[1] == "true" {
			isTest = true
		}

		return parseMessage(c, input, isTest)
	},
}

func parseMessage(c *cobra.Command, rawtx string, isTest bool) error {
	data := strings.Trim(string(rawtx), "\n ")

	// setup the tx
	b, err := hex.DecodeString(data)
	if err != nil {
		panic("Failed to decode payload")
	}

	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	m := loadMessageFromTX(tx, isTest)

	if m == nil {
		return nil
	}

	fmt.Printf("type=%v : %+v\n", m.Type(), m)

	return nil
}

func loadMessageFromTX(tx wire.MsgTx, isTest bool) protocol.OpReturnMessage {
	for _, txOut := range tx.TxOut {
		m, err := protocol.Deserialize(txOut.PkScript, isTest)
		if err != nil {
			continue
		}

		if m == nil {
			// this isn't something we are intersted in
			continue
		}

		return m
	}

	return nil
}
