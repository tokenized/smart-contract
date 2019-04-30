package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// Smart Contract Daemon
//
func main() {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging

	os.MkdirAll(path.Dir(os.Getenv("LOG_FILE_PATH")), os.ModePerm)
	logFileName := filepath.FromSlash(os.Getenv("LOG_FILE_PATH"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		fmt.Printf("Failed to open log file : %v\n", err)
		return
	}
	defer logFile.Close()

	ctx = node.ContextWithDevelopmentLogger(ctx, io.MultiWriter(os.Stdout, logFile))

	// -------------------------------------------------------------------------
	// Config

	cfg, err := config.Environment()
	if err != nil {
		logger.Fatal(ctx, "Parsing Config : %s", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	logger.Info(ctx, "Started : Application Initializing")
	defer logger.Info(ctx, "Completed")

	logger.Info(ctx, "Build %v (%v on %v)", buildVersion, buildUser, buildDate)

	// Mask sensitive values
	cfgSafe := config.SafeConfig(*cfg)
	cfgJSON, err := json.MarshalIndent(cfgSafe, "", "    ")
	if err != nil {
		logger.Fatal(ctx, "Marshalling Config to JSON : %s", err)
	}
	logger.Info(ctx, "Config : %v", string(cfgJSON))

	// -------------------------------------------------------------------------
	// Node Config

	appConfig := &node.Config{
		ContractProviderID: cfg.Contract.OperatorName,
		Version:            cfg.Contract.Version,
		FeeRate:            cfg.Contract.FeeRate,
		DustLimit:          cfg.Contract.DustLimit,
		ChainParams:        config.NewChainParams(cfg.Bitcoin.Network),
		RequestTimeout:     cfg.Contract.RequestTimeout,
		IsTest:             cfg.Contract.IsTest,
	}

	feeAddress, err := btcutil.DecodeAddress(cfg.Contract.FeeAddress, &appConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "Invalid fee address : %s", err)
	}
	appConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeAddress.ScriptAddress())

	// -------------------------------------------------------------------------
	// SPY Node
	spyStorageConfig := storage.NewConfig(cfg.NodeStorage.Region,
		cfg.NodeStorage.AccessKey,
		cfg.NodeStorage.Secret,
		cfg.NodeStorage.Bucket,
		cfg.NodeStorage.Root)

	var spyStorage storage.Storage
	if strings.ToLower(spyStorageConfig.Bucket) == "standalone" {
		spyStorage = storage.NewFilesystemStorage(spyStorageConfig)
	} else {
		spyStorage = storage.NewS3Storage(spyStorageConfig)
	}

	spyConfig, err := data.NewConfig(&appConfig.ChainParams, cfg.SpyNode.Address, cfg.SpyNode.UserAgent,
		cfg.SpyNode.StartHash, cfg.SpyNode.UntrustedNodes, cfg.SpyNode.SafeTxDelay)
	if err != nil {
		logger.Fatal(ctx, "Failed to create spynode config : %s", err)
		return
	}

	spyNode := spynode.NewNode(spyConfig, spyStorage)

	// -------------------------------------------------------------------------
	// RPC Node
	rpcConfig := &rpcnode.Config{
		Host:        cfg.RpcNode.Host,
		Username:    cfg.RpcNode.Username,
		Password:    cfg.RpcNode.Password,
		ChainParams: &appConfig.ChainParams,
	}

	rpcNode, err := rpcnode.NewNode(rpcConfig)
	if err != nil {
		panic(err)
	}

	// -------------------------------------------------------------------------
	// Wallet

	masterWallet := wallet.New()
	if err := masterWallet.Register(cfg.Contract.PrivateKey, &appConfig.ChainParams); err != nil {
		panic(err)
	}

	// -------------------------------------------------------------------------
	// Tx Filter

	rawPKHs, err := masterWallet.KeyStore.GetRawPubKeyHashes()
	if err != nil {
		panic(err)
	}
	tracer := listeners.NewTracer()
	txFilter := filters.NewTxFilter(&chaincfg.MainNetParams, rawPKHs, tracer)
	spyNode.AddTxFilter(txFilter)

	// -------------------------------------------------------------------------
	// Start Database / Storage

	logger.Info(ctx, "Started : Initialize Database")

	masterDB, err := db.New(&db.StorageConfig{
		Region:    cfg.Storage.Region,
		AccessKey: cfg.Storage.AccessKey,
		Secret:    cfg.Storage.Secret,
		Bucket:    cfg.Storage.Bucket,
		Root:      cfg.Storage.Root,
	})
	if err != nil {
		logger.Fatal(ctx, "Register DB : %s", err)
	}
	defer masterDB.Close()

	// -------------------------------------------------------------------------
	// Register Hooks
	sch := scheduler.Scheduler{}
	utxos, err := utxos.Load(ctx, masterDB)
	if err != nil {
		logger.Fatal(ctx, "Load UTXOs : %s", err)
	}

	appHandlers, apiErr := handlers.API(ctx, masterWallet, appConfig, masterDB, tracer,
		&sch, spyNode, utxos)
	if err != nil {
		logger.Fatal(ctx, "Generate API : %s", apiErr)
	}

	node := listeners.NewServer(masterWallet, appHandlers, appConfig, masterDB, rpcNode, spyNode,
		&sch, tracer, rawPKHs, utxos)

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// Start the service listening for requests.
	go func() {
		defer wg.Done()
		logger.Info(ctx, "Node Running")
		serverErrors <- node.Run(ctx)
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
		if err != nil {
			logger.Fatal(ctx, "Error starting server: %s", err)
		}

	case <-osSignals:
		logger.Info(ctx, "Shutting down")

		// Asking listener to shutdown and load shed.
		if err := node.Stop(ctx); err != nil {
			logger.Fatal(ctx, "Could not stop server: %s", err)
		}
	}

	// Block until goroutines finish as a result of Stop()
	wg.Wait()
	err = utxos.Save(ctx, masterDB)
	if err != nil {
		logger.Fatal(ctx, "Save UTXOs : %s", err)
	}
}
