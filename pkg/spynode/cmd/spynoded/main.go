package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/kelseyhightower/envconfig"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

func main() {

	// -------------------------------------------------------------------------
	// Logging
	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.AddFile("./tmp/main.log")
	logConfig.Main.Format |= logger.IncludeSystem
	logConfig.EnableSubSystem(spynode.SubSystem)
	ctx := logger.ContextWithLogConfig(context.Background(), logConfig)

	// -------------------------------------------------------------------------
	// Config

	var cfg struct {
		Node struct {
			Address        string `envconfig:"NODE_ADDRESS"`
			UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT"`
			StartHash      string `envconfig:"START_HASH"`
			UntrustedNodes string `default:"8" envconfig:"UNTRUSTED_NODES"`
		}
		NodeStorage struct {
			Region    string `default:"ap-southeast-2" envconfig:"NODE_STORAGE_REGION"`
			AccessKey string `envconfig:"NODE_STORAGE_ACCESS_KEY"`
			Secret    string `envconfig:"NODE_STORAGE_SECRET"`
			Bucket    string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
			Root      string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
		}
	}

	if err := envconfig.Process("Node", &cfg); err != nil {
		logger.Log(ctx, logger.Info, "Parsing Config : %v", err)
	}

	logger.Log(ctx, logger.Info, "Started : Application Initializing")
	defer log.Println("Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		logger.Log(ctx, logger.Fatal, "Marshalling Config to JSON : %v", err)
	}

	logger.Log(ctx, logger.Info, "Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// TODO: Mask sensitive values
	logger.Log(ctx, logger.Info, "Config : %v\n", string(cfgJSON))

	// -------------------------------------------------------------------------
	// Storage
	storageConfig := storage.NewConfig(cfg.NodeStorage.Region,
		cfg.NodeStorage.AccessKey,
		cfg.NodeStorage.Secret,
		cfg.NodeStorage.Bucket,
		cfg.NodeStorage.Root)

	var store storage.Storage
	if strings.ToLower(storageConfig.Bucket) == "standalone" {
		store = storage.NewFilesystemStorage(storageConfig)
	} else {
		store = storage.NewS3Storage(storageConfig)
	}

	// -------------------------------------------------------------------------
	// Node Config
	untrustedNodes, err := strconv.Atoi(cfg.Node.UntrustedNodes)
	if err != nil {
		logger.Log(ctx, logger.Error, "Invalid untrusted nodes count %s : %v\n", cfg.Node.UntrustedNodes, err)
		return
	}
	nodeConfig, err := data.NewConfig(cfg.Node.Address, cfg.Node.UserAgent,
		cfg.Node.StartHash, untrustedNodes)
	if err != nil {
		logger.Log(ctx, logger.Error, "Failed to create node config : %s\n", err)
		return
	}

	// -------------------------------------------------------------------------
	// Node

	node := spynode.NewNode(nodeConfig, store)

	logListener := LogListener{ctx: ctx}
	node.RegisterListener(&logListener)

	node.AddTxFilter(TokenizedFilter{})
	node.AddTxFilter(OPReturnFilter{})

	signals := make(chan os.Signal, 1)
	go func() {
		signal := <-signals
		logger.Log(ctx, logger.Info, "Received signal : %s\n", signal)
		if signal == os.Interrupt {
			logger.Log(ctx, logger.Info, "Stopping node\n")
			node.Stop(ctx)
		}
	}()

	// -------------------------------------------------------------------------
	// Start Node Service

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		logger.Log(ctx, logger.Info, "Node Running")
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
		logger.Log(ctx, logger.Fatal, "Error starting server: %v", err)

	case <-osSignals:
		logger.Log(ctx, logger.Info, "Start shutdown...")

		// Asking listener to shutdown and load shed.
		if err := node.Stop(ctx); err != nil {
			logger.Log(ctx, logger.Fatal, "Could not stop spvnode server: %v", err)
		}
	}
}

type LogListener struct {
	ctx   context.Context
	mutex sync.Mutex
}

func (listener LogListener) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)

	switch msgType {
	case handlers.ListenerMsgBlock:
		logger.Log(listener.ctx, logger.Info, "New Block (%d) : %s", block.Height, block.Hash.String())
	case handlers.ListenerMsgBlockRevert:
		logger.Log(listener.ctx, logger.Info, "Reverted Block (%d) : %s", block.Height, block.Hash.String())
	}

	return nil
}

func (listener LogListener) HandleTx(ctx context.Context, msg *wire.MsgTx) (bool, error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)
	logger.Log(ctx, logger.Info, "Tx : %s", msg.TxHash().String())

	return true, nil
}

func (listener LogListener) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)

	switch msgType {
	case handlers.ListenerMsgTxStateConfirm:
		logger.Log(listener.ctx, logger.Info, "Tx confirm : %s", txid.String())
	case handlers.ListenerMsgTxStateRevert:
		logger.Log(listener.ctx, logger.Info, "Tx revert : %s", txid.String())
	case handlers.ListenerMsgTxStateCancel:
		logger.Log(listener.ctx, logger.Info, "Tx cancel : %s", txid.String())
	case handlers.ListenerMsgTxStateUnsafe:
		logger.Log(listener.ctx, logger.Info, "Tx unsafe : %s", txid.String())
	}

	return nil
}

func (listener LogListener) HandleInSync(ctx context.Context) error {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)

	logger.Log(listener.ctx, logger.Info, "In Sync")
	return nil
}

var (
	// Tokenized.com OP_RETURN script signature
	// 0x6a <OP_RETURN>
	// 0x0d <Push next 13 bytes>
	// 0x746f6b656e697a65642e636f6d <"tokenized.com">
	tokenizedSignature = []byte{0x6a, 0x0d, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6d}
)

// Filters for transactions with tokenized.com op return scripts.
type TokenizedFilter struct{}

func (filter TokenizedFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	for _, output := range tx.TxOut {
		if IsTokenizedOpReturn(output.PkScript) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.Info, 3,
				"Matches TokenizedFilter : %s", tx.TxHash().String())
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

// Filters for transactions with tokenized.com op return scripts.
type OPReturnFilter struct{}

func (filter OPReturnFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	for _, output := range tx.TxOut {
		if IsOpReturn(output.PkScript) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.Info, 3,
				"Matches OPReturnFilter : %s", tx.TxHash().String())
			return true
		}
	}
	return false
}

// Checks if a script carries the tokenized.com protocol signature
func IsOpReturn(pkScript []byte) bool {
	if len(pkScript) == 0 {
		return false
	}
	return pkScript[0] == 0x6a
}
