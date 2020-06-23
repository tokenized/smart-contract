package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/spynode"
	"github.com/tokenized/pkg/spynode/handlers"
	"github.com/tokenized/pkg/spynode/handlers/data"
	"github.com/tokenized/pkg/storage"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Client struct {
	Wallet             Wallet
	Config             Config
	ContractAddress    bitcoin.RawAddress
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
	Net         bitcoin.Network
	Key         string  `envconfig:"CLIENT_WALLET_KEY"`
	FeeRate     float32 `default:"1.0" envconfig:"CLIENT_FEE_RATE"`
	DustFeeRate float32 `default:"1.0" envconfig:"CLIENT_DUST_FEE_RATE"`
	Contract    string  `envconfig:"CLIENT_CONTRACT_ADDRESS"`
	ContractFee uint64  `default:"1000" envconfig:"CLIENT_CONTRACT_FEE"`
	SpyNode     struct {
		Address          string `default:"127.0.0.1:8333" envconfig:"CLIENT_NODE_ADDRESS"`
		UserAgent        string `default:"/Tokenized:0.1.0/" envconfig:"CLIENT_NODE_USER_AGENT"`
		StartHash        string `envconfig:"CLIENT_START_HASH"`
		UntrustedClients int    `default:"16" envconfig:"CLIENT_UNTRUSTED_NODES"`
		SafeTxDelay      int    `default:"10" envconfig:"CLIENT_SAFE_TX_DELAY"`
		ShotgunCount     int    `default:"100" envconfig:"SHOTGUN_COUNT"`
	}
}

func Context() context.Context {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging
	logConfig := logger.NewDevelopmentConfig()

	if len(os.Getenv("CLIENT_LOG_FILE_PATH")) > 0 {
		os.MkdirAll(path.Dir(os.Getenv("CLIENT_LOG_FILE_PATH")), os.ModePerm)
		logFileName := filepath.FromSlash(os.Getenv("CLIENT_LOG_FILE_PATH"))
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file : %v\n", err)
			return nil
		}
		logConfig.Main.SetWriter(io.MultiWriter(os.Stdout, logFile))
	}

	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	// logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)

	if strings.ToUpper(os.Getenv("CLIENT_LOG_FORMAT")) == "TEXT" {
		logConfig.IsText = true
	}

	return logger.ContextWithLogConfig(ctx, logConfig)
}

func NewClient(ctx context.Context, network bitcoin.Network) (*Client, error) {
	if network == bitcoin.InvalidNet {
		return nil, errors.New("Invalid Bitcoin network specified")
	}
	client := Client{}

	// -------------------------------------------------------------------------
	// Config
	if err := envconfig.Process("API", &client.Config); err != nil {
		return nil, errors.Wrap(err, "config process")
	}

	client.Config.Net = network

	// -------------------------------------------------------------------------
	// Wallet
	err := client.Wallet.Load(ctx, client.Config.Key, os.Getenv("CLIENT_PATH"), client.Config.Net)
	if err != nil {
		return nil, errors.Wrap(err, "load wallet")
	}

	// -------------------------------------------------------------------------
	// Contract
	contractAddress, err := bitcoin.DecodeAddress(client.Config.Contract)
	if err != nil {
		return nil, errors.Wrap(err, "decode contract address")
	}
	client.ContractAddress = bitcoin.NewRawAddressFromAddress(contractAddress)
	if !bitcoin.DecodeNetMatches(contractAddress.Network(), client.Config.Net) {
		return nil, errors.Wrap(err, "Contract address encoded for wrong network")
	}
	logger.Info(ctx, "Contract address : %s", client.Config.Contract)

	return &client, nil
}

func (client *Client) setupSpyNode(ctx context.Context) error {
	spyStorage := storage.NewFilesystemStorage(storage.NewConfig("standalone", os.Getenv("CLIENT_PATH")))

	spyConfig, err := data.NewConfig(client.Config.Net, client.Config.SpyNode.Address,
		client.Config.SpyNode.UserAgent, client.Config.SpyNode.StartHash,
		client.Config.SpyNode.UntrustedClients, client.Config.SpyNode.SafeTxDelay,
		client.Config.SpyNode.ShotgunCount)
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
	client.StopOnSync = false
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

	waitChannel := make(chan error, 1)
	go func() {
		// Wait until ready
		for {
			if client.spyNode.IsReady(ctx) {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)

		// Wait for broadcast to finish
		for {
			if client.spyNode.BroadcastIsComplete(ctx) {
				waitChannel <- nil
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	time.Sleep(1 * time.Second)
	fmt.Printf("BroadcastTx\n")
	client.spyNode.BroadcastTx(ctx, tx)
	client.spyNode.HandleTx(ctx, tx)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-finishChannel:
		if err != nil {
			logger.Error(ctx, "Failed to run spynode : %s", err)
		}
		return err
	case _ = <-client.spyNodeStopChannel:
		logger.Info(ctx, "Stopping")
		return client.spyNode.Stop(ctx)
	case <-osSignals:
		logger.Info(ctx, "Shutting down on request")
		return client.spyNode.Stop(ctx)
	case <-waitChannel:
		logger.Info(ctx, "Shutting down after completion")
		return client.spyNode.Stop(ctx)
	}
}

func (client *Client) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	for _, output := range tx.TxOut {
		outputAddress, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}

		if client.Wallet.Address.Equal(outputAddress) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentToWallet : %s", tx.TxHash().String())
			return true
		}

		if client.ContractAddress.Equal(outputAddress) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches PaymentToContract : %s", tx.TxHash().String())
			return true
		}
	}

	for _, input := range tx.TxIn {
		address, err := bitcoin.RawAddressFromUnlockingScript(input.SignatureScript)
		if err != nil {
			continue
		}

		if client.Wallet.Address.Equal(address) {
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
func (client *Client) HandleTxState(ctx context.Context, msgType int, txid bitcoin.Hash32) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	var tx *wire.MsgTx
	txIndex := 0
	for i, pendingTx := range client.pendingTxs {
		if txid.Equal(pendingTx.TxHash()) {
			tx = pendingTx
			txIndex = i
			break
		}
	}

	if tx == nil {
		return nil
	}

	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		logger.Info(ctx, "Tx safe : %s", txid.String())
		client.pendingTxs = append(client.pendingTxs[:txIndex], client.pendingTxs[txIndex+1:]...)
		client.applyTx(ctx, tx, false)

	case handlers.ListenerMsgTxStateConfirm:
		logger.Info(ctx, "Tx confirmed : %s", txid.String())
		client.pendingTxs = append(client.pendingTxs[:txIndex], client.pendingTxs[txIndex+1:]...)
		client.applyTx(ctx, tx, false)

		// case handlers.ListenerMsgTxStateRevert:
		// 	logger.Info(ctx, "Tx reverted : %s", txid.String())
		//
		// 	client.applyTx(ctx, tx, true)
	}

	return nil
}

func (client *Client) applyTx(ctx context.Context, tx *wire.MsgTx, reverse bool) {
	for _, input := range tx.TxIn {
		address, err := bitcoin.RawAddressFromUnlockingScript(input.SignatureScript)
		if err != nil {
			continue
		}

		if client.Wallet.Address.Equal(address) {
			if reverse {
				spentValue, spent := client.Wallet.Unspend(&input.PreviousOutPoint, tx.TxHash())
				if spent {
					logger.Info(ctx, "Reverted send %.08f : %s", BitcoinsFromSatoshis(spentValue), tx.TxHash())
				}
			} else {
				spentValue, spent := client.Wallet.Spend(&input.PreviousOutPoint, tx.TxHash())
				if spent {
					logger.Info(ctx, "Sent %.08f : %s", BitcoinsFromSatoshis(spentValue), tx.TxHash())
				}
			}
		}
	}

	for index, output := range tx.TxOut {
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}

		if client.Wallet.Address.Equal(address) {
			if reverse {
				if client.Wallet.RemoveUTXO(tx.TxHash(), uint32(index), output.PkScript, uint64(output.Value)) {
					logger.Info(ctx, "Reverted receipt of %.08f : %d of %s",
						BitcoinsFromSatoshis(uint64(output.Value)), index, tx.TxHash())
				}
			} else {
				if client.Wallet.AddUTXO(tx.TxHash(), uint32(index), output.PkScript, uint64(output.Value)) {
					logger.Info(ctx, "Received %.08f : %d of %s",
						BitcoinsFromSatoshis(uint64(output.Value)), index, tx.TxHash())
				}
			}
		}
	}
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
