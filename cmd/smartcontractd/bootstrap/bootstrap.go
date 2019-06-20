package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/specification/dist/golang/protocol"
)

func NewContextWithDevelopmentLogger() context.Context {
	ctx := context.Background()

	os.MkdirAll(path.Dir(os.Getenv("LOG_FILE_PATH")), os.ModePerm)
	logFileName := filepath.FromSlash(os.Getenv("LOG_FILE_PATH"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Fatal(ctx, "Failed to open log file : %v\n", err)
	}
	defer logFile.Close()

	ctx = node.ContextWithDevelopmentLogger(ctx, io.MultiWriter(os.Stdout, logFile))

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
		Region:    cfg.Storage.Region,
		AccessKey: cfg.Storage.AccessKey,
		Secret:    cfg.Storage.Secret,
		Bucket:    cfg.Storage.Bucket,
		Root:      cfg.Storage.Root,
	})
	if err != nil {
		logger.Fatal(ctx, "Register DB : %s", err)
	}

	return masterDB
}

func NewNodeConfig(ctx context.Context, cfg *config.Config) *node.Config {
	appConfig := &node.Config{
		ContractProviderID: cfg.Contract.OperatorName,
		Version:            cfg.Contract.Version,
		FeeRate:            cfg.Contract.FeeRate,
		DustLimit:          cfg.Contract.DustLimit,
		ChainParams:        config.NewChainParams(cfg.Bitcoin.Network),
		RequestTimeout:     cfg.Contract.RequestTimeout,
		PreprocessThreads:  cfg.Contract.PreprocessThreads,
		IsTest:             cfg.Contract.IsTest,
	}

	feeAddress, err := btcutil.DecodeAddress(cfg.Contract.FeeAddress, &appConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "Invalid fee address : %s", err)
	}
	appConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeAddress.ScriptAddress())

	return appConfig
}

func LoadUTXOsFromDB(ctx context.Context, masterDB *db.DB) *utxos.UTXOs {
	utxos, err := utxos.Load(ctx, masterDB)
	if err != nil {
		logger.Fatal(ctx, "Load UTXOs : %s", err)
	}

	return utxos
}
