package listeners

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Implement the SpyNode Listener interface.

func (server *Server) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgBlock:
		logger.Info(ctx, "New Block (%d) : %s", block.Height, block.Hash.String())
	case handlers.ListenerMsgBlockRevert:
		logger.Info(ctx, "Reverted Block (%d) : %s", block.Height, block.Hash.String())
	}
	return nil
}

func (server *Server) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

	logger.Info(ctx, "Tx : %s", tx.TxHash().String())

	// Check if transaction relates to protocol
	itx, err := inspector.NewTransactionFromWire(ctx, tx)
	if err != nil {
		logger.Warn(ctx, "Failed to create inspector tx : %s", err)
		return false, err
	}

	// Prefilter out non-protocol messages
	if !itx.IsTokenized() {
		logger.Verbose(ctx, "Not tokenized tx : %s", tx.TxHash().String())
		return false, nil
	}

	// Promote TX
	if err := itx.Promote(ctx, server.RpcNode); err != nil {
		logger.Fatal(ctx, "Failed to promote inspector tx : %s", err)
		return false, err
	}

	server.pendingRequests = append(server.pendingRequests, itx)
	return true, nil
}

func (server *Server) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		logger.Info(ctx, "Tx safe : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)
				err := server.processTx(ctx, itx)
				if err != nil {
					logger.Warn(ctx, "Failed to process safe tx : %s", err)
				}
				return err
			}
		}

		logger.Verbose(ctx, "Tx safe not found : %s", txid.String())
		return nil

	case handlers.ListenerMsgTxStateConfirm:
		logger.Info(ctx, "Tx confirm : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)
				err := server.processTx(ctx, itx)
				if err != nil {
					logger.Warn(ctx, "Failed to process confirm tx : %s", err)
				}
				return err
			}
		}

		for i, itx := range server.unsafeRequests {
			if itx.Hash == txid {
				// Remove from unsafeRequests
				server.unsafeRequests = append(server.unsafeRequests[:i], server.unsafeRequests[i+1:]...)
				err := server.processTx(ctx, itx)
				if err != nil {
					logger.Warn(ctx, "Failed to process unsafe confirm tx : %s", err)
				}
				return err
			}
		}

		logger.Verbose(ctx, "Tx confirm not found : %s", txid.String())
		return nil

	case handlers.ListenerMsgTxStateCancel:
		logger.Info(ctx, "Tx cancel : %s", txid.String())
		for i, itx := range server.pendingRequests {
			if itx.Hash == txid {
				// Remove from pending
				server.pendingRequests = append(server.pendingRequests[:i], server.pendingRequests[i+1:]...)
				return nil
			}
		}

		// TODO We have to manually undo or revert action
		logger.Error(ctx, "Tx cancel not found : %s", txid.String())

	case handlers.ListenerMsgTxStateUnsafe:
		logger.Info(ctx, "Tx unsafe : %s", txid.String())
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
		logger.Error(ctx, "Tx unsafe not found : %s", txid.String())

	case handlers.ListenerMsgTxStateRevert:
		logger.Info(ctx, "Tx revert : %s", txid.String())
	}
	return nil
}

func (server *Server) HandleInSync(ctx context.Context) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	logger.Info(ctx, "Node is in sync")

	// Send pending responses
	server.inSync = true
	pending := server.pendingResponses
	server.pendingResponses = nil

	for _, pendingTx := range pending {
		logger.Info(ctx, "Sending pending response: %s", pendingTx.TxHash().String())
		if err := server.sendTx(ctx, pendingTx); err != nil {
			if err != nil {
				logger.Warn(ctx, "Failed to send tx : %s", err)
			}
			return err // TODO Probably a fatal error
		}
	}
	return nil
}
