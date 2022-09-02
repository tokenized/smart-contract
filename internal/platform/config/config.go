package config

// Config is used to hold all runtime configuration.
type Config struct {
	Contract struct {
		PrivateKey  string  `envconfig:"PRIV_KEY" json:"PRIV_KEY" masked:"true"`
		FeeAddress  string  `envconfig:"FEE_ADDRESS" json:"FEE_ADDRESS"`
		FeeRate     float32 `default:"0.05" envconfig:"FEE_RATE" json:"FEE_RATE"`
		DustFeeRate float32 `default:"0.05" envconfig:"DUST_FEE_RATE" json:"DUST_FEE_RATE"`

		RequestTimeout    uint64  `default:"60000000000" envconfig:"REQUEST_TIMEOUT" json:"REQUEST_TIMEOUT"` // Default 1 minute
		PreprocessThreads int     `default:"4" envconfig:"PREPROCESS_THREADS" json:"PREPROCESS_THREADS"`
		IsTest            bool    `default:"true" envconfig:"IS_TEST" json:"IS_TEST"`
		MinFeeRate        float32 `default:"0.05" envconfig:"MIN_FEE_RATE" json:"MIN_FEE_RATE"`
	}
	Bitcoin struct {
		Network string `default:"mainnet" envconfig:"BITCOIN_CHAIN" json:"BITCOIN_CHAIN"`
	}
	SpyNode struct {
		Address        string `default:"127.0.0.1:8333" envconfig:"NODE_ADDRESS" json:"NODE_ADDRESS"`
		UserAgent      string `default:"/Tokenized:0.1.0/" envconfig:"NODE_USER_AGENT" json:"NODE_USER_AGENT"`
		StartHash      string `envconfig:"START_HASH" json:"START_HASH"`
		UntrustedNodes int    `default:"25" envconfig:"UNTRUSTED_NODES" json:"UNTRUSTED_NODES"`
		SafeTxDelay    int    `default:"2000" envconfig:"SAFE_TX_DELAY" json:"SAFE_TX_DELAY"`
		ShotgunCount   int    `default:"100" envconfig:"SHOTGUN_COUNT" json:"SHOTGUN_COUNT"`
		MaxRetries     int    `default:"25" envconfig:"NODE_MAX_RETRIES" json:"NODE_MAX_RETRIES"`
		RetryDelay     int    `default:"5000" envconfig:"NODE_RETRY_DELAY" json:"NODE_RETRY_DELAY"`
		RequestMempool bool   `default:"true" envconfig:"REQUEST_MEMPOOL" json:"REQUEST_MEMPOOL"`
	}
	RpcNode struct {
		Host       string `envconfig:"RPC_HOST" json:"RPC_HOST"`
		Username   string `envconfig:"RPC_USERNAME" json:"RPC_USERNAME"`
		Password   string `envconfig:"RPC_PASSWORD" json:"RPC_PASSWORD" masked:"true"`
		MaxRetries int    `default:"10" envconfig:"RPC_MAX_RETRIES" json:"RPC_MAX_RETRIES"`
		RetryDelay int    `default:"2000" envconfig:"RPC_RETRY_DELAY" json:"RPC_RETRY_DELAY"`
	}
	AWS struct {
		Region          string `default:"ap-southeast-2" envconfig:"AWS_REGION" json:"AWS_REGION"`
		AccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" json:"AWS_ACCESS_KEY_ID"`
		SecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" json:"AWS_SECRET_ACCESS_KEY" masked:"true"`
		MaxRetries      int    `default:"10" envconfig:"AWS_MAX_RETRIES" json:"AWS_MAX_RETRIES"`
		RetryDelay      int    `default:"2000" envconfig:"AWS_RETRY_DELAY" json:"AWS_RETRY_DELAY"`
	}
	NodeStorage struct {
		Bucket string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET" json:"NODE_STORAGE_BUCKET"`
		Root   string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT" json:"NODE_STORAGE_ROOT"`
	}
	Storage struct {
		Bucket string `default:"standalone" envconfig:"CONTRACT_STORAGE_BUCKET" json:"CONTRACT_STORAGE_BUCKET"`
		Root   string `default:"./tmp" envconfig:"CONTRACT_STORAGE_ROOT" json:"CONTRACT_STORAGE_ROOT"`
	}
}
