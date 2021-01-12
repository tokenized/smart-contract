package client

import (
	"context"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/storage"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/spynode/cmd/spynoded/bootstrap"
	"github.com/tokenized/spynode/pkg/client"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Client struct {
	Wallet             Wallet
	Config             Config
	ContractAddress    bitcoin.RawAddress
	spyNode            bootstrap.SpyNodeEmbedded
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
	IsTest      bool    `envconfig:"IS_TEST"`
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
	RpcNode struct {
		Host     string `default:"127.0.0.1:8332" envconfig:"RPC_HOST"`
		Username string `envconfig:"RPC_USERNAME"`
		Password string `envconfig:"RPC_PASSWORD"`

		// Retry attempts when calls fail.
		MaxRetries int `default:"60" envconfig:"RPC_MAX_RETRIES"`
		RetryDelay int `default:"2000", envconfig:"RPC_RETRY_DELAY"`
	}
}

func Context() context.Context {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Logging
	if len(os.Getenv("CLIENT_LOG_FILE_PATH")) > 0 {
		os.MkdirAll(path.Dir(os.Getenv("CLIENT_LOG_FILE_PATH")), os.ModePerm)
	}

	logConfig := logger.NewConfig(true, strings.ToUpper(os.Getenv("CLIENT_LOG_FORMAT")) == "TEXT",
		os.Getenv("CLIENT_LOG_FILE_PATH"))

	// logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(txbuilder.SubSystem)

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
	rpcConfig := &rpcnode.Config{
		Host:       client.Config.RpcNode.Host,
		Username:   client.Config.RpcNode.Username,
		Password:   client.Config.RpcNode.Password,
		MaxRetries: client.Config.RpcNode.MaxRetries,
		RetryDelay: client.Config.RpcNode.RetryDelay,
	}

	rpcNode, err := rpcnode.NewNode(rpcConfig)
	if err != nil {
		logger.Warn(ctx, "Failed to create rpc node : %s", err)
		return err
	}

	spyStorage := storage.NewFilesystemStorage(storage.NewConfig("standalone",
		os.Getenv("CLIENT_PATH")))

	spyConfig, err := bootstrap.NewConfig(client.Config.Net, client.Config.IsTest,
		client.Config.SpyNode.Address, client.Config.SpyNode.UserAgent,
		client.Config.SpyNode.StartHash, client.Config.SpyNode.UntrustedClients,
		client.Config.SpyNode.SafeTxDelay, client.Config.SpyNode.ShotgunCount)
	if err != nil {
		logger.Warn(ctx, "Failed to create spynode config : %s", err)
		return err
	}

	client.spyNode = bootstrap.NewNode(spyConfig, spyStorage, rpcNode)

	hashes, err := client.Wallet.Address.Hashes()
	if err != nil {
		logger.Warn(ctx, "Failed to get wallet hashes : %s", err)
		return err
	}

	for _, hash := range hashes {
		client.spyNode.SubscribePushDatas(ctx, [][]byte{hash[:]})
	}

	client.spyNode.RegisterHandler(client)
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

func (client *Client) BroadcastTx(ctx context.Context, tx *wire.MsgTx) error {
	if err := client.spyNode.SendTx(ctx, tx); err != nil {
		return err
	}
	return nil
}

func (client *Client) StopSpyNode(ctx context.Context) error {
	return client.spyNode.Stop(ctx)
}

func (client *Client) HandleTx(ctx context.Context, tx *client.Tx) {

	if tx.State.Safe {
		client.applyTx(ctx, tx.Tx, false)
	} else {
		client.pendingTxs = append(client.pendingTxs, tx.Tx)
	}

	client.IncomingTx.Channel <- tx.Tx
}

func (client *Client) HandleTxUpdate(ctx context.Context, update *client.TxUpdate) {
	if !update.State.Safe {
		return // ignore other updates for now
	}

	var tx *wire.MsgTx
	txIndex := 0
	for i, pendingTx := range client.pendingTxs {
		if update.TxID.Equal(pendingTx.TxHash()) {
			tx = pendingTx
			txIndex = i
			break
		}
	}

	if tx == nil {
		return
	}

	if update.State.MerkleProof != nil {
		logger.Info(ctx, "Tx confirmed : %s", update.TxID)
	} else {
		logger.Info(ctx, "Tx safe : %s", update.TxID)
	}

	client.pendingTxs = append(client.pendingTxs[:txIndex], client.pendingTxs[txIndex+1:]...)
	client.applyTx(ctx, tx, false)
}

func (client *Client) HandleHeaders(ctx context.Context, headers *client.Headers) {
	client.blockHeight = int(headers.StartHeight) + len(headers.Headers)
	client.blocksAdded += len(headers.Headers)
	if client.blocksAdded%100 == 0 {
		logger.Info(ctx, "Added 100 blocks to height %d", client.blockHeight)
	}
	if client.spyNodeInSync {
		logger.Info(ctx, "New header (%d) : %s", client.blockHeight, headers.Headers[0].BlockHash())
	}
}

func (client *Client) HandleInSync(ctx context.Context) {
	client.lock.Lock()
	defer client.lock.Unlock()

	ctx = logger.ContextWithOutLogSubSystem(ctx)

	// TODO Build/Send outgoing transactions
	for _, tx := range client.TxsToSend {
		client.spyNode.SendTx(ctx, tx)
	}
	client.TxsToSend = nil

	if client.blocksAdded == 0 {
		logger.Info(ctx, "No new blocks found")
	} else {
		logger.Info(ctx, "Synchronized %d new block(s) to height %d", client.blocksAdded,
			client.blockHeight)
	}
	logger.Info(ctx, "Balance : %.08f", BitcoinsFromSatoshis(client.Wallet.Balance()))

	// Trigger close
	if client.StopOnSync {
		client.spyNodeStopChannel <- nil
	}
	client.spyNodeInSync = true
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
