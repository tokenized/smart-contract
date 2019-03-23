package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/cmd/smartcontract/client"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

const (
	FlagTx        = "tx"
	FlagHexFormat = "hex"
)

var cmdBuild = &cobra.Command{
	Use:   "build actionType jsonFile",
	Short: "build an action from a json file",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Missing json file parameter")
		}

		// Read json file
		path := filepath.FromSlash(args[1])
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "Failed to read json file")
		}

		opReturn := protocol.TypeMapping(strings.ToUpper(args[0]))
		if opReturn == nil {
			return fmt.Errorf("Unsupported action type : %s", args[0])
		}

		if err := json.Unmarshal(data, opReturn); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to unmarshal %s json file", args[0]))
		}

		ctx := client.Context()
		theClient, err := client.NewClient(ctx)
		if err != nil {
			return err
		}

		txFormat, _ := c.Flags().GetBool(FlagTx)
		if txFormat {
			tx := txbuilder.NewTx(theClient.Wallet.PublicKeyHash, theClient.Config.DustLimit, theClient.Config.FeeRate)

			// TODO Custom inputs/outputs based on opReturn type

			// Add output to contract
			err = tx.AddP2PKHDustOutput(theClient.ContractPKH, false)
			if err != nil {
				return errors.Wrap(err, "Failed to add contract output")
			}

			// Add output to issuer (mark as change because it is wallet)
			err = tx.AddP2PKHDustOutput(theClient.Wallet.PublicKeyHash, true)
			if err != nil {
				return errors.Wrap(err, "Failed to add issuer output")
			}

			// Add op return
			script, err := protocol.Serialize(opReturn)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s op return", args[0]))
			}
			err = tx.AddOutput(script, 0, false, false)
			if err != nil {
				return errors.Wrap(err, "Failed to add op return output")
			}

			// Add inputs
			var emptyHash chainhash.Hash
			fee := tx.EstimatedFee()
			inputValue := uint64(0)
			for _, output := range theClient.Wallet.UnspentOutputs() {
				if output.SpentByTxId != emptyHash {
					continue
				}
				err := tx.AddInput(output.OutPoint, output.PkScript, output.Value)
				if err != nil {
					return errors.Wrap(err, "Failed to add input")
				}
				inputValue += output.Value
				fee = tx.EstimatedFee()
				if fee < inputValue {
					break
				}
			}
			if fee > inputValue {
				return fmt.Errorf("Insufficient balance for tx fee %.08f : balance %.08f",
					client.BitcoinsFromSatoshis(fee), client.BitcoinsFromSatoshis(inputValue))
			}

			// Fund contract to be able to post contract formation
			estimatedSize, funding, err := protocol.EstimatedResponse(tx.MsgTx, theClient.Config.DustLimit)
			funding = uint64(float32(funding) * 1.1) // Add 10% buffer
			funding += uint64(float32(estimatedSize) * theClient.Config.FeeRate) // Add response tx fee
			err = tx.AddValueToOutput(0, funding)
			if err != nil {
				return errors.Wrap(err, "Failed to add estimated funding to contract output of tx")
			}

			err = tx.Sign([]*btcec.PrivateKey{theClient.Wallet.Key})
			if err != nil {
				return errors.Wrap(err, "Failed to sign tx")
			}

			// Check with inspector
			var itx *inspector.Transaction
			itx, err = inspector.NewTransactionFromWire(ctx, tx.MsgTx)
			if err != nil {
				logger.Warn(ctx, "Failed to convert tx to inspector")
			}

			if !itx.IsTokenized() {
				logger.Warn(ctx, "Tx is not inspector tokenized")
			}

			fmt.Printf("Tx Id : %s\n", tx.MsgTx.TxHash())

			hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
			if hexFormat {
				var buf bytes.Buffer
				err := tx.MsgTx.Serialize(&buf)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s tx", args[0]))
				}
				fmt.Printf("%x", buf.Bytes())
			} else {
				data, err = json.MarshalIndent(tx.MsgTx, "", "  ")
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s tx", args[0]))
				}
				fmt.Printf(string(data))
			}
		} else {
			hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
			if hexFormat {
				script, err := protocol.Serialize(opReturn)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s op return", args[0]))
				}
				fmt.Printf("%x", script)
			} else {
				data, err = json.MarshalIndent(opReturn, "", "  ")
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s", args[0]))
				}
				fmt.Printf(string(data))
			}
		}

		return nil
	},
}

func init() {
	cmdBuild.Flags().Bool(FlagTx, false, "build tx, if false only op return is built")
	cmdBuild.Flags().Bool(FlagHexFormat, false, "hex format")
}
