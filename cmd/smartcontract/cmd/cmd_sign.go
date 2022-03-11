package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdSign = &cobra.Command{
	Use:   "sign <jsonFile> <contract address> <receiverIndex> <blockhash> <oracle wifkey>",
	Short: "Provide oracle signature",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 5 {
			return errors.New("Invalid parameter count")
		}

		return transferSign(c, args)
	},
}

func transferSign(c *cobra.Command, args []string) error {
	// Create struct
	action := actions.NewActionFromCode(actions.CodeTransfer)
	if action == nil {
		fmt.Printf("Unsupported action type : %s\n", actions.CodeTransfer)
		return nil
	}

	// Read json file
	path := filepath.FromSlash(args[0])
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read json file : %s\n", err)
		return nil
	}

	// Put json data into action struct
	if err := json.Unmarshal(data, action); err != nil {
		fmt.Printf("Failed to unmarshal %s json file : %s\n", actions.CodeTransfer, err)
		return nil
	}

	// Contract key
	contractAddress, err := bitcoin.DecodeAddress(args[1])
	if err != nil {
		fmt.Printf("Invalid contract address : %s\n", err)
		return nil
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	receiverIndex, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Printf("Invalid receiver index : %s\n", err)
		return nil
	}

	// Block hash
	blockHash, err := bitcoin.NewHash32FromStr(args[3])
	if err != nil {
		fmt.Printf("Invalid block hash : %s\n", err)
		return nil
	}

	key, err := bitcoin.KeyFromStr(args[4])
	if err != nil {
		fmt.Printf("Invalid key : %s\n", err)
		return nil
	}

	transfer, ok := action.(*actions.Transfer)
	if !ok {
		fmt.Printf("Not a transfer\n")
		return nil
	}

	index := 0
	for _, instrument := range transfer.Instruments {
		for _, receiver := range instrument.InstrumentReceivers {
			if index == receiverIndex {
				receiverAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
				if err != nil {
					fmt.Printf("Failed to decode address : %s\n", err)
					return nil
				}
				fmt.Printf("Signing for address quantity %d : %x\n", receiver.Quantity,
					receiverAddress.Bytes())
				hash, err := protocol.TransferOracleSigHash(context.Background(), contractRawAddress,
					instrument.InstrumentCode, receiverAddress, *blockHash, receiver.OracleSigExpiry, 1)
				if err != nil {
					fmt.Printf("Failed to generate sig hash : %s\n", err)
					return nil
				}
				fmt.Printf("Hash : %s\n", *hash)

				signature, err := key.Sign(*hash)
				if err != nil {
					fmt.Printf("Failed to sign sig hash : %s\n", err)
					return nil
				}

				fmt.Printf("Signature : %s\n", signature)
				return nil
			}

			index++
		}
	}

	fmt.Printf("Failed to find receiver index")
	return nil
}

func init() {
}
