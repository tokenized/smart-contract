package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Client struct {
	Wallet             Wallet
	Config             Config
	ContractPKH        []byte
	spyNode            *spynode.Node
	spyNodeStopChannel chan error
	blocksAdded        int
	blockHeight        int
}

type Config struct {
	ChainParams chaincfg.Params
	Key         string  `envconfig:"CLIENT_WALLET_KEY"`
	FeeRate     float32 `default:"1.1" envconfig:"CLIENT_FEE_RATE"`
	DustLimit   uint64  `default:"546" envconfig:"CLIENT_DUST_LIMIT"`
	Contract    string  `envconfig:"CLIENT_CONTRACT_ADDRESS"`
	ContractFee uint64  `default:"1000" envconfig:"CLIENT_CONTRACT_FEE"`
	SpyNode     struct {
		Address          string `default:"127.0.0.1:8333" envconfig:"CLIENT_NODE_ADDRESS"`
		UserAgent        string `default:"/Tokenized:0.1.0/" envconfig:"CLIENT_NODE_USER_AGENT"`
		StartHash        string `envconfig:"CLIENT_START_HASH"`
		UntrustedClients int    `default:"0" envconfig:"CLIENT_UNTRUSTED_NODES"`
		SafeTxDelay      int    `default:"0" envconfig:"CLIENT_SAFE_TX_DELAY"`
	}
}

func Context() context.Context {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging
	os.MkdirAll(path.Dir(os.Getenv("CLIENT_LOG_FILE_PATH")), os.ModePerm)
	logFileName := filepath.FromSlash(os.Getenv("CLIENT_LOG_FILE_PATH"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		fmt.Printf("Failed to open log file : %v\n", err)
		return nil
	}

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(io.MultiWriter(os.Stdout, logFile))
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	// logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

func NewClient(ctx context.Context) (*Client, error) {
	client := Client{}

	// -------------------------------------------------------------------------
	// Config
	if err := envconfig.Process("API", &client.Config); err != nil {
		return nil, err
	}
	client.Config.ChainParams = chaincfg.MainNetParams
	client.Config.ChainParams.Net = 0xe8f3e1e3 // BCH MainNet Magic bytes

	// -------------------------------------------------------------------------
	// Wallet
	err := client.Wallet.Load(ctx, client.Config.Key, os.Getenv("CLIENT_PATH"), &client.Config.ChainParams)
	if err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// Contract
	address, err := btcutil.DecodeAddress(client.Config.Contract, &client.Config.ChainParams)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get contract address")
	}
	logger.Info(ctx, "Contract address : %s", address)
	client.ContractPKH = address.ScriptAddress()

	return &client, nil
}

// RunSpyNode runs spyclient to sync with the network.
func (client *Client) RunSpyNode(ctx context.Context) error {
	spyStorage := storage.NewFilesystemStorage(storage.NewConfig("", "", "", "standalone", os.Getenv("CLIENT_PATH")))

	spyConfig, err := data.NewConfig(&client.Config.ChainParams, client.Config.SpyNode.Address,
		client.Config.SpyNode.UserAgent, client.Config.SpyNode.StartHash,
		client.Config.SpyNode.UntrustedClients, client.Config.SpyNode.SafeTxDelay)
	if err != nil {
		logger.Warn(ctx, "Failed to create spynode config : %s", err)
		return err
	}

	client.spyNode = spynode.NewNode(spyConfig, spyStorage)
	client.spyNode.AddTxFilter(client)
	client.spyNode.RegisterListener(client)

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	finishChannel := make(chan error, 1)
	client.spyNodeStopChannel = make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		finishChannel <- client.spyNode.Run(ctx)
	}()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-finishChannel:
		return err
	case _ = <-client.spyNodeStopChannel:
		logger.Info(ctx, "Stopping")
		stopErr := client.spyNode.Stop(ctx)
		saveErr := client.Wallet.Save(ctx)
		if stopErr != nil {
			return stopErr
		}
		return saveErr
	case <-osSignals:
		logger.Info(ctx, "Shutting down")
		stopErr := client.spyNode.Stop(ctx)
		saveErr := client.Wallet.Save(ctx)
		if stopErr != nil {
			return stopErr
		}
		return saveErr
	}
}

func (client *Client) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	for _, output := range tx.TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}

		if bytes.Equal(client.Wallet.PublicKeyHash, pkh) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentToWallet : %s", tx.TxHash().String())
			return true
		}
	}

	for _, input := range tx.TxIn {
		pkh, err := txbuilder.PubKeyHashFromP2PKHSigScript(input.SignatureScript)
		if err != nil {
			continue
		}

		if bytes.Equal(client.Wallet.PublicKeyHash, pkh) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentFromWallet : %s", tx.TxHash().String())
			return true
		}
	}

	return false
}

// Block add and revert messages.
func (client *Client) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	switch msgType {
	case handlers.ListenerMsgBlock:
		client.blockHeight = block.Height
		client.blocksAdded++
		if client.blocksAdded%100 == 0 {
			logger.Info(ctx, "Added 100 blocks to height %d", client.blockHeight)
		}
	case handlers.ListenerMsgBlockRevert:
	}
	return nil
}

// Full message for a transaction broadcast on the network.
// Return true for txs that are relevant to ensure spynode sends further notifications for
//   that tx.
func (client *Client) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	result := false
	for _, input := range tx.TxIn {
		pkh, err := txbuilder.PubKeyHashFromP2PKHSigScript(input.SignatureScript)
		if err != nil {
			continue
		}

		if bytes.Equal(client.Wallet.PublicKeyHash, pkh) {
			// Spend UTXO
			spentValue, spent := client.Wallet.Spend(&input.PreviousOutPoint, tx.TxHash())
			if spent {
				logger.Info(ctx, "Sent Payment of %.08f : %s", BitcoinsFromSatoshis(spentValue), tx.TxHash())
			}
			result = true
		}
	}

	for index, output := range tx.TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}

		if bytes.Equal(client.Wallet.PublicKeyHash, pkh) {
			// Add UTXO
			if client.Wallet.AddUTXO(tx.TxHash(), uint32(index), output.PkScript, uint64(output.Value)) {
				logger.Info(ctx, "Received Payment of %.08f : %s", BitcoinsFromSatoshis(uint64(output.Value)), tx.TxHash())
			}
			result = true
		}
	}

	return result, nil
}

// Tx confirm, cancel, unsafe, and revert messages.
func (client *Client) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	return nil
}

// When in sync with network
func (client *Client) HandleInSync(ctx context.Context) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	// TODO Build/Send outgoing transactions

	if client.blocksAdded == 0 {
		logger.Info(ctx, "No new blocks found")
	} else {
		logger.Info(ctx, "Synchronized %d new block(s) to height %d", client.blocksAdded, client.blockHeight)
	}
	logger.Info(ctx, "Balance : %.08f", BitcoinsFromSatoshis(client.Wallet.Balance()))

	// Trigger close
	client.spyNodeStopChannel <- nil
	return nil
}
