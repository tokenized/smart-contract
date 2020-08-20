package listeners

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/inspector"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// AddTx adds a tx to the incoming pipeline.
func (server *Server) AddTx(ctx context.Context, tx *wire.MsgTx, txid bitcoin.Hash32) error {

	server.pendingLock.Lock()
	// Copy tx within lock to ensure that there is no possibility of a race condition with the
	//   original copy of the tx in the current thread.
	intx, err := NewIncomingTxData(ctx, tx, txid)
	if err != nil {
		server.pendingLock.Unlock()
		return err
	}

	_, exists := server.pendingTxs[*intx.Itx.Hash]
	if exists {
		server.pendingLock.Unlock()
		return fmt.Errorf("Tx already added : %s", intx.Itx.Hash.String())
	}
	server.pendingTxs[*intx.Itx.Hash] = intx
	server.pendingLock.Unlock()

	server.incomingTxs.Add(intx)

	node.LogVerbose(ctx, "Tx added to incoming : %s", intx.Itx.Hash.String())
	return nil
}

// ProcessIncomingTxs performs preprocessing on transactions coming into the smart contract.
func (server *Server) ProcessIncomingTxs(ctx context.Context, masterDB *db.DB,
	headers node.BitcoinHeaders) error {

	for intx := range server.incomingTxs.Channel {
		node.LogVerbose(ctx, "Processing incoming tx : %s", intx.Itx.Hash.String())

		if err := intx.Itx.Setup(ctx, server.Config.IsTest); err != nil {
			return err
		}
		if err := intx.Itx.Validate(ctx); err != nil {
			return err
		}
		if err := intx.Itx.Promote(ctx, server.RpcNode); err != nil {
			return err
		}

		if server.Config.MinFeeRate > 0.0 && intx.Itx.IsIncomingMessageType() {
			feeRate, err := intx.Itx.FeeRate()
			if err != nil {
				return errors.Wrap(err, "fee rate")
			}
			if feeRate < server.Config.MinFeeRate {
				intx.Itx.RejectCode = actions.RejectionsInsufficientTxFeeFunding
				node.LogWarn(ctx, "Low tx fee rate %f : %s", feeRate, intx.Itx.Hash.String())
			}
		}

		switch msg := intx.Itx.MsgProto.(type) {
		case *actions.Transfer:
			if err := validateOracles(ctx, masterDB, intx.Itx, msg, headers, server.Config.IsTest); err != nil {
				intx.Itx.RejectCode = actions.RejectionsInvalidSignature
				node.LogWarn(ctx, "Invalid receiver oracle signature : %s", err)
			}
		case *actions.ContractOffer:
			if err := validateContractOracleSig(ctx, masterDB, server.Config, intx.Itx, msg, headers,
				intx.Timestamp); err != nil {
				intx.Itx.RejectCode = actions.RejectionsInvalidSignature
				node.LogWarn(ctx, "Invalid contract oracle signature : %s", err)
			}
		}

		server.markPreprocessed(ctx, intx.Itx.Hash)
	}
	return nil
}

func (server *Server) markPreprocessed(ctx context.Context, txid *bitcoin.Hash32) {
	node.LogVerbose(ctx, "Marking tx preprocessed : %s", txid.String())
	defer node.LogVerbose(ctx, "Tx marked preprocessed : %s", txid.String())

	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		node.LogVerbose(ctx, "Pending tx doesn't exist for preprocessed : %s", txid.String())
		return
	}

	intx.IsPreprocessed = true
	if intx.IsReady {
		server.processReadyTxs(ctx)
	}
}

// processReadyTxs moves txs from pending into the processing channel in the proper order.
// pendingLock is locked by caller.
func (server *Server) processReadyTxs(ctx context.Context) {
	node.LogVerbose(ctx, "Process ready txs")
	defer node.LogVerbose(ctx, "Processed ready txs")

	toRemove := 0
	for _, txid := range server.readyTxs {
		node.LogVerbose(ctx, "Checking ready tx : %s", txid.String())

		intx, exists := server.pendingTxs[*txid]
		if !exists {
			node.LogVerbose(ctx, "Tx not pending : %s", txid.String())
			toRemove++
			continue
		}

		node.LogVerbose(ctx, "IsPreprocessed : %t", intx.IsPreprocessed)
		node.LogVerbose(ctx, "IsReady : %t", intx.IsReady)

		if intx.IsPreprocessed && intx.IsReady {
			node.LogVerbose(ctx, "Pending tx added to processing : %s", txid.String())
			server.processingTxs.Add(ProcessingTx{Itx: intx.Itx, Event: "SEE"})
			delete(server.pendingTxs, *intx.Itx.Hash)
			toRemove++
			continue
		}

		node.LogVerbose(ctx, "Tx not ready : %s", txid.String())
		break
	}

	// Remove processed txids
	if toRemove > 0 {
		node.LogVerbose(ctx, "Removing %d txs from ready", toRemove)
		for i := 0; i < toRemove; i++ {
			node.LogVerbose(ctx, "Removing tx from ready : %s", server.readyTxs[i].String())
		}
		server.readyTxs = append(server.readyTxs[:toRemove-1], server.readyTxs[toRemove:]...)
	}
}

func (server *Server) MarkSafe(ctx context.Context, txid *bitcoin.Hash32) {
	node.LogVerbose(ctx, "Marking tx safe : %s", txid.String())
	defer node.LogVerbose(ctx, "Tx marked safe : %s", txid.String())

	server.pendingLock.Lock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		node.LogVerbose(ctx, "Pending tx doesn't exist for safe : %s", txid.String())
		server.pendingLock.Unlock()
		return
	}

	intx.IsReady = true
	if !intx.InReady {
		node.LogVerbose(ctx, "Adding tx to ready : %s", txid.String())
		intx.InReady = true
		server.readyTxs = append(server.readyTxs, intx.Itx.Hash)
	}
	server.processReadyTxs(ctx)

	server.pendingLock.Unlock()

	// Broadcast to ensure it is accepted by the network.
	if server.IsInSync() && intx.Itx.IsIncomingMessageType() {
		if err := server.sendTx(ctx, intx.Itx.MsgTx); err != nil {
			node.LogWarn(ctx, "Failed to re-broadcast safe incoming : %s", err)
		}
	}
}

func (server *Server) MarkUnsafe(ctx context.Context, txid *bitcoin.Hash32) {
	node.LogVerbose(ctx, "Marking tx unsafe : %s", txid.String())
	defer node.LogVerbose(ctx, "Tx marked unsafe : %s", txid.String())

	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		node.LogVerbose(ctx, "Pending tx doesn't exist for unsafe : %s", txid.String())
		return
	}

	intx.IsReady = true
	intx.Itx.RejectCode = actions.RejectionsDoubleSpend
	for i, readyID := range server.readyTxs {
		if *readyID == *txid {
			node.LogVerbose(ctx, "Removing tx from ready : %s", txid.String())
			server.readyTxs = append(server.readyTxs[:i], server.readyTxs[i+1:]...)
			break
		}
	}
}

func (server *Server) CancelPendingTx(ctx context.Context, txid *bitcoin.Hash32) bool {
	node.LogVerbose(ctx, "Canceling pending tx : %s", txid.String())
	defer node.LogVerbose(ctx, "Cancelled pending tx : %s", txid.String())

	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		node.LogVerbose(ctx, "Pending tx doesn't exist for cancel : %s", txid.String())
		return false
	}

	delete(server.pendingTxs, *txid)
	if intx.InReady {
		for i, readyID := range server.readyTxs {
			if *readyID == *txid {
				node.LogVerbose(ctx, "Removing tx from ready : %s", txid.String())
				server.readyTxs = append(server.readyTxs[:i], server.readyTxs[i+1:]...)
				break
			}
		}
	}
	return true
}

func (server *Server) MarkConfirmed(ctx context.Context, txid *bitcoin.Hash32) {
	node.LogVerbose(ctx, "Marking tx confirmed : %s", txid.String())
	defer node.LogVerbose(ctx, "Tx marked confirmed : %s", txid.String())

	server.pendingLock.Lock()
	defer server.pendingLock.Unlock()

	intx, exists := server.pendingTxs[*txid]
	if !exists {
		node.LogVerbose(ctx, "Pending tx doesn't exist for confirmed : %s", txid.String())
		return
	}

	intx.IsReady = true
	if !intx.InReady {
		node.LogVerbose(ctx, "Adding tx to ready : %s", txid.String())
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
	Timestamp      protocol.Timestamp
}

func (itd *IncomingTxData) Serialize(buf *bytes.Buffer) error {
	if err := itd.Itx.Write(buf); err != nil {
		return errors.Wrap(err, "write itx")
	}

	if err := binary.Write(buf, DefaultEndian, itd.IsPreprocessed); err != nil {
		return errors.Wrap(err, "write is preprocessed")
	}

	if err := binary.Write(buf, DefaultEndian, itd.IsReady); err != nil {
		return errors.Wrap(err, "write is ready")
	}

	if err := binary.Write(buf, DefaultEndian, itd.InReady); err != nil {
		return errors.Wrap(err, "write in ready")
	}

	if err := itd.Timestamp.Serialize(buf); err != nil {
		return errors.Wrap(err, "write timestamp")
	}

	return nil
}

func (itd *IncomingTxData) Deserialize(buf *bytes.Reader, isTest bool) error {
	itd.Itx = &inspector.Transaction{}
	if err := itd.Itx.Read(buf, isTest); err != nil {
		return errors.Wrap(err, "read itx")
	}

	if err := binary.Read(buf, DefaultEndian, &itd.IsPreprocessed); err != nil {
		return errors.Wrap(err, "read is preprocessed")
	}

	if err := binary.Read(buf, DefaultEndian, &itd.IsReady); err != nil {
		return errors.Wrap(err, "read is ready")
	}

	if err := binary.Read(buf, DefaultEndian, &itd.InReady); err != nil {
		return errors.Wrap(err, "read in ready")
	}

	var err error
	itd.Timestamp, err = protocol.DeserializeTimestamp(buf)
	if err != nil {
		return errors.Wrap(err, "read timestamp")
	}

	return nil
}

func NewIncomingTxData(ctx context.Context, tx *wire.MsgTx, txid bitcoin.Hash32) (*IncomingTxData, error) {
	result := IncomingTxData{
		Timestamp:      protocol.CurrentTimestamp(),
		IsPreprocessed: false,
		IsReady:        false,
		InReady:        false,
	}
	var err error
	result.Itx, err = inspector.NewBaseTransactionFromHashWire(ctx, &txid, tx)
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
	transfer *actions.Transfer, headers node.BitcoinHeaders, isTest bool) error {

	for _, assetTransfer := range transfer.Assets {
		if assetTransfer.AssetType == "BSV" && len(assetTransfer.AssetCode) == 0 {
			continue // Skip bitcoin transfers since they should be handled already
		}

		if len(itx.Outputs) <= int(assetTransfer.ContractIndex) {
			continue
		}

		ct, err := contract.Retrieve(ctx, masterDB,
			itx.Outputs[assetTransfer.ContractIndex].Address, isTest)
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
		identityFound := false
		for _, oracle := range ct.Oracles {
			if len(oracle.OracleTypes) == 0 {
				identityFound = true
				break
			}
			for _, t := range oracle.OracleTypes {
				if t == actions.ServiceTypeIdentityOracle {
					identityFound = true
					break
				}
			}
			if identityFound {
				break
			}
		}
		if identityFound { // Signature required when an identity oracle is specified.
			return errors.New("Missing signature")
		}
		return nil // No signature required
	}

	if int(assetReceiver.OracleIndex) >= len(ct.FullOracles) {
		return fmt.Errorf("Oracle index out of range : %d / %d", assetReceiver.OracleIndex,
			len(ct.FullOracles))
	}

	// No oracle types specified is assumed to be identity oracle for backwards compatibility
	if len(ct.Oracles[assetReceiver.OracleIndex].OracleTypes) != 0 {
		identityFound := false
		for _, t := range ct.Oracles[assetReceiver.OracleIndex].OracleTypes {
			if t == actions.ServiceTypeIdentityOracle {
				identityFound = true
				break
			}
		}
		if !identityFound {
			return errors.New("Oracle is not an identity oracle")
		}
	}

	// Parse signature
	oracleSig, err := bitcoin.SignatureFromBytes(assetReceiver.OracleConfirmationSig)
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	oracle := ct.FullOracles[assetReceiver.OracleIndex]
	hash, err := headers.Hash(ctx, int(assetReceiver.OracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
			assetReceiver.OracleSigBlockHeight))
	}

	if assetReceiver.OracleSigExpiry != 0 {
		now := uint64(time.Now().UnixNano())
		if now > assetReceiver.OracleSigExpiry {
			return fmt.Errorf("Oracle sigature expired : %d > %d", now, assetReceiver.OracleSigExpiry)
		}
	}

	receiverAddress, err := bitcoin.DecodeRawAddress(assetReceiver.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to read receiver address")
	}

	node.LogVerbose(ctx, "Checking sig against oracle %d with block hash %d : %s",
		assetReceiver.OracleIndex, assetReceiver.OracleSigBlockHeight, hash.String())
	sigHash, err := protocol.TransferOracleSigHash(ctx, contractAddress, assetCode,
		receiverAddress, hash, assetReceiver.OracleSigExpiry, 1)
	if err != nil {
		return errors.Wrap(err, "Failed to calculate oracle sig hash")
	}

	if oracleSig.Verify(sigHash, oracle.PublicKey) {
		node.Log(ctx, "Receiver oracle signature is valid")
		return nil // Valid signature found
	}

	return fmt.Errorf("Valid signature not found")
}

func validateContractOracleSig(ctx context.Context, dbConn *db.DB, config *node.Config,
	itx *inspector.Transaction, contractOffer *actions.ContractOffer, headers node.BitcoinHeaders,
	ts protocol.Timestamp) error {
	if len(contractOffer.AdminIdentityCertificates) == 0 {
		return nil
	}

	for _, cert := range contractOffer.AdminIdentityCertificates {
		if cert.Expiration != 0 && cert.Expiration < ts.Nano() {
			return errors.New("Expired Admin Identity Certificate")
		}

		ra, err := bitcoin.DecodeRawAddress(cert.EntityContract)
		if err != nil {
			return errors.Wrap(err, "entity address")
		}

		cf, err := contract.FetchContractFormation(ctx, dbConn, ra, config.IsTest)
		if err != nil {
			return errors.Wrap(err, "fetch entity")
		}

		publicKey, err := contract.GetIdentityOracleKey(cf)
		if err != nil {
			return errors.Wrap(err, "get identity oracle")
		}

		// Parse signature
		oracleSig, err := bitcoin.SignatureFromBytes(cert.Signature)
		if err != nil {
			return errors.Wrap(err, "Failed to parse oracle signature")
		}

		// Check if block time is beyond expiration
		// TODO Figure out how to get tx time to here. node.KeyValues is not set in context.
		expire := uint32((time.Now().Unix())) - 21600 // 6 hours ago, unix timestamp in seconds
		blockTime, err := headers.Time(ctx, int(cert.BlockHeight))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to retrieve time for block height %d",
				cert.BlockHeight))
		}
		if blockTime < expire {
			return fmt.Errorf("Oracle sig block hash expired : %d < %d", blockTime, expire)
		}

		hash, err := headers.Hash(ctx, int(cert.BlockHeight))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
				cert.BlockHeight))
		}

		var entity interface{}
		if len(contractOffer.EntityContract) > 0 {
			// Use parent entity contract address in signature instead of entity structure.
			entityRA, err := bitcoin.DecodeRawAddress(contractOffer.EntityContract)
			if err != nil {
				return errors.Wrap(err, "entity address")
			}

			entity = entityRA
		} else {
			entity = contractOffer.Issuer
		}

		sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, itx.Inputs[0].Address,
			entity, hash, 1)
		if err != nil {
			return err
		}

		if !oracleSig.Verify(sigHash, publicKey) {
			return fmt.Errorf("Contract signature invalid")
		}

		node.Log(ctx, "Contract oracle signature is valid")
	}

	return nil
}
