package listeners

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// TODO Handle scenario when txs are marked ready before proprocessing is complete. They still need
//   to be added to processing in the order that they were marked safe.

func (server *Server) AddTx(ctx context.Context, tx *wire.MsgTx) error {
	intx, err := NewIncomingTxData(ctx, tx)
	if err != nil {
		return err
	}

	server.pendingLock.Lock()

	_, exists := server.pendingTxs[intx.Itx.Hash]
	if exists {
		server.pendingLock.Unlock()
		return fmt.Errorf("Tx already added : %s", intx.Itx.Hash.String())
	}
	server.pendingTxs[intx.Itx.Hash] = intx
	server.pendingLock.Unlock()

	server.incomingTxs.Add(intx)
	return nil
}

// ProcessIncomingTxs performs preprocessing on transactions coming into the smart contract.
func (server *Server) ProcessIncomingTxs(ctx context.Context, masterDB *db.DB, contractPKHs []*protocol.PublicKeyHash,
	headers node.BitcoinHeaders) error {

	for intx := range server.incomingTxs.Channel {
		if err := intx.Itx.Setup(ctx, server.Config.IsTest); err != nil {
			return err
		}
		if err := intx.Itx.Validate(ctx); err != nil {
			return err
		}
		if err := intx.Itx.Promote(ctx, server.RpcNode); err != nil {
			return err
		}

		switch msg := intx.Itx.MsgProto.(type) {
		case *protocol.Transfer:
			if err := validateOracles(ctx, masterDB, contractPKHs, intx.Itx, msg, headers); err != nil {
				intx.Itx.RejectCode = protocol.RejectInvalidSignature
				node.LogWarn(ctx, "Invalid oracle signature : %s", err)
			}
		}

		server.markPreprocessed(ctx, &intx.Itx.Hash)
	}
	return nil
}

func (server *Server) markPreprocessed(ctx context.Context, txid *chainhash.Hash) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	intx.IsPreprocessed = true
	if intx.IsReady {
		server.processReadyTxs(ctx)
	}
}

// processReadyTxs moves txs from pending into the processing channel in the proper order.
func (server *Server) processReadyTxs(ctx context.Context) {
	toRemove := 0
	for _, txid := range server.readyTxs {
		intx, exists := server.pendingTxs[*txid]
		if !exists {
			toRemove++
		} else if intx.IsPreprocessed && intx.IsReady {
			server.processingTxs.Add(ProcessingTx{Itx: intx.Itx, Event: "SEE"})
			delete(server.pendingTxs, intx.Itx.Hash)
			toRemove++
		} else {
			break
		}
	}

	// Remove processed txids
	if toRemove > 0 {
		server.readyTxs = append(server.readyTxs[:toRemove-1], server.readyTxs[toRemove:]...)
	}
}

func (server *Server) MarkSafe(ctx context.Context, txid *chainhash.Hash) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	intx.IsReady = true
	if !intx.InReady {
		intx.InReady = true
		server.readyTxs = append(server.readyTxs, &intx.Itx.Hash)
	}
	server.processReadyTxs(ctx)
}

func (server *Server) MarkUnsafe(ctx context.Context, txid *chainhash.Hash) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	intx.IsReady = true
	intx.Itx.RejectCode = protocol.RejectDoubleSpend
	for i, readyID := range server.readyTxs {
		if *readyID == *txid {
			server.readyTxs = append(server.readyTxs[:i], server.readyTxs[i+1:]...)
			break
		}
	}
}

func (server *Server) CancelPendingTx(ctx context.Context, txid *chainhash.Hash) bool {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return false
	}

	delete(server.pendingTxs, *txid)
	if intx.InReady {
		for i, readyID := range server.readyTxs {
			if *readyID == *txid {
				server.readyTxs = append(server.readyTxs[:i], server.readyTxs[i+1:]...)
				break
			}
		}
	}
	return true
}

func (server *Server) MarkConfirmed(ctx context.Context, txid *chainhash.Hash) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	intx.IsReady = true
	if !intx.InReady {
		intx.InReady = true
		server.readyTxs = append(server.readyTxs, &intx.Itx.Hash)
	}
	server.processReadyTxs(ctx)
}

type IncomingTxData struct {
	Itx            *inspector.Transaction
	IsPreprocessed bool // Preprocessing has completed
	IsReady        bool // Is ready to be processed
	InReady        bool // In ready list
}

func NewIncomingTxData(ctx context.Context, tx *wire.MsgTx) (*IncomingTxData, error) {
	result := IncomingTxData{}
	var err error
	result.Itx, err = inspector.NewBaseTransactionFromWire(ctx, tx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type IncomingTxChannel struct {
	Channel chan *IncomingTxData
	lock    sync.Mutex
	open    bool
}

func (c *IncomingTxChannel) Add(tx *IncomingTxData) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- tx
	return nil
}

func (c *IncomingTxChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan *IncomingTxData, count)
	c.open = true
	return nil
}

func (c *IncomingTxChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}

// validateOracles verifies all oracle signatures related to this tx.
func validateOracles(ctx context.Context, masterDB *db.DB, contractPKHs []*protocol.PublicKeyHash,
	itx *inspector.Transaction, transfer *protocol.Transfer, headers node.BitcoinHeaders) error {

	cts := make([]*state.Contract, len(contractPKHs))
	var ct *state.Contract
	var err error

	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "CUR" && assetTransfer.AssetCode.IsZero() {
			continue // Skip bitcoin transfers since they should be handled already
		}

		if len(itx.Outputs) <= int(assetTransfer.ContractIndex) {
			continue
		}

		contractOutputPKH := protocol.PublicKeyHashFromBytes(itx.Outputs[assetTransfer.ContractIndex].Address.ScriptAddress())
		if contractOutputPKH == nil {
			continue // Invalid contract index
		}

		ct = nil
		for i, pkh := range contractPKHs {
			if bytes.Equal(contractOutputPKH.Bytes(), pkh.Bytes()) {
				ct = cts[i]
				if ct == nil {
					ct, err = contract.Retrieve(ctx, masterDB, contractOutputPKH)
					if err != nil {
						return err
					}
					cts[i] = ct
				}
			}
		}

		if ct == nil {
			continue // Not associated with one of our contracts
		}

		// Process receivers
		for _, receiver := range assetTransfer.AssetReceivers {
			// Check Oracle Signature
			if err := validateOracle(ctx, contractOutputPKH, ct, &assetTransfer.AssetCode, &receiver,
				headers); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateOracle(ctx context.Context, contractPKH *protocol.PublicKeyHash, ct *state.Contract,
	assetCode *protocol.AssetCode, assetReceiver *protocol.AssetReceiver, headers node.BitcoinHeaders) error {

	if assetReceiver.OracleSigAlgorithm == 0 {
		if len(ct.Oracles) > 0 {
			return fmt.Errorf("Missing signature")
		}
		return nil // No signature required
	}

	if int(assetReceiver.OracleIndex) >= len(ct.FullOracles) {
		return fmt.Errorf("Oracle index out of range : %d / %d", assetReceiver.OracleIndex,
			len(ct.FullOracles))
	}

	// Parse signature
	oracleSig, err := btcec.ParseSignature(assetReceiver.OracleConfirmationSig, btcec.S256())
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Check if block time is beyond expiration
	expire := (v.Now.Seconds()) - 3600 // Hour ago, unix timestamp in seconds
	blockTime, err := headers.Time(ctx, int(assetReceiver.OracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve time for block height %d",
			assetReceiver.OracleSigBlockHeight))
	}
	if blockTime < expire {
		return fmt.Errorf("Oracle sig block hash expired : %d < %d", blockTime, expire)
	}

	oracle := ct.FullOracles[assetReceiver.OracleIndex]
	hash, err := headers.Hash(ctx, int(assetReceiver.OracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
			assetReceiver.OracleSigBlockHeight))
	}
	node.LogVerbose(ctx, "Checking sig against oracle %d with block hash %d : %s",
		assetReceiver.OracleIndex, assetReceiver.OracleSigBlockHeight, hash.String())
	sigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, assetCode,
		&assetReceiver.Address, assetReceiver.Quantity, hash)
	if err != nil {
		return errors.Wrap(err, "Failed to calculate oracle sig hash")
	}

	if oracleSig.Verify(sigHash, oracle) {
		return nil // Valid signature found
	}

	return fmt.Errorf("Valid signature not found")
}
