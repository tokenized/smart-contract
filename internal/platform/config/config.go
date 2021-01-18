package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config is used to hold all runtime configuration.
type Config struct {
	Contract struct {
		PrivateKey  string  `envconfig:"PRIV_KEY"`
		FeeAddress  string  `envconfig:"FEE_ADDRESS"`
		FeeRate     float32 `default:"1.0" envconfig:"FEE_RATE"`
		DustFeeRate float32 `default:"1.0" envconfig:"DUST_FEE_RATE"`

		RequestTimeout    uint64  `default:"60000000000" envconfig:"REQUEST_TIMEOUT"` // Default 1 minute
		PreprocessThreads int     `default:"4" envconfig:"PREPROCESS_THREADS"`
		IsTest            bool    `default:"true" envconfig:"IS_TEST"`
		MinFeeRate        float32 `default:"0.5" envconfig:"MIN_FEE_RATE"`
	}
	Bitcoin struct {
		Network string `default:"mainnet" envconfig:"BITCOIN_CHAIN"`
	}
	SpyNode struct {
		Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS"`
		UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT"`
		StartHash      string `envconfig:"START_HASH"`
		UntrustedNodes int    `default:"25" envconfig:"UNTRUSTED_NODES"`
		SafeTxDelay    int    `default:"2000" envconfig:"SAFE_TX_DELAY"`
		ShotgunCount   int    `default:"100" envconfig:"SHOTGUN_COUNT"`
		MaxRetries     int    `default:"25" envconfig:"NODE_MAX_RETRIES"`
		RetryDelay     int    `default:"5000" envconfig:"NODE_RETRY_DELAY"`
	}
	RpcNode struct {
		Host       string `envconfig:"RPC_HOST"`
		Username   string `envconfig:"RPC_USERNAME"`
		Password   string `envconfig:"RPC_PASSWORD"`
		MaxRetries int    `default:"10" envconfig:"RPC_MAX_RETRIES"`
		RetryDelay int    `default:"2000" envconfig:"RPC_RETRY_DELAY"`
	}
	AWS struct {
		Region          string `default:"ap-southeast-2" envconfig:"AWS_REGION" json:"AWS_REGION"`
		AccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" json:"AWS_ACCESS_KEY_ID"`
		SecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" json:"AWS_SECRET_ACCESS_KEY"`
		MaxRetries      int    `default:"10" envconfig:"AWS_MAX_RETRIES"`
		RetryDelay      int    `default:"2000" envconfig:"AWS_RETRY_DELAY"`
	}
	NodeStorage struct {
		Bucket string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
		Root   string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
	}
	Storage struct {
		Bucket string `default:"standalone" envconfig:"CONTRACT_STORAGE_BUCKET"`
		Root   string `default:"./tmp" envconfig:"CONTRACT_STORAGE_ROOT"`
	}
}

// SafeConfig masks sensitive config values
func SafeConfig(cfg Config) *Config {
	cfgSafe := cfg

	if len(cfgSafe.Contract.PrivateKey) > 0 {
		cfgSafe.Contract.PrivateKey = "*** Masked ***"
	}
	if len(cfgSafe.RpcNode.Password) > 0 {
		cfgSafe.RpcNode.Password = "*** Masked ***"
	}
	if len(cfgSafe.AWS.AccessKeyID) > 0 {
		cfgSafe.AWS.AccessKeyID = "*** Masked ***"
	}
	if len(cfgSafe.AWS.SecretAccessKey) > 0 {
		cfgSafe.AWS.SecretAccessKey = "*** Masked ***"
	}

	return &cfgSafe
}

// Environment returns configuration sourced from environment variables
func Environment() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("NODE", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
