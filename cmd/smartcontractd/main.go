package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/crypto/ripemd160"
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

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(io.MultiWriter(os.Stdout, logFile))
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	ctx = logger.ContextWithLogConfig(ctx, logConfig)

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
	cfgSafe := config.SafeConfig(cfg)
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
		FeeAddress:         cfg.Contract.FeeAddress,
		FeeValue:           cfg.Contract.FeeAmount,
		FeeRate:            cfg.Contract.FeeRate,
		DustLimit:          cfg.Contract.DustLimit,
		ChainParams:        config.NewChainParams(cfg.Bitcoin.Network),
	}

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
	if err := masterWallet.Register(cfg.Contract.PrivateKey); err != nil {
		panic(err)
	}

	// -------------------------------------------------------------------------
	// Tx Filter

	rawPKHs, err := masterWallet.KeyStore.GetRawPubKeyHashes()
	if err != nil {
		panic(err)
	}
	if len(rawPKHs) != 1 {
		panic("More than one key in wallet")
	}
	txFilter := NewTxFilter(&chaincfg.MainNetParams, rawPKHs[0])
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

	appHandlers := handlers.API(masterWallet, appConfig, masterDB, handlers.NewTxCache())

	node := listeners.NewServer(appConfig, rpcNode, spyNode, appHandlers, rawPKHs[0])

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
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
}

var (
	// Tokenized.com OP_RETURN script signature
	// 0x6a <OP_RETURN>
	// 0x0d <Push next 13 bytes>
	// 0x746f6b656e697a65642e636f6d <"tokenized.com">
	tokenizedSignature = []byte{0x6a, 0x0d, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6d}
)

// Filters for transactions with tokenized.com op return scripts.
type TxFilter struct {
	chainParams *chaincfg.Params
	contractPKH []byte
	hash256     hash.Hash
	hash160     hash.Hash
}

func NewTxFilter(chainParams *chaincfg.Params, contractPKH []byte) *TxFilter {
	result := TxFilter{
		chainParams: chainParams,
		contractPKH: contractPKH,
		hash256:     sha256.New(),
		hash160:     ripemd160.New(),
	}

	return &result
}

func (filter *TxFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	containsTokenized := false
	for _, output := range tx.TxOut {
		if IsTokenizedOpReturn(output.PkScript) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches TokenizedFilter : %s", tx.TxHash().String())
			containsTokenized = true
			break
		}
	}

	if !containsTokenized {
		return false
	}

	// Check if relevant to contract
	for _, output := range tx.TxOut {
		// TODO Remove extra logic in this. Should only check if P2PKH and get raw 20 bytes for PKH
		class, addresses, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, filter.chainParams)
		if err != nil {
			continue
		}
		if class == txscript.PubKeyHashTy && bytes.Equal(addresses[0].ScriptAddress(), filter.contractPKH) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentToContract : %s", tx.TxHash().String())
			return true
		}
	}

	// Check if txin is from contract
	// Reject responses don't go to the contract. They are from contract to request sender.
	for _, input := range tx.TxIn {
		pushes, err := txscript.PushedData(input.SignatureScript)
		if err != nil {
			continue
		}
		if len(pushes) != 2 {
			continue
		}

		// Calculate RIPEMD160(SHA256(PublicKey))
		filter.hash256.Reset()
		filter.hash256.Write(pushes[1])
		filter.hash160.Reset()
		filter.hash160.Write(filter.hash256.Sum(nil))
		pkh := filter.hash160.Sum(nil)

		if bytes.Equal(pkh, filter.contractPKH) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentFromContract : %s", tx.TxHash().String())
			return true
		}
	}

	return false
}

// Checks if a script carries the tokenized.com protocol signature
func IsTokenizedOpReturn(pkScript []byte) bool {
	if len(pkScript) < len(tokenizedSignature) {
		return false
	}
	return bytes.Equal(pkScript[:len(tokenizedSignature)], tokenizedSignature)
}
