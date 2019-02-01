package main

import (
	"encoding/json"
	"strings"

	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/kelseyhightower/envconfig"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// SPV Node
//
func main() {
	// -------------------------------------------------------------------------
	// Logging

	_, log := logger.NewLoggerWithContext()

	// -------------------------------------------------------------------------
	// Config

	var cfg struct {
		SpvNode struct {
			Address   string `envconfig:"NODE_ADDRESS"`
			UserAgent string `envconfig:"NODE_USER_AGENT"`
		}
		Storage struct {
			Region    string `default:"ap-southeast-2" envconfig:"NODE_STORAGE_REGION"`
			AccessKey string `envconfig:"NODE_STORAGE_ACCESS_KEY"`
			Secret    string `envconfig:"NODE_STORAGE_SECRET"`
			Bucket    string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
			Root      string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
		}
	}

	if err := envconfig.Process("API", &cfg); err != nil {
		log.Fatalf("main : Parsing Config : %v", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info("main : Started : Application Initializing")
	defer log.Info("main : Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}

	log.Infof("main : Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// TODO: Mask sensitive values
	log.Infof("main : Config : %v\n", string(cfgJSON))

	// -------------------------------------------------------------------------
	// Start Storage

	storeConfig := storage.NewConfig(cfg.Storage.Region,
		cfg.Storage.AccessKey,
		cfg.Storage.Secret,
		cfg.Storage.Bucket,
		cfg.Storage.Root)

	var spvStorage storage.Storage
	if strings.ToLower(storeConfig.Bucket) == "standalone" {
		spvStorage = storage.NewFilesystemStorage(storeConfig)
	} else {
		spvStorage = storage.NewS3Storage(storeConfig)
	}

	// -------------------------------------------------------------------------
	// Start Node Service

	config := spvnode.NewConfig(cfg.SpvNode.Address, cfg.SpvNode.UserAgent)

	n := spvnode.NewNode(config, spvStorage)
	if err := n.Start(); err != nil {
		panic(err)
	}
}
