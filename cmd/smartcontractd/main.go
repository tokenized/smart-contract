package main

import (
	"encoding/json"
	"strings"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/node"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/kelseyhightower/envconfig"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// Smart Contract Daemon
//
func main() {
	// -------------------------------------------------------------------------
	// Logging

	ctx, log := logger.NewLoggerWithContext()

	// -------------------------------------------------------------------------
	// Config

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
	// App Starting

	log.Info("main : Started : Application Initializing")
	defer log.Info("main : Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}

	log.Infof("main : Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// TODO: Mask sensitive values
	log.Infof("main : Config : %v\n", string(cfgJSON))

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
	// Always watch Contract address
	//
	// TODO Move this to app config, watch the address from the node instead
	//

	contractAddr, err := btcutil.DecodeAddress(string(wallet.PublicAddress), &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	network.WatchAddress(ctx, contractAddr)

	log.Infof("Running contract %s", contractAddr)

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
	// Start Node Service

	n := node.NewNode(*appConfig, network, *wallet, contractStorage)
	if err := n.Start(); err != nil {
		panic(err)
	}
}
