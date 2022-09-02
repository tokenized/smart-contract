package main

import (
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/storage"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/bootstrap"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	spynodeBootstrap "github.com/tokenized/spynode/cmd/spynoded/bootstrap"
	"github.com/tokenized/threads"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// Smart Contract Daemon
//

func main() {
	// ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging

	ctx := bootstrap.NewContextWithDevelopmentLogger()

	// -------------------------------------------------------------------------
	// Config

	cfg := bootstrap.NewConfig(ctx)

	// -------------------------------------------------------------------------
	// App Starting

	logger.Info(ctx, "Started : Application Initializing")
	defer logger.Info(ctx, "Completed")
	defer func() {
		if err := recover(); err != nil {
			logger.Error(ctx, "Panic : %s : %s", err, string(debug.Stack()))
		}
	}()

	logger.Info(ctx, "Build %v (%v on %v)", buildVersion, buildUser, buildDate)

	// -------------------------------------------------------------------------
	// Node Config

	logger.Info(ctx, "Configuring for %s network", cfg.Bitcoin.Network)

	appConfig := bootstrap.NewNodeConfig(ctx, cfg)

	// -------------------------------------------------------------------------
	// RPC Node
	rpcConfig := &rpcnode.Config{
		Host:                   cfg.RpcNode.Host,
		Username:               cfg.RpcNode.Username,
		Password:               cfg.RpcNode.Password,
		MaxRetries:             cfg.RpcNode.MaxRetries,
		RetryDelay:             cfg.RpcNode.RetryDelay,
		IgnoreAlreadyInMempool: true, // multiple clients might send the same tx
	}

	rpcNode, err := rpcnode.NewNode(rpcConfig)
	if err != nil {
		panic(err)
	}

	// -------------------------------------------------------------------------
	// SPY Node
	spyStorageConfig := storage.NewConfig(cfg.NodeStorage.Bucket, cfg.NodeStorage.Root)
	spyStorageConfig.SetupRetry(cfg.AWS.MaxRetries, cfg.AWS.RetryDelay)

	var spyStorage storage.Storage
	if strings.ToLower(spyStorageConfig.Bucket) == "standalone" {
		spyStorage = storage.NewFilesystemStorage(spyStorageConfig)
	} else {
		spyStorage = storage.NewS3Storage(spyStorageConfig)
	}

	spyConfig, err := spynodeBootstrap.NewConfig(appConfig.Net, appConfig.IsTest,
		cfg.SpyNode.Address, cfg.SpyNode.UserAgent, cfg.SpyNode.StartHash,
		cfg.SpyNode.UntrustedNodes, cfg.SpyNode.SafeTxDelay, cfg.SpyNode.ShotgunCount,
		cfg.SpyNode.MaxRetries, cfg.SpyNode.RetryDelay, cfg.SpyNode.RequestMempool)
	if err != nil {
		logger.Fatal(ctx, "Failed to create spynode config : %s", err)
		return
	}

	spyNode := spynodeBootstrap.NewNode(spyConfig, spyStorage, rpcNode, rpcNode)

	// -------------------------------------------------------------------------
	// Wallet

	masterWallet := bootstrap.NewWallet()
	if err := masterWallet.Register(cfg.Contract.PrivateKey, appConfig.Net); err != nil {
		panic(err)
	}

	contractAddress := bitcoin.NewAddressFromRawAddress(masterWallet.KeyStore.GetAddresses()[0],
		appConfig.Net)
	logger.Info(ctx, "Contract address : %s", contractAddress.String())

	// -------------------------------------------------------------------------
	// Start Database / Storage

	logger.Info(ctx, "Started : Initialize Database")

	masterDB := bootstrap.NewMasterDB(ctx, cfg)

	defer masterDB.Close()

	// -------------------------------------------------------------------------
	// Register Hooks
	sch := scheduler.Scheduler{}

	utxos := bootstrap.LoadUTXOsFromDB(ctx, masterDB)

	holdingsChannel := bootstrap.CreateHoldingsCacheChannel(ctx)

	tracer := filters.NewTracer()

	appHandlers, apiErr := handlers.API(
		ctx,
		masterWallet,
		appConfig,
		masterDB,
		tracer,
		&sch,
		spyNode,
		utxos,
		holdingsChannel,
	)

	if apiErr != nil {
		logger.Fatal(ctx, "Generate API : %s", apiErr)
	}

	node := listeners.NewServer(
		masterWallet,
		appHandlers,
		appConfig,
		masterDB,
		spyNode,
		spyNode,
		&sch,
		tracer,
		utxos,
		holdingsChannel,
	)

	if err := node.SyncWallet(ctx); err != nil {
		logger.Fatal(ctx, "Load Wallet : %s", err)
	}

	if err := node.Load(ctx); err != nil {
		logger.Fatal(ctx, "Load Server : %s", err)
	}

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	wait := sync.WaitGroup{}

	// Start the service listening for requests.
	nodeThread, nodeComplete := threads.NewInterruptableThreadComplete("Node", node.Run, &wait)

	// -------------------------------------------------------------------------
	// Start Spynode

	spyNodeThread, spyNodeComplete := threads.NewUninterruptableThreadComplete("SpyNode",
		spyNode.Run, &wait)

	// -------------------------------------------------------------------------
	// Shutdown

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	nodeThread.Start(ctx)
	spyNodeThread.Start(ctx)

	// -------------------------------------------------------------------------
	// Stop API Service

	// Blocking main and waiting for shutdown.
	select {
	case err := <-nodeComplete:
		if err != nil {
			logger.Error(ctx, "Error starting server: %s", err)
		}

		if err := spyNode.Stop(ctx); err != nil {
			logger.Error(ctx, "Could not stop spynode: %s", err)
		}

	case err := <-spyNodeComplete:
		if err != nil {
			logger.Error(ctx, "Error starting server: %s", err)
		}

	case <-osSignals:
		logger.Info(ctx, "Shutting down")

		if err := spyNode.Stop(ctx); err != nil {
			logger.Error(ctx, "Could not stop spynode: %s", err)
		}
	}

	// Block until goroutines finish as a result of Stop()
	nodeThread.Stop(ctx)
	wait.Wait()

	if err := utxos.Save(ctx, masterDB); err != nil {
		logger.Error(ctx, "Save UTXOs : %s", err)
	}
}
