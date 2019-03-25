package rpcnode

/**
 * RPC Node Kit
 *
 * What is my purpose?
 * - You connect to a bitcoind node
 * - You make RPC calls for me
 */

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
	SubSystem = "RPCNode" // For logger
)

type RPCNode struct {
	client      *rpcclient.Client
	txCache     map[chainhash.Hash]*wire.MsgTx
	chainParams *chaincfg.Params
}

// NewNode returns a new instance of an RPC node
func NewNode(config *Config) (*RPCNode, error) {
	rpcConfig := rpcclient.ConnConfig{
		HTTPPostMode: true,
		DisableTLS:   true,
		Host:         config.Host,
		User:         config.Username,
		Pass:         config.Password,
	}

	client, err := rpcclient.New(&rpcConfig, nil)
	if err != nil {
		return nil, err
	}

	n := &RPCNode{
		client:      client,
		txCache:     make(map[chainhash.Hash]*wire.MsgTx),
		chainParams: config.ChainParams,
	}

	return n, nil
}

// GetChainParams returns the bitcoin network parameters (mainnet/testnet)
func (r *RPCNode) GetChainParams() *chaincfg.Params {
	return r.chainParams
}

// GetTX requests a tx from the remote server.
func (r *RPCNode) GetTX(ctx context.Context, id *chainhash.Hash) (*wire.MsgTx, error) {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	defer logger.Elapsed(ctx, time.Now(), "GetTX")

	msg, ok := r.txCache[*id]
	if ok {
		logger.Verbose(ctx, "Using tx from RPC cache : %s\n", id.String())
		delete(r.txCache, *id)
		return msg, nil
	}

	logger.Verbose(ctx, "Requesting tx from RPC : %s\n", id.String())
	raw, err := r.client.GetRawTransactionVerbose(id)
	if err != nil {
		return nil, err
	}

	b, err := hex.DecodeString(raw.Hex)
	if err != nil {
		return nil, err
	}

	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		return nil, err
	}

	return &tx, nil
}

// WatchAddress instructs the RPC node to watch an address without rescan
func (r *RPCNode) WatchAddress(ctx context.Context, address btcutil.Address) error {
	strAddr := address.String()

	// Make address known to node without rescan
	if err := r.client.ImportAddressRescan(strAddr, strAddr, false); err != nil {
		return err
	}

	return nil
}

// ListTransactions returns all transactions for watched addresses
func (r *RPCNode) ListTransactions(ctx context.Context) ([]btcjson.ListTransactionsResult, error) {

	// Prepare listtransactions command
	cmd := btcjson.NewListTransactionsCmd(
		btcjson.String("*"),
		btcjson.Int(99999),
		btcjson.Int(0),
		btcjson.Bool(true))

	id := r.client.NextID()
	marshalledJSON, err := btcjson.MarshalCmd(id, cmd)
	if err != nil {
		return nil, err
	}

	// Unmarhsal in to a request to extract the params
	var request btcjson.Request
	if err = json.Unmarshal(marshalledJSON, &request); err != nil {
		return nil, err
	}

	// Submit raw request
	out, err := r.client.RawRequest("listtransactions", request.Params)
	if err != nil {
		return nil, err
	}

	// Unmarshal response in to a ListTransactionsResult
	var response []btcjson.ListTransactionsResult

	if err = json.Unmarshal(out, &response); err != nil {
		return nil, err
	}

	return response, nil
}

// ListUnspent returns unspent transactions
func (r *RPCNode) ListUnspent(ctx context.Context, address btcutil.Address) ([]btcjson.ListUnspentResult, error) {

	// Make address known to node without rescan
	if err := r.WatchAddress(ctx, address); err != nil {
		return nil, err
	}

	addresses := []btcutil.Address{address}

	// out []btcjson.ListUnspentResult
	out, err := r.client.ListUnspentMinMaxAddresses(0, 999999, addresses)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// SendRawTransaction broadcasts a raw transaction
func (r *RPCNode) SendRawTransaction(ctx context.Context,
	tx *wire.MsgTx) error {

	nx, err := r.txToBtcdTX(tx)
	if err != nil {
		return err
	}

	_, err = r.client.SendRawTransaction(nx, false)

	return err
}

// SaveTX saves a tx to be used later.
func (r *RPCNode) SaveTX(ctx context.Context, msg *wire.MsgTx) error {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	logger.Verbose(ctx, "Saving tx to rpc cache : %s\n", msg.TxHash().String())
	r.txCache[msg.TxHash()] = msg
	return nil
}

// SendTX sends a tx to the remote server to be broadcast to the P2P network.
func (r *RPCNode) SendTX(ctx context.Context,
	tx *wire.MsgTx) (*chainhash.Hash, error) {

	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	defer logger.Elapsed(ctx, time.Now(), "SendTX")

	nx, err := r.txToBtcdTX(tx)
	if err != nil {
		return nil, err
	}

	logger.Debug(ctx, "Sending tx payload : %s", r.getRawPayload(nx))

	return r.client.SendRawTransaction(nx, false)
}

func (r *RPCNode) GetLatestBlock() (*chainhash.Hash, int32, error) {
	// Get the best block hash
	hash, err := r.client.GetBestBlockHash()
	if err != nil {
		return nil, -1, err
	}

	// The height is in the header
	header, err := r.client.GetBlockHeaderVerbose(hash)
	if err != nil {
		return nil, -1, err
	}

	return hash, header.Height, nil
}

func (r *RPCNode) getRawPayload(tx *btcwire.MsgTx) string {
	var buf bytes.Buffer
	tx.Serialize(&buf)

	out := fmt.Sprintf("%#x", buf.Bytes())
	s := fmt.Sprintf("%s", strings.Replace(out, "0x", "", 1))

	return s
}

// txToBtcdTx converts a "pkg/wire".MsgTx to a "btcsuite/btcd/wire".MsgTx".
func (r *RPCNode) txToBtcdTX(tx *wire.MsgTx) (*btcwire.MsgTx, error) {
	// Read the payload from the input TX, into the output TX.
	var buf bytes.Buffer
	tx.Serialize(&buf)

	reader := bytes.NewReader(buf.Bytes())

	nx := &btcwire.MsgTx{}

	if err := nx.Deserialize(reader); err != nil {
		return nil, err
	}

	return nx, nil
}
