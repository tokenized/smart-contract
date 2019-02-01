package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"

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

	log := log.New(os.Stdout, "Node : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// -------------------------------------------------------------------------
	// Config

	var cfg struct {
		SpvNode struct {
			Address   string `envconfig:"NODE_ADDRESS"`
			UserAgent string `envconfig:"NODE_USER_AGENT"`
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
		log.Fatalf("main : Parsing Config : %v", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Println("main : Started : Application Initializing")
	defer log.Println("main : Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}

	log.Printf("main : Build %v (%v on %v)\n", buildVersion, buildUser, buildDate)

	// TODO: Mask sensitive values
	log.Printf("main : Config : %v\n", string(cfgJSON))

	// -------------------------------------------------------------------------
	// Node

	spvStorageConfig := storage.NewConfig(cfg.NodeStorage.Region,
		cfg.NodeStorage.AccessKey,
		cfg.NodeStorage.Secret,
		cfg.NodeStorage.Bucket,
		cfg.NodeStorage.Root)

	var spvStorage storage.Storage
	if strings.ToLower(spvStorageConfig.Bucket) == "standalone" {
		spvStorage = storage.NewFilesystemStorage(spvStorageConfig)
	} else {
		spvStorage = storage.NewS3Storage(spvStorageConfig)
	}

	spvConfig := spvnode.NewConfig(cfg.SpvNode.Address, cfg.SpvNode.UserAgent)

	spvNode := spvnode.NewNode(spvConfig, spvStorage)

	if err := spvNode.Start(); err != nil {
		panic(err)
	}
}
