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
	"sync"
	"syscall"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/config"
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
	TxsToSend          []*wire.MsgTx
	IncomingTx         TxChannel
	pendingTxs         []*wire.MsgTx
	StopOnSync         bool
	spyNodeInSync      bool
	lock               sync.Mutex
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
		UntrustedClients int    `default:"16" envconfig:"CLIENT_UNTRUSTED_NODES"`
		SafeTxDelay      int    `default:"10" envconfig:"CLIENT_SAFE_TX_DELAY"`
	}
}

func Context() context.Context {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging
	os.MkdirAll(path.Dir(os.Getenv("CLIENT_LOG_FILE_PATH")), os.ModePerm)
	logFileName := filepath.FromSlash(os.Getenv("CLIENT_LOG_FILE_PATH"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

func NewClient(ctx context.Context, network string) (*Client, error) {
	client := Client{}

	// -------------------------------------------------------------------------
	// Config
	if err := envconfig.Process("API", &client.Config); err != nil {
		return nil, err
	}

	client.Config.ChainParams = config.NewChainParams(network)

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

func (client *Client) setupSpyNode(ctx context.Context) error {
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
	return nil
}

// RunSpyNode runs spyclient to sync with the network.
func (client *Client) RunSpyNode(ctx context.Context, stopOnSync bool) error {
	if err := client.setupSpyNode(ctx); err != nil {
		return err
	}
	client.StopOnSync = stopOnSync
	client.IncomingTx.Open(100)
	defer client.IncomingTx.Close()
	defer func() {
		if saveErr := client.Wallet.Save(ctx); saveErr != nil {
			logger.Info(ctx, "Failed to save UTXOs : %s", saveErr)
		}
	}()

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	finishChannel := make(chan error, 1)
	client.spyNodeStopChannel = make(chan error, 1)
	client.spyNodeInSync = false

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
		return client.spyNode.Stop(ctx)
	case <-osSignals:
		logger.Info(ctx, "Shutting down")
		return client.spyNode.Stop(ctx)
	}
}

func (client *Client) OutgoingCount() int {
	return client.spyNode.OutgoingCount()
}

func (client *Client) BroadcastTx(ctx context.Context, tx *wire.MsgTx) error {
	if err := client.spyNode.BroadcastTx(ctx, tx); err != nil {
		return err
	}
	return client.spyNode.HandleTx(ctx, tx)
}

func (client *Client) BroadcastTxUntrustedOnly(ctx context.Context, tx *wire.MsgTx) error {
	if err := client.spyNode.BroadcastTxUntrustedOnly(ctx, tx); err != nil {
		return err
	}
	return nil
}

func (client *Client) StopSpyNode(ctx context.Context) error {
	return client.spyNode.Stop(ctx)
}

func (client *Client) Scan(ctx context.Context, count int) error {
	if err := client.setupSpyNode(ctx); err != nil {
		return err
	}
	return client.spyNode.Scan(ctx, count)
}

func (client *Client) AddPeer(ctx context.Context, address string, score int) error {
	if err := client.setupSpyNode(ctx); err != nil {
		return err
	}
	return client.spyNode.AddPeer(ctx, address, score)
}

func (client *Client) ShotgunTx(ctx context.Context, tx *wire.MsgTx, count int) error {
	if err := client.setupSpyNode(ctx); err != nil {
		return err
	}
	return client.spyNode.ShotgunTransmitTx(ctx, tx, count)
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

		if bytes.Equal(client.ContractPKH, pkh) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentToContract : %s", tx.TxHash().String())
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
		if client.spyNodeInSync {
			logger.Info(ctx, "New block (%d) : %s", block.Height, block.Hash.String())
		}
	case handlers.ListenerMsgBlockRevert:
	}
	return nil
}

// Full message for a transaction broadcast on the network.
// Return true for txs that are relevant to ensure spynode sends further notifications for
//   that tx.
func (client *Client) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	client.pendingTxs = append(client.pendingTxs, tx)
	client.IncomingTx.Channel <- tx
	return true, nil
}

// Tx confirm, cancel, unsafe, and revert messages.
func (client *Client) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	var tx *wire.MsgTx
	for i, pendingTx := range client.pendingTxs {
		if pendingTx.TxHash() == txid {
			tx = pendingTx
			client.pendingTxs = append(client.pendingTxs[:i], client.pendingTxs[i+1:]...)
			break
		}
	}

	if tx == nil {
		logger.Info(ctx, "Confirmed tx not found : %s", txid.String())
		return nil
	}

	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		logger.Info(ctx, "Tx safe : %s", txid.String())

	case handlers.ListenerMsgTxStateConfirm:
		logger.Info(ctx, "Tx confirmed : %s", txid.String())
		for _, input := range tx.TxIn {
			pkh, err := txbuilder.PubKeyHashFromP2PKHSigScript(input.SignatureScript)
			if err != nil {
				continue
			}

			if bytes.Equal(client.Wallet.PublicKeyHash, pkh) {
				// Spend UTXO
				spentValue, spent := client.Wallet.Spend(&input.PreviousOutPoint, tx.TxHash())
				if spent {
					logger.Info(ctx, "Confirmed sent payment of %.08f : %s", BitcoinsFromSatoshis(spentValue), tx.TxHash())
				}
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
					logger.Info(ctx, "Confirmed received payment of %.08f : %d of %s",
						BitcoinsFromSatoshis(uint64(output.Value)), index, tx.TxHash())
				}
			}
		}
	}

	return nil
}

// When in sync with network
func (client *Client) HandleInSync(ctx context.Context) error {
	client.lock.Lock()
	defer client.lock.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)

	// TODO Build/Send outgoing transactions
	for _, tx := range client.TxsToSend {
		client.spyNode.BroadcastTx(ctx, tx)
	}
	client.TxsToSend = nil

	if client.blocksAdded == 0 {
		logger.Info(ctx, "No new blocks found")
	} else {
		logger.Info(ctx, "Synchronized %d new block(s) to height %d", client.blocksAdded, client.blockHeight)
	}
	logger.Info(ctx, "Balance : %.08f", BitcoinsFromSatoshis(client.Wallet.Balance()))

	// Trigger close
	if client.StopOnSync {
		client.spyNodeStopChannel <- nil
	}
	client.spyNodeInSync = true
	return nil
}

func (client *Client) IsInSync() bool {
	client.lock.Lock()
	defer client.lock.Unlock()
	return client.spyNodeInSync
}

type TxChannel struct {
	Channel chan *wire.MsgTx
	lock    sync.Mutex
	open    bool
}

func (c *TxChannel) Add(tx *wire.MsgTx) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- tx
	return nil
}

func (c *TxChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan *wire.MsgTx, count)
	c.open = true
	return nil
}

func (c *TxChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
