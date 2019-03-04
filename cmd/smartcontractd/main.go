package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/rpcnode"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

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
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging

	logFileName := filepath.FromSlash("tmp/main.log")
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Fatalf("main : Failed to open log file : %v", err)
		return
	}
	defer logFile.Close()

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(io.MultiWriter(os.Stdout, logFile))
	logConfig.Main.Format |= logger.IncludeSystem
	logConfig.EnableSubSystem(spynode.SubSystem)
	ctx = logger.ContextWithLogConfig(ctx, logConfig)

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	log := log.New(os.Stdout, "Node : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// -------------------------------------------------------------------------
	// Config

	var cfg struct {
		Contract struct {
			PrivateKey   string `envconfig:"PRIV_KEY"`
			OperatorName string `envconfig:"OPERATOR_NAME"`
			Version      string `envconfig:"VERSION"`
			FeeAddress   string `envconfig:"FEE_ADDRESS"`
			FeeAmount    uint64 `envconfig:"FEE_AMOUNT"`
		}
		SpyNode struct {
			Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS"`
			UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT"`
			StartHash      string `envconfig:"START_HASH"`
			UntrustedNodes int    `default:"8" envconfig:"UNTRUSTED_NODES"`
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

	log.Printf("main : Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// Mask sensitive values
	cfgSafe := cfg
	if len(cfgSafe.Contract.PrivateKey) > 0 {
		cfgSafe.Contract.PrivateKey = "*** Masked ***"
	}
	if len(cfgSafe.RpcNode.Password) > 0 {
		cfgSafe.RpcNode.Password = "*** Masked ***"
	}
	if len(cfgSafe.NodeStorage.Secret) > 0 {
		cfgSafe.NodeStorage.Secret = "*** Masked ***"
	}
	if len(cfgSafe.Storage.Secret) > 0 {
		cfgSafe.Storage.Secret = "*** Masked ***"
	}
	cfgJSON, err := json.MarshalIndent(cfgSafe, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}
	log.Printf("main : Config : %v\n", string(cfgJSON))

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

	spyConfig, err := data.NewConfig(cfg.SpyNode.Address, cfg.SpyNode.UserAgent, cfg.SpyNode.StartHash, cfg.SpyNode.UntrustedNodes)
	if err != nil {
		log.Fatalf("Failed to create spynode config : %v\n", err)
		return
	}

	spyNode := spynode.NewNode(spyConfig, spyStorage)

	// -------------------------------------------------------------------------
	// RPC Node
	rpcConfig := rpcnode.NewConfig(cfg.RpcNode.Host,
		cfg.RpcNode.Username,
		cfg.RpcNode.Password)

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
	txFilter := TxFilter{chainParams: &chaincfg.MainNetParams, contractPKH: rawPKHs[0]}
	spyNode.AddTxFilter(&txFilter)

	// -------------------------------------------------------------------------
	// Start Database / Storage

	log.Println("main : Started : Initialize Database")

	masterDB, err := db.New(&db.StorageConfig{
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

	appConfig := &node.Config{
		ContractProviderID: cfg.Contract.OperatorName,
		Version:            cfg.Contract.Version,
		FeeAddress:         cfg.Contract.FeeAddress,
		FeeValue:           cfg.Contract.FeeAmount,
		DustLimit:          546,
	}

	// -------------------------------------------------------------------------
	// Register Hooks

	appHandlers := handlers.API(log, masterWallet, appConfig, masterDB)

	node := listeners.NewServer(rpcNode, spyNode, appHandlers, rawPKHs[0])

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Print("main : Node Running")
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
		log.Fatalf("main : Error starting server: %v", err)

	case <-osSignals:
		log.Println("main : Start shutdown...")

		// Asking listener to shutdown and load shed.
		if err := node.Stop(ctx); err != nil {
			log.Fatalf("main : Could not stop spvnode server: %v", err)
		}
	}
}

var (
	// Tokenized.com OP_RETURN script signature
	// 0x6a <OP_RETURN>
	// 0x0d <Push next 13 bytes>
	// 0x746f6b656e697a65642e636f6d <"tokenized.com">
	tokenizedSignature = []byte{0x6a, 0x0d, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6d}

	// old targetVersion Protocol prefix
	oldTokenizedTargetVersion = []byte{0x0, 0x0, 0x0, 0x20}
)

// Filters for transactions with tokenized.com op return scripts.
type TxFilter struct {
	chainParams *chaincfg.Params
	contractPKH []byte
}

func (filter *TxFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	containsTokenized := false
	for _, output := range tx.TxOut {
		if IsTokenizedOpReturn(output.PkScript) || IsOldTokenizedOpReturn(output.PkScript) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.Info, 3,
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
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.Info, 3,
				"Matches PaymentToContract : %s", tx.TxHash().String())
			return true
		}
	}

	// Check if txin is from contract
	// for _, input := range tx.TxIn {
	// }

	// TODO Not sure if all relevant txs will have output to contract or if some might only have input from contract
	return false
}

// Checks if a script carries the tokenized.com protocol signature
func IsTokenizedOpReturn(pkScript []byte) bool {
	if len(pkScript) < len(tokenizedSignature) {
		return false
	}
	return bytes.Equal(pkScript[:len(tokenizedSignature)], tokenizedSignature)
}

func IsOldTokenizedOpReturn(pkScript []byte) bool {
	if len(pkScript) < 20 {
		return false // This isn't long enough to be a sane message
	}
	if pkScript[0] != 0x6a {
		return false // This isn't an OP_RETURN
	}

	version := make([]byte, 4)

	// Get the version. Where that is, depends on the message structure.
	if pkScript[1] < 0x4c { // single byte push op code
		version = pkScript[2:6]
	} else { // assume no more than two byte push op code
		version = pkScript[3:7]
	}

	return bytes.Equal(version, oldTokenizedTargetVersion)
}
