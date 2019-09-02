package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tokenized/smart-contract/cmd/smartcontract/client"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/json"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	FlagTx           = "tx"
	FlagHexFormat    = "hex"
	FlagSend         = "send"
)

var cmdBuild = &cobra.Command{
	Use:   "build <typeCode> <jsonFile>",
	Short: "Build an action/asset/message payload from a json file.",
	Long:  "Build and action/asset/message payload from a json file. Note: fixedbin (fixed size binary) in json is an array of 8 bit integers and bin (variable size binary) is hex encoded binary data.",
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
	action := actions.NewActionFromCode(actionType)
	if action == nil {
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
	if err := json.Unmarshal(data, action); err != nil {
		fmt.Printf("Failed to unmarshal %s json file : %s\n", actionType, err)
		return nil
	}

	// validate the message
	if err := action.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Message : %+v\n", action)
		return nil
	}

	// Validate smart contract rules
	switch m := action.(type) {
	case *actions.ContractOffer:
		fmt.Printf("Checking Contract Offer\n")
		_, err := protocol.ReadAuthFlags(m.ContractAuthFlags, actions.ContractFieldCount, len(m.VotingSystems))
		if err != nil {
			fmt.Printf("Setting default auth flags\n")
			permissions := make([]protocol.Permission, actions.ContractFieldCount)
			votingSystems := make([]bool, len(m.VotingSystems))
			for i, _ := range votingSystems {
				votingSystems[i] = true
			}
			permission := protocol.Permission{
				Permitted:              true,
				AdministrationProposal: true,
				HolderProposal:         true,
				VotingSystemsAllowed:   votingSystems,
			}
			for i, _ := range permissions {
				permissions[i] = permission
				fmt.Printf("Field %d : %+v\n", i, permissions[i])
			}

			m.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return nil
			}
		}

	case *actions.AssetDefinition:
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("How many voting systems are in the contract: ")
		votingSystemCountString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read user input : %s\n", err)
			return nil
		}
		votingSystemCount, err := strconv.Atoi(strings.TrimSpace(votingSystemCountString))
		if err != nil {
			fmt.Printf("User input is not an integer : %s\n", err)
			return nil
		}
		fmt.Printf("Checking Asset Definition\n")
		_, err = protocol.ReadAuthFlags(m.AssetAuthFlags, actions.AssetFieldCount, votingSystemCount)
		if err != nil {
			fmt.Printf("Setting default auth flags\n")
			permissions := make([]protocol.Permission, actions.AssetFieldCount)
			votingSystems := make([]bool, votingSystemCount)
			for i, _ := range votingSystems {
				votingSystems[i] = true
			}
			permission := protocol.Permission{
				Permitted:              true,
				AdministrationProposal: true,
				HolderProposal:         true,
				VotingSystemsAllowed:   votingSystems,
			}
			for i, _ := range permissions {
				permissions[i] = permission
				fmt.Printf("Field %d : %+v\n", i, permissions[i])
			}
			m.AssetAuthFlags, err = protocol.WriteAuthFlags(permissions)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return nil
			}
		}

	}

	script, err := protocol.Serialize(action, true)
	if err != nil {
		fmt.Printf("Failed to serialize %s op return : %s\n", actionType, err)
		return nil
	}

	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	buildTx, _ := c.Flags().GetBool(FlagTx)
	if buildTx {
		ctx := client.Context()
		if ctx == nil {
			return nil
		}

		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			fmt.Printf("Failed to create client : %s\n", err)
			return nil
		}

		tx := txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit, theClient.Config.FeeRate)

		// Add output to contract
		contractOutputIndex := uint32(0)
		err = tx.AddDustOutput(theClient.ContractAddress, false)
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
		estimatedSize, funding, err := protocol.EstimatedResponse(tx.MsgTx, 0,
			theClient.Config.DustLimit, theClient.Config.ContractFee, true)
		if err != nil {
			fmt.Printf("Failed to estimate funding : %s\n", err)
			return nil
		}
		fmt.Printf("Response estimated : %d bytes, %d funding\n", estimatedSize, funding)
		funding += uint64(float32(estimatedSize)*theClient.Config.FeeRate*1.1) + 2500 // Add response tx fee
		err = tx.AddValueToOutput(contractOutputIndex, funding)
		if err != nil {
			fmt.Printf("Failed to add estimated funding to contract output of tx : %s\n", err)
			return nil
		}

		// Add inputs
		var emptyHash bitcoin.Hash32
		fee := tx.EstimatedFee()
		inputValue := uint64(0)
		for _, output := range theClient.Wallet.UnspentOutputs() {
			if fee+tx.OutputValue(false)+funding < inputValue {
				break
			}
			if !emptyHash.Equal(output.SpentByTxId) {
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

		err = tx.Sign([]bitcoin.Key{theClient.Wallet.Key})
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

		if hexFormat {
			fmt.Printf("Tx Id (%d bytes) : %s\n", tx.MsgTx.SerializeSize(), tx.MsgTx.TxHash())
			var buf bytes.Buffer
			err := tx.MsgTx.Serialize(&buf)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s tx", actionType))
			}
			fmt.Printf("%x\n", buf.Bytes())
		} else {
			fmt.Println(tx.MsgTx.String())
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
	} else {
		data, err = json.MarshalIndent(action, "", "  ")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s", actionType))
		}
		fmt.Printf(string(data) + "\n")
	}

	switch actionType {
	case "A1":
		assetDef, ok := action.(*actions.AssetDefinition)
		if !ok {
			fmt.Printf("Failed to convert to asset definition")
			return nil
		}

		if err := assetDef.Validate(); err != nil {
			fmt.Printf("Invalid asset definition : %s\n", err)
			return nil
		}

		asset, err := assets.Deserialize([]byte(assetDef.AssetType), assetDef.AssetPayload)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", assetDef.AssetType))
		}

		fmt.Printf("Payload : %s\n", assetDef.AssetType)
		if hexFormat {
			fmt.Printf("%x\n", assetDef.AssetPayload)
		} else {
			data, err = json.MarshalIndent(asset, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal asset payload %s", assetDef.AssetType))
			}
			fmt.Printf(string(data) + "\n")
		}

	case "A2":
		assetCreation, ok := action.(*actions.AssetCreation)
		if !ok {
			fmt.Printf("Failed to convert to asset creation")
			return nil
		}

		if err := assetCreation.Validate(); err != nil {
			fmt.Printf("Invalid asset creation : %s\n", err)
			return nil
		}

		payload := assets.NewAssetFromCode(assetCreation.AssetType)
		if payload == nil {
			fmt.Printf("Invalid asset type : %s\n", assetCreation.AssetType)
			return nil
		}

		assetCreation.AssetPayload, err = payload.Bytes()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", assetCreation.AssetType))
		}

		fmt.Printf("Payload : %s\n", assetCreation.AssetType)
		if hexFormat {
			fmt.Printf("%x\n", payload)
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
	payload := assets.NewAssetFromCode(assetType)
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

	data, err := payload.Bytes()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s asset payload", assetType))
	}

	fmt.Printf("Asset : %s\n", assetType)
	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	if hexFormat {
		fmt.Printf("%x\n", data)
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
	cmdBuild.Flags().Bool(FlagSend, false, "send to network")
}
