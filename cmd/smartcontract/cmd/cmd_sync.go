package cmd

import (
	"strings"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/rebuilder"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
)

const (
	FlagDebugMode = "debug"
)

var cmdSync = &cobra.Command{
	Use:   "sync",
	Short: "Syncronize contract state with the network",
	RunE: func(c *cobra.Command, args []string) error {
		debugMode, _ := c.Flags().GetBool(FlagDebugMode)

		// -------------------------------------------------------------------------
		// Logging

		ctx, log := logger.NewLoggerWithContext()

		if debugMode {
			log.Infof("Starting sync...")
		}

		// -------------------------------------------------------------------------
		// Configuration

		var cfg struct {
			Contract struct {
				PrivateKey   string `envconfig:"PRIV_KEY"`
				OperatorName string `envconfig:"OPERATOR_NAME"`
				Version      string `envconfig:"VERSION"`
				FeeAddress   string `envconfig:"FEE_ADDRESS"`
				FeeAmount    string `envconfig:"FEE_VALUE"` // TODO rename FEE_AMOUNT
			}
			SpvNode struct {
				Address   string `envconfig:"NODE_ADDRESS"`
				UserAgent string `envconfig:"NODE_USER_AGENT"`
			}
			RpcNode struct {
				Host     string `envconfig:"RPC_HOST"`
				Username string `envconfig:"RPC_USERNAME"`
				Password string `envconfig:"RPC_PASSWORD"`
			}
			NodeStorage struct {
				Region    string `default:"ap-southeast-2" envconfig:"NODE_STORAGE_REGION"`
				AccessKey string `envconfig:"NODE_STORAGE_ACCESS_KEY"`
				Secret    string `envconfig:"NODE_STORAGE_SECRET"`
				Bucket    string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
				Root      string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
			}
			Storage struct {
				Region    string `default:"ap-southeast-2" envconfig:"CONTRACT_STORAGE_REGION"`
				AccessKey string `envconfig:"CONTRACT_STORAGE_ACCESS_KEY"`
				Secret    string `envconfig:"CONTRACT_STORAGE_SECRET"`
				Bucket    string `default:"standalone" envconfig:"CONTRACT_STORAGE_BUCKET"`
				Root      string `default:"./tmp" envconfig:"CONTRACT_STORAGE_ROOT"`
			}
		}

		if err := envconfig.Process("API", &cfg); err != nil {
			log.Fatalf("main : Parsing Config : %v", err)
		}

		// -------------------------------------------------------------------------
		// Trusted Peer Node

		spvStorageConfig := storage.NewConfig(cfg.NodeStorage.Region,
			cfg.NodeStorage.AccessKey,
			cfg.NodeStorage.Secret,
			cfg.NodeStorage.Bucket,
			cfg.NodeStorage.Root)

		var spvStorage storage.Storage
		if strings.ToLower(spvStorageConfig.Bucket) == "standalone" {
			spvStorage = storage.NewFilesystemStorage(spvStorageConfig)
		} else {
			spvStorage = storage.NewS3Storage(spvStorageConfig)
		}

		spvConfig := spvnode.NewConfig(cfg.SpvNode.Address, cfg.SpvNode.UserAgent)

		spvNode := spvnode.NewNode(spvConfig, spvStorage)

		// -------------------------------------------------------------------------
		// Network

		rpcConfig := rpcnode.NewConfig(cfg.RpcNode.Host,
			cfg.RpcNode.Username,
			cfg.RpcNode.Password)

		network, err := network.NewNetwork(rpcConfig, spvNode)
		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Wallet

		wallet, err := wallet.NewWallet(cfg.Contract.PrivateKey)
		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Contract Storage

		contractStorageConfig := storage.NewConfig(cfg.Storage.Region,
			cfg.Storage.AccessKey,
			cfg.Storage.Secret,
			cfg.Storage.Bucket,
			cfg.Storage.Root)

		var contractStorage storage.Storage
		if strings.ToLower(contractStorageConfig.Bucket) == "standalone" {
			contractStorage = storage.NewFilesystemStorage(contractStorageConfig)
		} else {
			contractStorage = storage.NewS3Storage(contractStorageConfig)
		}

		// -------------------------------------------------------------------------
		// App Config

		appConfig, err := config.NewConfig(cfg.Contract.OperatorName,
			cfg.Contract.Version,
			cfg.Contract.FeeAddress,
			cfg.Contract.FeeAmount)

		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Builder

		state := state.NewStateService(contractStorage)
		inspector := inspector.NewInspectorService(network)
		request := request.NewRequestService(*appConfig, wallet, state, inspector)
		response := response.NewResponseService(*appConfig, wallet, state, inspector)
		validator := validator.NewValidatorService(*appConfig, wallet, state)

		// -------------------------------------------------------------------------
		// Rebuilder

		reb := rebuilder.NewRebuilderService(network, inspector, request, response, validator, state)

		// -------------------------------------------------------------------------
		// Contract address

		contractAddr, err := btcutil.DecodeAddress(string(wallet.PublicAddress), &chaincfg.MainNetParams)
		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Find or create state

		hard, soft, err := reb.FindState(ctx, contractAddr)
		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Sync

		reb.Sync(ctx, soft, hard, contractAddr)

		return nil
	},
}

func init() {
	cmdSync.Flags().Bool(FlagDebugMode, false, "Debug mode")
}
