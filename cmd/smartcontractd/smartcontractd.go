package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/node"
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/internal/app/rpcnode"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// Smart Contract Daemon
//
func main() {
	// Logger
	_, log := logger.NewLoggerWithContext()

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

	// Log startup sequence
	log.Infof("Started %v with config %s", buildDetails(), *config)
	log.Infof("Running contract %s", wallet.PublicAddress)

	// Smart Contract Node
	n := node.NewNode(*config, network, *wallet, contractStorage)
	if err := n.Start(); err != nil {
		panic(err)
	}
}

// buildDetails returns a string that describes the details of the build.
func buildDetails() string {
	return fmt.Sprintf("%v (%v on %v)", buildVersion, buildUser, buildDate)
}
