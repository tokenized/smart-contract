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

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/smart-contract/cmd/smartcontract/client"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	FlagTx        = "tx"
	FlagHexFormat = "hex"
	FlagSend      = "send"
)

var cmdBuild = &cobra.Command{
	Use:   "build <typeCode> <jsonFile>",
	Short: "Build an action/instrument/message payload from a json file.",
	Long:  "Build and action/instrument/message payload from a json file. Note: fixedbin (fixed size binary) in json is an array of 8 bit integers and bin (variable size binary) is hex encoded binary data.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Missing json file parameter")
		}

		switch len(args[0]) {
		case 2:
			return buildAction(c, args)
		case 3:
			return buildInstrumentPayload(c, args)
		case 4:
			return buildMessage(c, args)
		default:
			return fmt.Errorf("Unknown type code length %d\n  Actions are 2 characters\n  Instruments are 3 characters\n  Messages are 4 characters", len(args[0]))
		}
	},
}

func buildAction(c *cobra.Command, args []string) error {
	ctx := client.Context()
	if ctx == nil {
		return nil
	}

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
		_, err := permissions.PermissionsFromBytes(m.ContractPermissions, len(m.VotingSystems))
		if err != nil {
			fmt.Printf("Invalid permissions\n")
		}

	case *actions.InstrumentDefinition:
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
		fmt.Printf("Checking Instrument Definition\n")
		_, err = permissions.PermissionsFromBytes(m.InstrumentPermissions, votingSystemCount)
		if err != nil {
			fmt.Printf("Invalid permissions\n")
		}

	}

	theClient, err := client.NewClient(ctx, network(c))
	if err != nil {
		fmt.Printf("Failed to create client : %s\n", err)
		return nil
	}

	script, err := protocol.Serialize(action, theClient.Config.IsTest)
	if err != nil {
		fmt.Printf("Failed to serialize %s op return : %s\n", actionType, err)
		return nil
	}

	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	buildTx, _ := c.Flags().GetBool(FlagTx)
	var tx *txbuilder.TxBuilder
	if buildTx {

		tx = txbuilder.NewTxBuilder(theClient.Config.FeeRate, theClient.Config.DustFeeRate)
		tx.SetChangeAddress(theClient.Wallet.Address, "")

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
		dustLimit := txbuilder.DustLimit(txbuilder.P2PKHOutputSize, theClient.Config.DustFeeRate)
		estimatedSize, funding, err := protocol.EstimatedResponse(tx.MsgTx, 0,
			dustLimit, theClient.Config.ContractFee, theClient.Config.IsTest)
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
			err := tx.AddInput(output.OutPoint, output.LockingScript, output.Value)
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
		itx, err = inspector.NewTransactionFromWire(ctx, tx.MsgTx, theClient.Config.IsTest)
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
			fmt.Println(tx.MsgTx.StringWithAddresses(network(c)))
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
		instrumentDef, ok := action.(*actions.InstrumentDefinition)
		if !ok {
			fmt.Printf("Failed to convert to instrument definition")
			return nil
		}

		if err := instrumentDef.Validate(); err != nil {
			fmt.Printf("Invalid instrument definition : %s\n", err)
			return nil
		}

		instrument, err := instruments.Deserialize([]byte(instrumentDef.InstrumentType), instrumentDef.InstrumentPayload)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", instrumentDef.InstrumentType))
		}

		fmt.Printf("Payload : %s\n", instrumentDef.InstrumentType)
		if hexFormat {
			fmt.Printf("%x\n", instrumentDef.InstrumentPayload)
		} else {
			data, err = json.MarshalIndent(instrument, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal instrument payload %s", instrumentDef.InstrumentType))
			}
			fmt.Printf(string(data) + "\n")
		}

	case "A2":
		instrumentCreation, ok := action.(*actions.InstrumentCreation)
		if !ok {
			fmt.Printf("Failed to convert to instrument creation")
			return nil
		}

		if err := instrumentCreation.Validate(); err != nil {
			fmt.Printf("Invalid instrument creation : %s\n", err)
			return nil
		}

		payload := instruments.NewInstrumentFromCode(instrumentCreation.InstrumentType)
		if payload == nil {
			fmt.Printf("Invalid instrument type : %s\n", instrumentCreation.InstrumentType)
			return nil
		}

		instrumentCreation.InstrumentPayload, err = payload.Bytes()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to deserialize %s payload", instrumentCreation.InstrumentType))
		}

		fmt.Printf("Payload : %s\n", instrumentCreation.InstrumentType)
		if hexFormat {
			fmt.Printf("%x\n", payload)
		} else {
			data, err = json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to marshal instrument payload %s", instrumentCreation.InstrumentType))
			}
			fmt.Printf(string(data) + "\n")
		}

	case "A3":
		instrumentModification, ok := action.(*actions.InstrumentModification)
		if !ok {
			fmt.Printf("Failed to convert to instrument modification")
			return nil
		}

		if err := instrumentModification.Validate(); err != nil {
			fmt.Printf("Invalid instrument modification : %s\n", err)
			return nil
		}

		for i, mod := range instrumentModification.Amendments {
			fip, err := permissions.FieldIndexPathFromBytes(mod.FieldIndexPath)
			if err != nil {
				fmt.Printf("Invalid field index path : %s\n", err)
				return nil
			}
			fmt.Printf("Field index path %d : %v\n", i, fip)
		}
	}

	if buildTx {
		send, _ := c.Flags().GetBool(FlagSend)
		if send {
			fmt.Printf("Sending to network\n")
			if err := theClient.BroadcastTx(ctx, tx.MsgTx); err != nil {
				fmt.Printf("Failed to send tx : %s\n", err)
			}
		}
	}

	return nil
}

func buildInstrumentPayload(c *cobra.Command, args []string) error {
	instrumentType := strings.ToUpper(args[0])

	// Create struct
	payload := instruments.NewInstrumentFromCode(instrumentType)
	if payload == nil {
		return fmt.Errorf("Unsupported instrument type : %s", instrumentType)
	}

	// Read json file
	path := filepath.FromSlash(args[1])
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "Failed to read json file")
	}

	// Put json data into payload struct
	if err := json.Unmarshal(jsonData, payload); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to unmarshal %s json file", instrumentType))
	}

	data, err := payload.Bytes()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s instrument payload", instrumentType))
	}

	fmt.Printf("Instrument : %s\n", instrumentType)
	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	if hexFormat {
		fmt.Printf("%x\n", data)
	} else {
		jsonData, err = json.MarshalIndent(&payload, "", "  ")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to marshal %s", instrumentType))
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
