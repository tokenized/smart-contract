package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tokenized/config"
	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type BuildConfig struct {
	Net         bitcoin.Network `default:"mainnet" envconfig:"NETWORK"`
	IsTest      bool            `default:"true" envconfig:"IS_TEST"`
	FeeRate     float32         `default:"0.05" envconfig:"CLIENT_FEE_RATE"`
	DustFeeRate float32         `default:"0.0" envconfig:"CLIENT_DUST_FEE_RATE"`
	ContractFee uint64          `default:"1000" envconfig:"CLIENT_CONTRACT_FEE"`
}

const (
	FlagTx        = "tx"
	FlagHexFormat = "hex"
	FlagSend      = "send"
)

var cmdBuild = &cobra.Command{
	Use:   "build <contractAddress> <typeCode> <jsonFile> <fundingKey> <fundingOutpoint> <fundingValue>",
	Short: "Build an action/instrument/message payload from a json file.",
	Long:  "Build and action/instrument/message payload from a json file. Note: fixedbin (fixed size binary) in json is an array of 8 bit integers and bin (variable size binary) is hex encoded binary data.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 6 {
			return errors.New("Missing json file parameter")
		}

		switch len(args[1]) {
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
	ctx := context.Background()
	if ctx == nil {
		return nil
	}

	cfg := &BuildConfig{}
	if err := config.LoadConfig(ctx, cfg); err != nil {
		logger.Fatal(ctx, "Failed to load config : %s", err)
	}

	maskedConfig, err := config.MarshalJSONMaskedRaw(cfg)
	if err != nil {
		logger.Fatal(ctx, "Failed to marshal masked config : %s", err)
	}

	logger.InfoWithFields(ctx, []logger.Field{
		logger.JSON("config", maskedConfig),
	}, "Config")

	if len(args) != 6 {
		logger.Fatal(ctx, "Wrong arguments : build <contractAddress> <typeCode> <jsonFile> <fundingKey> <fundingOutpoint> <fundingValue>")
	}

	contractAddress, err := bitcoin.DecodeAddress(args[0])
	if err != nil {
		logger.Fatal(ctx, "Invalid address : %s", err)
	}

	contractLockingScript, err := bitcoin.NewRawAddressFromAddress(contractAddress).LockingScript()
	if err != nil {
		logger.Fatal(ctx, "Failed to create locking script : %s", err)
	}

	actionType := strings.ToUpper(args[1])
	filePath := filepath.FromSlash(args[2])

	key, err := bitcoin.KeyFromStr(args[3])
	if err != nil {
		logger.Fatal(ctx, "Invalid key : %s", err)
	}

	fundingLockingScript, err := key.LockingScript()
	if err != nil {
		logger.Fatal(ctx, "Failed to create locking script : %s", err)
	}

	parts := strings.Split(args[4], ":")
	if len(parts) != 2 {
		logger.Fatal(ctx, "Invalid funding outpoint : %d parts", len(parts))
	}

	fundingHash, err := bitcoin.NewHash32FromStr(parts[0])
	if err != nil {
		logger.Fatal(ctx, "Invalid funding outpoint hash : %s", err)
	}

	fundingIndex, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		logger.Fatal(ctx, "Invalid funding outpoint index : %s", err)
	}

	fundingValue, err := strconv.ParseUint(args[5], 10, 64)
	if err != nil {
		logger.Fatal(ctx, "Invalid funding value : %s", err)
	}

	// Create struct
	action := actions.NewActionFromCode(actionType)
	if action == nil {
		fmt.Printf("Unsupported action type : %s\n", actionType)
		return nil
	}

	// Read json file
	data, err := ioutil.ReadFile(filePath)
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

	actionScript, err := protocol.Serialize(action, cfg.IsTest)
	if err != nil {
		fmt.Printf("Failed to serialize %s op return : %s\n", actionType, err)
		return nil
	}

	hexFormat, _ := c.Flags().GetBool(FlagHexFormat)
	buildTx, _ := c.Flags().GetBool(FlagTx)
	var tx *txbuilder.TxBuilder
	if buildTx {
		tx = txbuilder.NewTxBuilder(cfg.FeeRate, cfg.DustFeeRate)
		tx.SetChangeLockingScript(fundingLockingScript, "")

		// Add output to contract
		contractOutputIndex := uint32(0)
		err = tx.AddOutput(contractLockingScript, 1025, false, false)
		if err != nil {
			fmt.Printf("Failed to add contract output : %s\n", err)
			return nil
		}

		// Add op return
		err = tx.AddOutput(actionScript, 0, false, false)
		if err != nil {
			fmt.Printf("Failed to add op return output : %s\n", err)
			return nil
		}

		// Determine funding required for contract to be able to post response tx.
		dustLimit := txbuilder.DustLimit(txbuilder.P2PKHOutputSize, cfg.DustFeeRate)
		estimatedSize, funding, err := protocol.EstimatedResponse(tx.MsgTx, 0,
			dustLimit, cfg.ContractFee, cfg.IsTest)
		if err != nil {
			fmt.Printf("Failed to estimate funding : %s\n", err)
			return nil
		}
		fmt.Printf("Response estimated : %d bytes, %d funding\n", estimatedSize, funding)
		funding += uint64(float32(estimatedSize) * cfg.FeeRate * 1.1) // Add response tx fee
		err = tx.AddValueToOutput(contractOutputIndex, funding)
		if err != nil {
			fmt.Printf("Failed to add estimated funding to contract output of tx : %s\n", err)
			return nil
		}

		if err := tx.AddInputUTXO(bitcoin.UTXO{
			Hash:          *fundingHash,
			Index:         uint32(fundingIndex),
			Value:         fundingValue,
			LockingScript: fundingLockingScript,
		}); err != nil {
			fmt.Printf("Failed to add input : %s\n", err)
			return nil
		}

		if _, err := tx.Sign([]bitcoin.Key{key}); err != nil {
			fmt.Printf("Failed to sign tx : %s\n", err)
			return nil
		}

		if hexFormat {
			fmt.Printf("TxID (%d bytes) : %s\n", tx.MsgTx.SerializeSize(), tx.MsgTx.TxHash())
			var buf bytes.Buffer
			if err := tx.MsgTx.Serialize(&buf); err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to serialize %s tx", actionType))
			}
			fmt.Printf("%x\n", buf.Bytes())
		} else {
			fmt.Println(tx.MsgTx.StringWithAddresses(network(c)))
		}
	}

	fmt.Printf("Action : %s\n", actionType)
	if hexFormat {
		fmt.Printf("%x\n", actionScript)
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
}
