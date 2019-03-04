package listeners

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Implement the SpyNode Listener interface.

func (server *Server) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgBlock:
		logger.Log(ctx, logger.Info, "New Block (%d) : %s", block.Height, block.Hash.String())
	case handlers.ListenerMsgBlockRevert:
		logger.Log(ctx, logger.Info, "Reverted Block (%d) : %s", block.Height, block.Hash.String())
	}
	return nil
}

func (server *Server) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	logger.Log(ctx, logger.Info, "Tx : %s", tx.TxHash().String())

	// Check if transaction relates to protocol
	itx, err := inspector.NewTransactionFromWire(ctx, tx)
	if err != nil {
		logger.Log(ctx, logger.Warn, "Failed to create inspector tx : %s", err.Error())
		return false, err
	}

	// Prefilter out non-protocol messages
	if !itx.IsTokenized() {
		logger.Log(ctx, logger.Verbose, "Not tokenized tx : %s", tx.TxHash().String())
		return false, nil
	}

	// Promote TX
	if err := itx.Promote(ctx, server.RpcNode); err != nil {
		logger.Log(ctx, logger.Fatal, "Failed to promote inspector tx : %s", err.Error())
		return false, err
	}

	if err := server.removeConflictingPending(ctx, itx); err != nil {
		logger.Log(ctx, logger.Warn, "Failed to remove conflicting pending : %s", err.Error())
		return false, err
	}

	// Save tx to cache so it can be used to process the response
	if bytes.Equal(itx.Outputs[0].Address.ScriptAddress(), server.contractPKH) {
		if err := server.RpcNode.AddTX(ctx, itx.MsgTx); err != nil {
			return true, err
		}
	}

	return true, server.Handler.Trigger(ctx, protomux.SEE, itx)
}

func (server *Server) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgTxStateConfirm:
		logger.Log(ctx, logger.Info, "Tx confirm : %s", txid.String())
	case handlers.ListenerMsgTxStateRevert:
		logger.Log(ctx, logger.Info, "Tx revert : %s", txid.String())
	case handlers.ListenerMsgTxStateCancel:
		logger.Log(ctx, logger.Info, "Tx cancel : %s", txid.String())
	case handlers.ListenerMsgTxStateUnsafe:
		logger.Log(ctx, logger.Info, "Tx unsafe : %s", txid.String())
	}
	return nil
}

func (server *Server) HandleInSync(ctx context.Context) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
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
	return nil
}
