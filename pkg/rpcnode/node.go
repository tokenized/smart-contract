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
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
	// SubSystem is used by the logger package
	SubSystem = "RPCNode"
)

type RPCNode struct {
	client  *rpcclient.Client
	txCache map[bitcoin.Hash32]*wire.MsgTx
	lock    sync.Mutex
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
		client:  client,
		txCache: make(map[bitcoin.Hash32]*wire.MsgTx),
	}

	return n, nil
}

// GetTX requests a tx from the remote server.
func (r *RPCNode) GetTX(ctx context.Context, id *bitcoin.Hash32) (*wire.MsgTx, error) {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	defer logger.Elapsed(ctx, time.Now(), "GetTX")

	r.lock.Lock()
	msg, ok := r.txCache[*id]
	if ok {
		logger.Verbose(ctx, "Using tx from RPC cache : %s\n", id.String())
		delete(r.txCache, *id)
		r.lock.Unlock()
		return msg, nil
	}
	r.lock.Unlock()

	logger.Verbose(ctx, "Requesting tx from RPC : %s\n", id.String())
	ch, _ := chainhash.NewHash(id[:])
	raw, err := r.client.GetRawTransactionVerbose(ch)
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

// GetTXs requests a list of txs from the remote server.
func (r *RPCNode) GetTXs(ctx context.Context, txids []*bitcoin.Hash32) ([]*wire.MsgTx, error) {
	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	defer logger.Elapsed(ctx, time.Now(), "GetTXs")

	results := make([]*wire.MsgTx, len(txids))
	requests := make([]*rpcclient.FutureGetRawTransactionVerboseResult, len(txids))

	r.lock.Lock()
	for i, txid := range txids {
		msg, ok := r.txCache[*txid]
		if ok {
			logger.Verbose(ctx, "Using tx from RPC cache : %s\n", txid.String())
			delete(r.txCache, *txid)
			results[i] = msg
		} else {
			logger.Verbose(ctx, "Requesting tx from RPC : %s\n", txid.String())
			ch, _ := chainhash.NewHash(txid[:])
			request := r.client.GetRawTransactionVerboseAsync(ch)
			requests[i] = &request
		}
	}
	r.lock.Unlock()

	for i, request := range requests {
		if request == nil {
			continue
		}
		rawTx, err := request.Receive()
		if err != nil {
			return results, err // TODO Determine when error is tx not seen yet
		}

		b, err := hex.DecodeString(rawTx.Hex)
		if err != nil {
			return results, err
		}

		tx := wire.MsgTx{}
		buf := bytes.NewReader(b)

		if err := tx.Deserialize(buf); err != nil {
			return results, err
		}

		results[i] = &tx
	}

	return results, nil
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
func (r *RPCNode) SendRawTransaction(ctx context.Context, tx *wire.MsgTx) error {

	nx, err := r.txToBtcdTX(tx)
	if err != nil {
		return err
	}

	_, err = r.client.SendRawTransaction(nx, false)

	return err
}

// SaveTX saves a tx to be used later.
func (r *RPCNode) SaveTX(ctx context.Context, msg *wire.MsgTx) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	hash := msg.TxHash()
	logger.Verbose(ctx, "Saving tx to rpc cache : %s\n", hash.String())
	r.txCache[*hash] = msg
	return nil
}

// SendTX sends a tx to the remote server to be broadcast to the P2P network.
func (r *RPCNode) SendTX(ctx context.Context, tx *wire.MsgTx) (*bitcoin.Hash32, error) {

	ctx = logger.ContextWithLogSubSystem(ctx, SubSystem)
	defer logger.Elapsed(ctx, time.Now(), "SendTX")

	nx, err := r.txToBtcdTX(tx)
	if err != nil {
		return nil, err
	}

	logger.Debug(ctx, "Sending tx payload : %s", r.getRawPayload(nx))

	ch, err := r.client.SendRawTransaction(nx, false)
	if err != nil {
		return nil, err
	}
	return bitcoin.NewHash32(ch[:])
}

func (r *RPCNode) GetLatestBlock() (*bitcoin.Hash32, int32, error) {
	// Get the best block hash
	hash, err := r.client.GetBestBlockHash()
	if err != nil {
		return nil, -1, err
	}

	bhash, err := bitcoin.NewHash32(hash[:])
	if err != nil {
		return nil, -1, err
	}

	// The height is in the header
	header, err := r.client.GetBlockHeaderVerbose(hash)
	if err != nil {
		return nil, -1, err
	}

	return bhash, header.Height, nil
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
