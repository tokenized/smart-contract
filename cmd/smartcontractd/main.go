package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/node"
	"github.com/tokenized/smart-contract/internal/platform/db"
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

	log := log.New(os.Stdout, "Node : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

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

	log.Println("main : Started : Application Initializing")
	defer log.Println("main : Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}

	log.Printf("main : Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// TODO: Mask sensitive values
	log.Printf("main : Config : %v\n", string(cfgJSON))

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

	// Always watch Contract address
	// TODO Move this to app config, watch the address from the node instead
	contractAddr, err := btcutil.DecodeAddress(string(wallet.PublicAddress), &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	network.WatchAddress(ctx, contractAddr)

	log.Printf("main : Running Contract %s", contractAddr)

	// -------------------------------------------------------------------------
	// Start Database / Storage

	log.Println("main : Started : Initialize Database")

	masterDB, err := db.New(nil, &db.StorageConfig{
		Region:    cfg.Storage.Region,
		AccessKey: cfg.Storage.AccessKey,
		Secret:    cfg.Storage.Secret,
		Bucket:    cfg.Storage.Bucket,
		Root:      cfg.Storage.Root,
	})
	if err != nil {
		log.Fatalf("main : Register DB : %v", err)
	}
	defer masterDB.Close()

	// -------------------------------------------------------------------------
	// Node Config

	nodeConfig := &node.Config{
		ContractProviderID: cfg.Contract.OperatorName,
		Version:            cfg.Contract.Version,
		FeeAddress:         cfg.Contract.FeeAddress,
		FeeValue:           cfg.Contract.FeeAmount,
	}

	// -------------------------------------------------------------------------
	// Register Hooks

	handlers.Register(spvNode, log, nodeConfig, masterDB)

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Print("main : Node Running")
		serverErrors <- spvNode.Start()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	// -------------------------------------------------------------------------
	// Stop API Service

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		log.Fatalf("main : Error starting server: %v", err)

	case <-osSignals:
		log.Println("main : Start shutdown...")

		// Asking listener to shutdown and load shed.
		if err := spvNode.Close(); err != nil {
			log.Fatalf("main : Could not stop spvnode server: %v", err)
		}
	}
}
