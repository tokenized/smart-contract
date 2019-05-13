package cmd

import (
	"bytes"
	"encoding/base64"
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
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/specification/dist/golang/protocol"
)

const (
	FlagTx           = "tx"
	FlagHexFormat    = "hex"
	FlagBase64Format = "b64"
	FlagSend         = "send"
)

var cmdBuild = &cobra.Command{
	Use:   "build <typeCode> <jsonFile>",
	Short: "Build an action/asset/message payload from a json file.",
	Long:  "Build and action/asset/message payload from a json file. Note: fixedbin (fixed size binary) in json is an array of 8 bit integers and bin (variable size binary) is base64 encoded binary data.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Missing json file parameter")
		}

		switch len(args[0]) {
		case 2:
			return buildAction(c, args)
		case 3:
			return buildAssetPayload(c, args)
		case 4:
			return buildMessage(c, args)
		default:
			return fmt.Errorf("Unknown type code length %d\n  Actions are 2 characters\n  Assets are 3 characters\n  Messages are 4 characters", len(args[0]))
		}
	},
}

func buildAction(c *cobra.Command, args []string) error {
	actionType := strings.ToUpper(args[0])

	// Create struct
	opReturn := protocol.TypeMapping(actionType)
	if opReturn == nil {
		fmt.Printf("Unsupported action type : %s\n", actionType)
		return nil
	}

	// Read json file
	path := filepath.FromSlash(args[1])
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read json file : %s\n", err)
		return nil
	}

	// Put json data into opReturn struct
	if err := json.Unmarshal(data, opReturn); err != nil {
		fmt.Printf("Failed to unmarshal %s json file : %s\n", actionType, err)
		return nil
	}

	script, err := protocol.Serialize(opReturn, true)
	if err != nil {
		fmt.Printf("Failed to serialize %s op return : %s\n", actionType, err)
		return nil
	}

	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	b64Format, _ := c.Flags().GetBool(FlagBase64Format)
	txFormat, _ := c.Flags().GetBool(FlagTx)
	if txFormat {
		ctx := client.Context()
		if ctx == nil {
			return nil
		}

		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			fmt.Printf("Failed to create client : %s\n", err)
			return nil
		}

		tx := txbuilder.NewTx(theClient.Wallet.PublicKeyHash, theClient.Config.DustLimit, theClient.Config.FeeRate)

		// Add output to contract
		contractOutputIndex := uint32(0)
		err = tx.AddP2PKHDustOutput(theClient.ContractPKH, false)
		if err != nil {
			fmt.Printf("Failed to add contract output : %s\n", err)
			return nil
		}

		// Add op return
		err = tx.AddOutput(script, 0, false, false)
		if err != nil {
			fmt.Printf("Failed to add op return output : %s\n", err)
			return nil
		}

		// Determine funding required for contract to be able to post response tx.
		estimatedSize, funding, err := protocol.EstimatedResponse(tx.MsgTx, 0, theClient.Config.DustLimit, theClient.Config.ContractFee, true)
		if err != nil {
			fmt.Printf("Failed to estimate funding : %s\n", err)
			return nil
		}
		fmt.Printf("Response estimated : %d bytes, %d funding\n", estimatedSize, funding)
		funding += uint64(float32(estimatedSize)*theClient.Config.FeeRate*1.1) + 1500 // Add response tx fee
		err = tx.AddValueToOutput(contractOutputIndex, funding)
		if err != nil {
			fmt.Printf("Failed to add estimated funding to contract output of tx : %s\n", err)
			return nil
		}

		// Add inputs
		var emptyHash chainhash.Hash
		fee := tx.EstimatedFee()
		inputValue := uint64(0)
		for _, output := range theClient.Wallet.UnspentOutputs() {
			if fee+tx.OutputValue(false)+funding < inputValue {
				break
			}
			if output.SpentByTxId != emptyHash {
				continue
			}
			err := tx.AddInput(output.OutPoint, output.PkScript, output.Value)
			if err != nil {
				fmt.Printf("Failed to add input : %s\n", err)
				return nil
			}
			inputValue += output.Value
			fee = tx.EstimatedFee()
		}
		if fee > inputValue {
			fmt.Printf("Insufficient balance for tx fee %.08f : balance %.08f\n",
				client.BitcoinsFromSatoshis(fee), client.BitcoinsFromSatoshis(inputValue))
			return nil
		}

		err = tx.Sign([]*btcec.PrivateKey{theClient.Wallet.Key})
		if err != nil {
			fmt.Printf("Failed to sign tx : %s\n", err)
			return nil
		}

		// Check with inspector
		var itx *inspector.Transaction
		itx, err = inspector.NewTransactionFromWire(ctx, tx.MsgTx, true)
		if err != nil {
			logger.Warn(ctx, "Failed to convert tx to inspector")
		}

		if !itx.IsTokenized() {
			logger.Warn(ctx, "Tx is not inspector tokenized")
		}

		fmt.Printf("Tx Id (%d bytes) : %s\n", tx.MsgTx.SerializeSize(), tx.MsgTx.TxHash())

		if hexFormat {
			var buf bytes.Buffer
			err := tx.MsgTx.Serialize(&buf)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s tx", actionType))
			}
			fmt.Printf("%x\n", buf.Bytes())
		} else if b64Format {
			var buf bytes.Buffer
			err := tx.MsgTx.Serialize(&buf)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s tx", actionType))
			}
			fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(buf.Bytes()))
		} else {
			data, err = json.MarshalIndent(tx.MsgTx, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s tx", actionType))
			}
			fmt.Printf(string(data) + "\n")
		}

		send, _ := c.Flags().GetBool(FlagSend)
		if send {
			if err := theClient.ShotgunTx(ctx, tx.MsgTx, 250); err != nil {
				fmt.Printf("Failed to send tx : %s\n", err)
			}
		}
	}

	fmt.Printf("Action : %s\n", actionType)
	if hexFormat {
		fmt.Printf("%x\n", script)
	} else if b64Format {
		fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(script))
	} else {
		data, err = json.MarshalIndent(opReturn, "", "  ")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s", actionType))
		}
		fmt.Printf(string(data) + "\n")
	}

	switch actionType {
	case "A1":
		assetDef, ok := opReturn.(*protocol.AssetDefinition)
		if !ok {
			fmt.Printf("Failed to convert to asset definition")
			return nil
		}

		if err := assetDef.Validate(); err != nil {
			fmt.Printf("Invalid asset definition : %s\n", err)
			return nil
		}

		payload := protocol.AssetTypeMapping(assetDef.AssetType)
		if payload == nil {
			fmt.Printf("Invalid asset type : %s\n", assetDef.AssetType)
			return nil
		}

		_, err := payload.Write(assetDef.AssetPayload)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", assetDef.AssetType))
		}

		fmt.Printf("Payload : %s\n", assetDef.AssetType)
		if hexFormat {
			fmt.Printf("%x\n", payload)
		} else if b64Format {
			fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(assetDef.AssetPayload))
		} else {
			data, err = json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal asset payload %s", assetDef.AssetType))
			}
			fmt.Printf(string(data) + "\n")
		}

	case "A2":
		assetCreation, ok := opReturn.(*protocol.AssetCreation)
		if !ok {
			fmt.Printf("Failed to convert to asset creation")
			return nil
		}

		if err := assetCreation.Validate(); err != nil {
			fmt.Printf("Invalid asset creation : %s\n", err)
			return nil
		}

		payload := protocol.AssetTypeMapping(assetCreation.AssetType)
		if payload == nil {
			fmt.Printf("Invalid asset type : %s\n", assetCreation.AssetType)
			return nil
		}

		_, err := payload.Write(assetCreation.AssetPayload)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", assetCreation.AssetType))
		}

		fmt.Printf("Payload : %s\n", assetCreation.AssetType)
		if hexFormat {
			fmt.Printf("%x\n", payload)
		} else if b64Format {
			fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(assetCreation.AssetPayload))
		} else {
			data, err = json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal asset payload %s", assetCreation.AssetType))
			}
			fmt.Printf(string(data) + "\n")
		}
	}

	return nil
}

func buildAssetPayload(c *cobra.Command, args []string) error {
	assetType := strings.ToUpper(args[0])

	// Create struct
	payload := protocol.AssetTypeMapping(assetType)
	if payload == nil {
		return fmt.Errorf("Unsupported asset type : %s", assetType)
	}

	// Read json file
	path := filepath.FromSlash(args[1])
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "Failed to read json file")
	}

	// Put json data into payload struct
	if err := json.Unmarshal(jsonData, payload); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to unmarshal %s json file", assetType))
	}

	data, err := payload.Serialize()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s asset payload", assetType))
	}

	fmt.Printf("Asset : %s\n", assetType)
	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	b64Format, _ := c.Flags().GetBool(FlagBase64Format)
	if hexFormat {
		fmt.Printf("%x\n", data)
	} else if b64Format {
		fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(data))
	} else {
		jsonData, err = json.MarshalIndent(&payload, "", "  ")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s", assetType))
		}
		fmt.Printf(string(jsonData) + "\n")
	}

	return nil
}

func buildMessage(c *cobra.Command, args []string) error {
	return errors.New("Message building not implemented")
}

func init() {
	cmdBuild.Flags().Bool(FlagTx, false, "build a tx, if false only op return is built")
	cmdBuild.Flags().Bool(FlagHexFormat, false, "hex format")
	cmdBuild.Flags().Bool(FlagBase64Format, false, "base64 format")
	cmdBuild.Flags().Bool(FlagSend, false, "send to network")
}
