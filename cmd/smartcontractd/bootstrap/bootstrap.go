package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wallet"
)

func NewContextWithDevelopmentLogger() context.Context {
	ctx := context.Background()

	logPath := os.Getenv("LOG_FILE_PATH")
	if len(logPath) > 0 {
		os.MkdirAll(path.Dir(os.Getenv("LOG_FILE_PATH")), os.ModePerm)
		logFileName := filepath.FromSlash(os.Getenv("LOG_FILE_PATH"))
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logger.Fatal(ctx, "Failed to open log file : %v\n", err)
		}

		if strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE" {
			ctx = node.ContextWithDevelopmentLogger(ctx, logFile, os.Getenv("LOG_FORMAT"))
		} else {
			ctx = node.ContextWithProductionLogger(ctx, logFile, os.Getenv("LOG_FORMAT"))
		}
	} else {
		if strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE" {
			ctx = node.ContextWithDevelopmentLogger(ctx, os.Stdout, os.Getenv("LOG_FORMAT"))
		} else {
			ctx = node.ContextWithProductionLogger(ctx, os.Stdout, os.Getenv("LOG_FORMAT"))
		}
	}

	return ctx
}

func NewWallet() *wallet.Wallet {
	return wallet.New()
}

func NewConfigFromEnv(ctx context.Context) *config.Config {
	cfg, err := config.Environment()
	if err != nil {
		logger.Fatal(ctx, "Parsing Config : %s", err)
	}

	// Mask sensitive values
	cfgSafe := config.SafeConfig(*cfg)
	cfgJSON, err := json.MarshalIndent(cfgSafe, "", "    ")
	if err != nil {
		logger.Fatal(ctx, "Marshalling Config to JSON : %s", err)
	}
	logger.Info(ctx, "Config : %v", string(cfgJSON))

	return cfg
}

func NewMasterDB(ctx context.Context, cfg *config.Config) *db.DB {
	masterDB, err := db.New(&db.StorageConfig{
		Bucket:     cfg.Storage.Bucket,
		Root:       cfg.Storage.Root,
		MaxRetries: cfg.AWS.MaxRetries,
		RetryDelay: cfg.AWS.RetryDelay,
	})
	if err != nil {
		logger.Fatal(ctx, "Register DB : %s", err)
	}

	return masterDB
}

func NewNodeConfig(ctx context.Context, cfg *config.Config) *node.Config {
	appConfig := &node.Config{
		Net:                bitcoin.NetworkFromString(cfg.Bitcoin.Network),
		ContractProviderID: cfg.Contract.OperatorName,
		Version:            cfg.Contract.Version,
		FeeRate:            cfg.Contract.FeeRate,
		DustLimit:          cfg.Contract.DustLimit,
		RequestTimeout:     cfg.Contract.RequestTimeout,
		PreprocessThreads:  cfg.Contract.PreprocessThreads,
		IsTest:             cfg.Contract.IsTest,
	}

	feeAddress, err := bitcoin.DecodeAddress(cfg.Contract.FeeAddress)
	if err != nil {
		logger.Fatal(ctx, "Invalid fee address : %s", err)
	}
	if !bitcoin.DecodeNetMatches(feeAddress.Network(), appConfig.Net) {
		logger.Fatal(ctx, "Wrong fee address encoding network")
	}
	appConfig.FeeAddress = bitcoin.NewRawAddressFromAddress(feeAddress)

	return appConfig
}

func LoadUTXOsFromDB(ctx context.Context, masterDB *db.DB) *utxos.UTXOs {
	utxos, err := utxos.Load(ctx, masterDB)
	if err != nil {
		logger.Fatal(ctx, "Load UTXOs : %s", err)
	}

	return utxos
}

func CreateHoldingsCacheChannel(ctx context.Context) *holdings.CacheChannel {
	return &holdings.CacheChannel{}
}
