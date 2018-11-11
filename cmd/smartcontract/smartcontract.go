package main

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/pkg/storage"
)

// Smart Contract CLI
//
func main() {
	// Logger
	ctx, log := logger.NewLoggerWithContext()

	config, err := contract.NewConfig()
	if err != nil {
		panic(err)
	}

	address, err := btcutil.DecodeAddress(config.ContractAddress, &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}

	if config.Fee.Value == 0 {
		panic("Fee is set to 0 sats")
	}

	logger.Infof("Started with config %s", *config)

	client, err := contract.NewNode(config.RPC)
	if err != nil {
		panic(err)
	}

	peerRepo := contract.NewRPCNode(client)

	docStore := storage.NewFilesystemStorage(config.Storage)

	contractState := contract.NewStateService(docStore)

	txService := contract.NewParser(peerRepo)

	cs := contract.NewContractService(contractState, config.Fee)

	builder := contract.NewContractBuilder(cs, peerRepo, txService)

	contract, err := builder.Rebuild(ctx, address)
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(*contract)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", b)
}
