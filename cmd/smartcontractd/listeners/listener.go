package listeners

import (
	"context"

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

	server.pendingRequests = append(server.pendingRequests, itx)
	return true, nil
}

func (server *Server) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		logger.Log(ctx, logger.Info, "Tx safe : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)

				return server.processTx(ctx, itx)
			}
		}

		logger.Log(ctx, logger.Verbose, "Tx safe not found : %s", txid.String())
		return nil

	case handlers.ListenerMsgTxStateConfirm:
		logger.Log(ctx, logger.Info, "Tx confirm : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)

				return server.processTx(ctx, itx)
			}
		}

		for i, itx := range server.unsafeRequests {
			if itx.Hash == txid {
				// Remove from unsafeRequests
				server.unsafeRequests = append(server.unsafeRequests[:i], server.unsafeRequests[i+1:]...)

				return server.processTx(ctx, itx)
			}
		}

		logger.Log(ctx, logger.Verbose, "Tx confirm not found : %s", txid.String())
		return nil

	case handlers.ListenerMsgTxStateCancel:
		logger.Log(ctx, logger.Info, "Tx cancel : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)
				return nil
			}
		}

		// TODO We have to manually undo or revert action
		logger.Log(ctx, logger.Error, "Tx cancel not found : %s", txid.String())

	case handlers.ListenerMsgTxStateUnsafe:
		logger.Log(ctx, logger.Info, "Tx unsafe : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Add to unsafe
				server.unsafeRequests = append(server.unsafeRequests, server.pendingRequests[i])

				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)

				return nil
			}
		}

		// TODO We have to manually undo or revert action
		logger.Log(ctx, logger.Error, "Tx unsafe not found : %s", txid.String())

	case handlers.ListenerMsgTxStateRevert:
		logger.Log(ctx, logger.Info, "Tx revert : %s", txid.String())
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
