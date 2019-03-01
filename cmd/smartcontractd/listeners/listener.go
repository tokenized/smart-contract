package listeners

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Handle implements the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (server *Server) Handle(ctx context.Context, msgType int, msgValue interface{}) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	switch msgType {
	case handlers.ListenerMsgTx:
		tx, ok := msgValue.(wire.MsgTx)
		if !ok {
			return errors.New("Could not assert msgValue as wire.MsgTx")
		}
		logger.Log(ctx, logger.Info, "Tx : %s", tx.TxHash().String())

		// Check if transaction relates to protocol
		itx, err := inspector.NewTransactionFromWire(ctx, &tx)
		if err != nil {
			logger.Log(ctx, logger.Warn, "Failed to create inspector tx : %s", err.Error())
			return err
		}

		// Prefilter out non-protocol messages
		if !itx.IsTokenized() {
			logger.Log(ctx, logger.Verbose, "Not tokenized tx : %s", tx.TxHash().String())
			return nil
		}

		// Promote TX
		if err := itx.Promote(ctx, server.RpcNode); err != nil {
			logger.Log(ctx, logger.Fatal, "Failed to promote inspector tx : %s", err.Error())
			return err
		}

		if err := server.removeConflictingPending(ctx, itx); err != nil {
			logger.Log(ctx, logger.Warn, "Failed to remove conflicting pending : %s", err.Error())
			return err
		}

		return server.Handler.Trigger(ctx, protomux.SEE, itx)

	case handlers.ListenerMsgTxConfirm:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			return errors.New("Could not assert msgValue as chainhash.Hash")
		}
		logger.Log(ctx, logger.Info, "Tx confirm : %s", txHash.String())
		// TODO Mark as confirmed

	case handlers.ListenerMsgBlock:
		blockData, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			return errors.New("Could not assert msgValue as BlockMessage")
		}
		logger.Log(ctx, logger.Info, "New Block (%d) : %s", blockData.Height, blockData.Hash.String())
		// Remember current block height for confirm messages
		server.blockHeight = blockData.Height

		// TODO Mark in contract so confirm counts are known

	case handlers.ListenerMsgBlockRevert:
		blockData, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			return errors.New("Could not assert msgValue as BlockMessage")
		}
		logger.Log(ctx, logger.Info, "Revert Block (%d) : %s", blockData.Height, blockData.Hash.String())
		// Remember current block height for confirm messages
		server.blockHeight = blockData.Height - 1

		// TODO Mark in contract so confirm counts are known

	case handlers.ListenerMsgTxRevert:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			return errors.New("Could not assert msgValue as chainhash.Hash")
		}
		logger.Log(ctx, logger.Info, "Tx revert : %s", txHash.String())

	case handlers.ListenerMsgTxCancel:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			return errors.New("Could not assert msgValue as chainhash.Hash")
		}
		logger.Log(ctx, logger.Info, "Tx cancel : %s", txHash.String())
		// TODO Undo anything related to this transaction as a conflicting tx has been mined, so it
		//   will never confirm (without a reorg).

	case handlers.ListenerMsgTxUnsafe:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			return errors.New("Could not assert msgValue as chainhash.Hash")
		}
		// TODO Pause anything related to this transaction as a conflicting tx has been seen, so it
		//   might never confirm.
		logger.Log(ctx, logger.Info, "Tx unsafe : %s", txHash.String())

	case handlers.ListenerMsgInSync:
		logger.Log(ctx, logger.Info, "Node is in sync")

		// Send pending responses
		server.inSync = true
		pending := server.pendingResponses
		server.pendingResponses = nil

		for _, pendingTx := range pending {
			logger.Log(ctx, logger.Info, "Sending pending response: %s", pendingTx.TxHash().String())
			if err := server.sendTx(ctx, pendingTx); err != nil {
				return err // TODO Probably a fatal error
			}
		}
	}

	return nil
}
