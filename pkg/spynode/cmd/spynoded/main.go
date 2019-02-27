package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"bitbucket.org/tokenized/nexus-api/pkg/spynode"
	"bitbucket.org/tokenized/nexus-api/pkg/spynode/handlers"
	"bitbucket.org/tokenized/nexus-api/pkg/spynode/handlers/data"
	"bitbucket.org/tokenized/nexus-api/pkg/spynode/logger"
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
	defer log.Println("main : Completed")

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

	node := spynode.NewNode(nodeConfig, store, listeners)

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

	if err := node.Load(ctx); err != nil {
		panic(err)
	}

	if err := node.Run(ctx); err != nil {
		panic(err)
	}
}

type LogListener struct {
	ctx   context.Context
	mutex sync.Mutex
}

func (listener LogListener) Handle(ctx context.Context, msgType int, msgValue interface{}) error {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	switch msgType {
	case handlers.ListenerMsgTx:
		value, ok := msgValue.(wire.MsgTx)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "Could not assert as wire.MsgTx")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "Tx : %s", value.TxHash().String())

	case handlers.ListenerMsgTxConfirm:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "TxConfirm : Could not assert as chainhash.Hash")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "Tx confirm : %s", value.String())

	case handlers.ListenerMsgBlock:
		value, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "Block : Could not assert as handlers.BlockMessage")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "New Block (%d) : %s", value.Height, value.Hash.String())

	case handlers.ListenerMsgBlockRevert:
		value, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "Block Revert : Could not assert as handlers.BlockMessage")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "Revert Block (%d) : %s", value.Height, value.Hash.String())

	case handlers.ListenerMsgTxRevert:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "TxRevert : Could not assert as chainhash.Hash")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "Tx revert : %s", value.String())

	case handlers.ListenerMsgTxCancel:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			logger.Log(listener.ctx, logger.Error, "TxCancel : Could not assert as chainhash.Hash")
			return nil
		}
		logger.Log(listener.ctx, logger.Info, "Tx cancel : %s", value.String())
	}
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
