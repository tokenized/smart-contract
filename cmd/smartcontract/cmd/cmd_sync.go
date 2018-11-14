package cmd

import (
	"os"
	"strings"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/internal/app/rpcnode"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/internal/broadcaster"
	"github.com/tokenized/smart-contract/internal/rebuilder"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"

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

		// Logger
		ctx, log := logger.NewLoggerWithContext()

		if debugMode {
			log.Infof("Starting sync...")
		}

		// Configuration
		config, err := config.NewConfig()
		if err != nil {
			panic(err)
		}

		// Trusted Peer Node
		spvStorageConfig := storage.NewConfig(os.Getenv("NODE_STORAGE_REGION"),
			os.Getenv("NODE_STORAGE_ACCESS_KEY"),
			os.Getenv("NODE_STORAGE_SECRET"),
			os.Getenv("NODE_STORAGE_BUCKET"),
			os.Getenv("NODE_STORAGE_ROOT"))

		var spvStorage storage.Storage
		if strings.ToLower(spvStorageConfig.Bucket) == "standalone" {
			spvStorage = storage.NewFilesystemStorage(spvStorageConfig)
		} else {
			spvStorage = storage.NewS3Storage(spvStorageConfig)
		}

		spvConfig := spvnode.NewConfig(os.Getenv("NODE_ADDRESS"),
			os.Getenv("NODE_USER_AGENT"))

		spvNode := spvnode.NewNode(spvConfig, spvStorage)

		// Network
		rpcConfig := rpcnode.NewConfig(os.Getenv("RPC_HOST"),
			os.Getenv("RPC_USERNAME"),
			os.Getenv("RPC_PASSWORD"))

		network, err := network.NewNetwork(rpcConfig, spvNode)
		if err != nil {
			panic(err)
		}

		// Wallet
		wallet, err := wallet.NewWallet(os.Getenv("PRIV_KEY"))
		if err != nil {
			panic(err)
		}

		// Contract Storage
		contractStorageConfig := storage.NewConfig(os.Getenv("CONTRACT_STORAGE_REGION"),
			os.Getenv("CONTRACT_STORAGE_ACCESS_KEY"),
			os.Getenv("CONTRACT_STORAGE_SECRET"),
			os.Getenv("CONTRACT_STORAGE_BUCKET"),
			os.Getenv("CONTRACT_STORAGE_ROOT"))

		var contractStorage storage.Storage
		if strings.ToLower(contractStorageConfig.Bucket) == "standalone" {
			contractStorage = storage.NewFilesystemStorage(contractStorageConfig)
		} else {
			contractStorage = storage.NewS3Storage(contractStorageConfig)
		}

		// Builder
		state := state.NewStateService(contractStorage)
		inspector := inspector.NewInspectorService(network)
		request := request.NewRequestService(*config, wallet, state, inspector)
		response := response.NewResponseService(*config, state)
		validator := validator.NewValidatorService(*config, wallet, state)
		broadcaster := broadcaster.NewBroadcastService(network)

		// Go
		reb := rebuilder.NewRebuilderService(network, inspector, broadcaster, request, response, validator, state)

		soft, _ := state.Read(ctx, wallet.PublicAddress)
		hard, _ := state.Read(ctx, wallet.PublicAddress)

		reb.Sync(ctx, soft, hard, wallet.PublicAddress)

		return nil
	},
}

func init() {
	cmdSync.Flags().Bool(FlagDebugMode, false, "Debug mode")
}
