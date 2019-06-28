package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config is used to hold all runtime configuration.
type Config struct {
	Contract struct {
		PrivateKey        string  `envconfig:"PRIV_KEY"`
		OperatorName      string  `envconfig:"OPERATOR_NAME"`
		Version           string  `envconfig:"VERSION"`
		FeeAddress        string  `envconfig:"FEE_ADDRESS"`
		FeeRate           float32 `default:"1.1" envconfig:"FEE_RATE"`
		DustLimit         uint64  `default:"546" envconfig:"DUST_LIMIT"`
		RequestTimeout    uint64  `default:"60000000000" envconfig:"REQUEST_TIMEOUT"` // Default 1 minute
		PreprocessThreads int     `default:"4" envconfig:"PREPROCESS_THREADS"`
		IsTest            bool    `default:"true" envconfig:"IS_TEST"`
	}
	Bitcoin struct {
		Network string `default:"mainnet" envconfig:"BITCOIN_CHAIN"`
	}
	SpyNode struct {
		Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS"`
		UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT"`
		StartHash      string `envconfig:"START_HASH"`
		UntrustedNodes int    `default:"8" envconfig:"UNTRUSTED_NODES"`
		SafeTxDelay    int    `default:"2000" envconfig:"SAFE_TX_DELAY"`
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

// SafeConfig masks sensitive config values
func SafeConfig(cfg Config) *Config {
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