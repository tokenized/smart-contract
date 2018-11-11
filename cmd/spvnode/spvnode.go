package main

import (
	"fmt"

	"github.com/tokenized/smart-contract/internal/app/logger"
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
	ctx, log := logger.NewLoggerWithContext()

	config := spvnode.NewConfig()

	store := storage.NewFilesystemStorage(config.Storage)
	stateRepo := spvnode.NewStateRepository(store)
	blockRepo := spvnode.NewBlockRepository(store)

	logger.Infof("Started %v with config %s", buildDetails(), config)

	blockService := spvnode.NewBlockService(blockRepo, stateRepo)

	n := spvnode.NewNode(config, &blockService)
	if err := n.Start(); err != nil {
		panic(err)
	}
}

// buildDetails returns a string that describes the details of the build.
func buildDetails() string {
	return fmt.Sprintf("%v (%v on %v)", buildVersion, buildUser, buildDate)
}
