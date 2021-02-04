package bootstrap

import (
	"context"
	"os"
	"strings"

	"github.com/tokenized/config"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/holdings"
	smartContractConfig "github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/wallet"
)

func NewContextWithDevelopmentLogger() context.Context {
	ctx := context.Background()

	logPath := os.Getenv("LOG_FILE_PATH")
	ctx = node.ContextWithLogger(ctx, strings.ToUpper(os.Getenv("DEVELOPMENT")) == "TRUE",
		strings.ToUpper(os.Getenv("LOG_FORMAT")) == "TEXT", logPath)

	return ctx
}

func NewWallet() *wallet.Wallet {
	return wallet.New()
}

func NewConfig(ctx context.Context) *smartContractConfig.Config {

	cfg := &smartContractConfig.Config{}
	// load config using sane fallbacks
	if err := config.LoadConfig(ctx, &cfg); err != nil {
		logger.Fatal(ctx, "main : LoadConfig : %v", err)
	}

	config.DumpSafe(ctx, &cfg)

	return cfg
}

func NewMasterDB(ctx context.Context, cfg *smartContractConfig.Config) *db.DB {
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

func NewMasterDBFromValues(ctx context.Context, bucket, root string,
	maxRetries, retryDelay int) *db.DB {
	masterDB, err := db.New(&db.StorageConfig{
		Bucket:     bucket,
		Root:       root,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
	})
	if err != nil {
		logger.Fatal(ctx, "Register DB : %s", err)
	}

	return masterDB
}

func NewNodeConfig(ctx context.Context, cfg *smartContractConfig.Config) *node.Config {
	appConfig := &node.Config{
		Net:               bitcoin.NetworkFromString(cfg.Bitcoin.Network),
		FeeRate:           cfg.Contract.FeeRate,
		DustFeeRate:       cfg.Contract.DustFeeRate,
		MinFeeRate:        cfg.Contract.MinFeeRate,
		RequestTimeout:    cfg.Contract.RequestTimeout,
		PreprocessThreads: cfg.Contract.PreprocessThreads,
		IsTest:            cfg.Contract.IsTest,
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

func NewNodeConfigFromValues(
	net bitcoin.Network,
	isTest bool,
	feeRate, dustFeeRate, minFeeRate float32,
	requestTimeout uint64,
	preprocessThreads int,
	feeAddress bitcoin.RawAddress) *node.Config {

	return &node.Config{
		Net:               net,
		IsTest:            isTest,
		FeeRate:           feeRate,
		DustFeeRate:       dustFeeRate,
		MinFeeRate:        minFeeRate,
		RequestTimeout:    requestTimeout,
		PreprocessThreads: preprocessThreads,
		FeeAddress:        feeAddress,
	}
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
