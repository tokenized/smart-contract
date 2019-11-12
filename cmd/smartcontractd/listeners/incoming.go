package listeners

import (
	"context"
	"fmt"
	"sync"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
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

	_, exists := server.pendingTxs[*intx.Itx.Hash]
	if exists {
		server.pendingLock.Unlock()
		return fmt.Errorf("Tx already added : %s", intx.Itx.Hash.String())
	}
	server.pendingTxs[*intx.Itx.Hash] = intx
	server.pendingLock.Unlock()

	server.incomingTxs.Add(intx)
	return nil
}

// ProcessIncomingTxs performs preprocessing on transactions coming into the smart contract.
func (server *Server) ProcessIncomingTxs(ctx context.Context, masterDB *db.DB,
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
		case *actions.Transfer:
			if err := validateOracles(ctx, masterDB, intx.Itx, msg, headers); err != nil {
				intx.Itx.RejectCode = actions.RejectionsInvalidSignature
				node.LogWarn(ctx, "Invalid receiver oracle signature : %s", err)
			}
		case *actions.ContractOffer:
			if err := validateContractOracleSig(ctx, intx.Itx, msg, headers); err != nil {
				intx.Itx.RejectCode = actions.RejectionsInvalidSignature
				node.LogWarn(ctx, "Invalid contract oracle signature : %s", err)
			}
		}

		server.markPreprocessed(ctx, intx.Itx.Hash)
	}
	return nil
}

func (server *Server) markPreprocessed(ctx context.Context, txid *bitcoin.Hash32) {
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
			delete(server.pendingTxs, *intx.Itx.Hash)
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

func (server *Server) MarkSafe(ctx context.Context, txid *bitcoin.Hash32) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	// Broadcast to ensure it is accepted by the network.
	if err := server.sendTx(ctx, intx.Itx.MsgTx); err != nil {
		node.LogWarn(ctx, "Failed to re-broadcast safe incoming : %s", err)
	}

	intx.IsReady = true
	if !intx.InReady {
		intx.InReady = true
		server.readyTxs = append(server.readyTxs, intx.Itx.Hash)
	}
	server.processReadyTxs(ctx)
}

func (server *Server) MarkUnsafe(ctx context.Context, txid *bitcoin.Hash32) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	intx.IsReady = true
	intx.Itx.RejectCode = actions.RejectionsDoubleSpend
	for i, readyID := range server.readyTxs {
		if *readyID == *txid {
			server.readyTxs = append(server.readyTxs[:i], server.readyTxs[i+1:]...)
			break
		}
	}
}

func (server *Server) CancelPendingTx(ctx context.Context, txid *bitcoin.Hash32) bool {
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

func (server *Server) MarkConfirmed(ctx context.Context, txid *bitcoin.Hash32) {
	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		return
	}

	// Broadcast to ensure it is accepted by the network.
	if err := server.sendTx(ctx, intx.Itx.MsgTx); err != nil {
		node.LogWarn(ctx, "Failed to re-broadcast confirmed incoming : %s", err)
	}

	intx.IsReady = true
	if !intx.InReady {
		intx.InReady = true
		server.readyTxs = append(server.readyTxs, intx.Itx.Hash)
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
func validateOracles(ctx context.Context, masterDB *db.DB, itx *inspector.Transaction,
	transfer *actions.Transfer, headers node.BitcoinHeaders) error {

	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			continue // Skip bitcoin transfers since they should be handled already
		}

		if len(itx.Outputs) <= int(assetTransfer.ContractIndex) {
			continue
		}

		ct, err := contract.Retrieve(ctx, masterDB, itx.Outputs[assetTransfer.ContractIndex].Address)
		if err == contract.ErrNotFound {
			continue // Not associated with one of our contracts
		}
		if err != nil {
			return err
		}

		// Process receivers
		for _, receiver := range assetTransfer.AssetReceivers {
			// Check Oracle Signature
			if err := validateOracle(ctx, itx.Outputs[assetTransfer.ContractIndex].Address,
				ct, assetTransfer.AssetCode, receiver, headers); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateOracle(ctx context.Context, contractAddress bitcoin.RawAddress, ct *state.Contract,
	assetCode []byte, assetReceiver *actions.AssetReceiverField, headers node.BitcoinHeaders) error {

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
	oracleSig, err := bitcoin.DecodeSignatureBytes(assetReceiver.OracleConfirmationSig)
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

	receiverAddress, err := bitcoin.DecodeRawAddress(assetReceiver.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to read receiver address")
	}

	node.LogVerbose(ctx, "Checking sig against oracle %d with block hash %d : %s",
		assetReceiver.OracleIndex, assetReceiver.OracleSigBlockHeight, hash.String())
	sigHash, err := protocol.TransferOracleSigHash(ctx, contractAddress, assetCode,
		receiverAddress, assetReceiver.Quantity, hash, 1)
	if err != nil {
		return errors.Wrap(err, "Failed to calculate oracle sig hash")
	}

	if oracleSig.Verify(sigHash, oracle) {
		return nil // Valid signature found
	}

	return fmt.Errorf("Valid signature not found")
}

func validateContractOracleSig(ctx context.Context, itx *inspector.Transaction,
	contractOffer *actions.ContractOffer, headers node.BitcoinHeaders) error {
	if contractOffer.AdminOracle == nil {
		return nil
	}

	oracle, err := bitcoin.DecodePublicKeyBytes(contractOffer.AdminOracle.PublicKey)
	if err != nil {
		return err
	}

	// Parse signature
	oracleSig, err := bitcoin.DecodeSignatureBytes(contractOffer.AdminOracleSignature)
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	hash, err := headers.Hash(ctx, int(contractOffer.AdminOracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
			contractOffer.AdminOracleSigBlockHeight))
	}

	addresses := make([]bitcoin.RawAddress, 0, 2)
	entities := make([]*actions.EntityField, 0, 2)

	addresses = append(addresses, itx.Inputs[0].Address)
	entities = append(entities, contractOffer.Issuer)

	if contractOffer.ContractOperator != nil {
		addresses = append(addresses, itx.Inputs[1].Address)
		entities = append(entities, contractOffer.ContractOperator)
	}

	sigHash, err := protocol.ContractOracleSigHash(ctx, addresses, entities, hash, 1)
	if err != nil {
		return err
	}

	if oracleSig.Verify(sigHash, oracle) {
		return nil // Valid signature found
	}

	return fmt.Errorf("Contract signature invalid")
}
