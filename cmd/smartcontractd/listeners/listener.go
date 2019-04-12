package listeners

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
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

func (server *Server) removeFromReverted(ctx context.Context, txid *chainhash.Hash) bool {
	for i, id := range server.revertedTxs {
		if bytes.Equal(id[:], txid[:]) {
			server.revertedTxs = append(server.revertedTxs[:i], server.revertedTxs[i+1:]...)
			return true
		}
	}

	return false
}

func (server *Server) HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		logger.Info(ctx, "Tx safe : %s", txid.String())

		if server.removeFromReverted(ctx, &txid) {
			logger.Verbose(ctx, "Tx safe again after reorg : %s", txid.String())
			return nil // Already accepted. Reverted by reorg and safe again.
		}

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

		if server.removeFromReverted(ctx, &txid) {
			logger.Verbose(ctx, "Tx reconfirmed in reorg : %s", txid.String())
			return nil // Already accepted. Reverted and reconfirmed by reorg
		}

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

		itx, err := transactions.GetTx(ctx, server.MasterDB, &txid, &server.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "Failed to get cancelled tx : %s", err)
		}

		err = server.cancelTx(ctx, itx)
		if err != nil {
			logger.Warn(ctx, "Failed to cancel tx : %s", err)
		}

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

		// This shouldn't happen. We should only get unsafe messages for txs that are not marked
		//   safe or confirmed yet.
		logger.Error(ctx, "Tx unsafe not found : %s", txid.String())

	case handlers.ListenerMsgTxStateRevert:
		logger.Info(ctx, "Tx revert : %s", txid.String())
		server.revertedTxs = append(server.revertedTxs, &txid)
	}
	return nil
}

func (server *Server) HandleInSync(ctx context.Context) error {
	if server.inSync {
		// Check for reorged reverted txs
		for _, txid := range server.revertedTxs {
			itx, err := transactions.GetTx(ctx, server.MasterDB, txid, &server.Config.ChainParams)
			if err != nil {
				logger.Warn(ctx, "Failed to get reverted tx : %s", err)
			}

			err = server.revertTx(ctx, itx)
			if err != nil {
				logger.Warn(ctx, "Failed to revert tx : %s", err)
			}
		}
		server.revertedTxs = nil
		return nil // Only execute below on first sync
	}

	ctx = logger.ContextWithOutLogSubSystem(ctx)
	logger.Info(ctx, "Node is in sync")
	server.inSync = true

	// Send pending responses
	pending := server.pendingResponses
	server.pendingResponses = nil

	for _, pendingTx := range pending {
		logger.Info(ctx, "Sending pending response: %s", pendingTx.TxHash().String())
		if err := server.sendTx(ctx, pendingTx); err != nil {
			if err != nil {
				logger.Warn(ctx, "Failed to send tx : %s", err)
			}
			return errors.Wrap(err, "Failed to send tx") // TODO Probably a fatal error
		}
	}

	// -------------------------------------------------------------------------
	// Schedule vote finalizers
	// Iterate through votes for each contract and if they aren't complete schedule a finalizer.
	keys := server.wallet.ListAll()
	for _, key := range keys {
		contractPKH := protocol.PublicKeyHashFromBytes(key.Address.ScriptAddress())
		votes, err := vote.List(ctx, server.MasterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to list votes")
		}
		for _, vt := range votes {
			if vt.CompletedAt.Nano() != 0 {
				continue // Already complete
			}

			// Retrieve voteTx
			var hash *chainhash.Hash
			hash, err = chainhash.NewHash(vt.VoteTxId.Bytes())
			if err != nil {
				return errors.Wrap(err, "Failed to create tx hash")
			}
			voteTx, err := transactions.GetTx(ctx, server.MasterDB, hash, &server.Config.ChainParams)
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve vote tx")
			}

			// Schedule vote finalizer
			if err = server.Scheduler.ScheduleJob(ctx, NewVoteFinalizer(server.Handler, voteTx, vt.Expires)); err != nil {
				return errors.Wrap(err, "Failed to schedule vote finalizer")
			}
		}
	}

	// -------------------------------------------------------------------------
	// Schedule pending transfer timeouts
	// Iterate through pending transfers for each contract and if they aren't complete schedule a timeout.
	for _, key := range keys {
		contractPKH := protocol.PublicKeyHashFromBytes(key.Address.ScriptAddress())
		transfers, err := transfer.List(ctx, server.MasterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to list transfers")
		}
		for _, pt := range transfers {
			// Retrieve transferTx
			var hash *chainhash.Hash
			hash, err = chainhash.NewHash(pt.TransferTxId.Bytes())
			if err != nil {
				return errors.Wrap(err, "Failed to create tx hash")
			}
			transferTx, err := transactions.GetTx(ctx, server.MasterDB, hash, &server.Config.ChainParams)
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve transfer tx")
			}

			// Schedule transfer timeout
			if err = server.Scheduler.ScheduleJob(ctx, NewTransferTimeout(server.Handler, transferTx, pt.Timeout)); err != nil {
				return errors.Wrap(err, "Failed to schedule transfer timeout")
			}
		}
	}

	return nil
}
