package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/storage"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	buildUser    = "unknown"
)

// SPV Node
//
func main() {
	// Logger
	_, log := logger.NewLoggerWithContext()

	storeConfig := storage.NewConfig(os.Getenv("NODE_STORAGE_REGION"),
		os.Getenv("NODE_STORAGE_ACCESS_KEY"),
		os.Getenv("NODE_STORAGE_SECRET"),
		os.Getenv("NODE_STORAGE_BUCKET"),
		os.Getenv("NODE_STORAGE_ROOT"))

	var spvStorage storage.Storage
	if strings.ToLower(storeConfig.Bucket) == "standalone" {
		spvStorage = storage.NewFilesystemStorage(storeConfig)
	} else {
		spvStorage = storage.NewS3Storage(storeConfig)
	}

	config := spvnode.NewConfig(os.Getenv("NODE_ADDRESS"),
		os.Getenv("NODE_USER_AGENT"))

	// Log startup sequence
	log.Infof("Started %v with config %s", buildDetails(), config)

	n := spvnode.NewNode(config, spvStorage)
	if err := n.Start(); err != nil {
		panic(err)
	}
}

// buildDetails returns a string that describes the details of the build.
func buildDetails() string {
	return fmt.Sprintf("%v (%v on %v)", buildVersion, buildUser, buildDate)
}
