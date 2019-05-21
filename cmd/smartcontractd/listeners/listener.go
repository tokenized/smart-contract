package listeners

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Implement the SpyNode Listener interface.

func (server *Server) HandleBlock(ctx context.Context, msgType int, block *handlers.BlockMessage) error {
	ctx = node.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgBlock:
		node.Log(ctx, "New Block (%d) : %s", block.Height, block.Hash.String())
	case handlers.ListenerMsgBlockRevert:
		node.Log(ctx, "Reverted Block (%d) : %s", block.Height, block.Hash.String())
	}
	return nil
}

func (server *Server) HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error) {
	ctx = node.ContextWithOutLogSubSystem(ctx)
	err := server.AddTx(ctx, tx)
	if err != nil {
		node.LogError(ctx, "Failed to add tx : %s", err)
		return true, nil
	}

	node.Log(ctx, "Tx : %s", tx.TxHash().String())
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
	ctx = node.ContextWithOutLogSubSystem(ctx)
	switch msgType {
	case handlers.ListenerMsgTxStateSafe:
		node.Log(ctx, "Tx safe : %s", txid.String())

		if server.removeFromReverted(ctx, &txid) {
			node.LogVerbose(ctx, "Tx safe again after reorg : %s", txid.String())
			return nil // Already accepted. Reverted by reorg and safe again.
		}

		server.MarkSafe(ctx, &txid)
		return nil

	case handlers.ListenerMsgTxStateConfirm:
		node.Log(ctx, "Tx confirm : %s", txid.String())

		if server.removeFromReverted(ctx, &txid) {
			node.LogVerbose(ctx, "Tx reconfirmed in reorg : %s", txid.String())
			return nil // Already accepted. Reverted and reconfirmed by reorg
		}

		server.MarkConfirmed(ctx, &txid)
		return nil

	case handlers.ListenerMsgTxStateCancel:
		node.Log(ctx, "Tx cancel : %s", txid.String())

		if server.CancelPendingTx(ctx, &txid) {
			return nil
		}

		itx, err := transactions.GetTx(ctx, server.MasterDB, &txid, &server.Config.ChainParams, server.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Failed to get cancelled tx : %s", err)
		}

		err = server.cancelTx(ctx, itx)
		if err != nil {
			node.LogWarn(ctx, "Failed to cancel tx : %s", err)
		}

	case handlers.ListenerMsgTxStateUnsafe:
		node.Log(ctx, "Tx unsafe : %s", txid.String())
		server.MarkUnsafe(ctx, &txid)

	case handlers.ListenerMsgTxStateRevert:
		node.Log(ctx, "Tx revert : %s", txid.String())
		server.revertedTxs = append(server.revertedTxs, &txid)
	}
	return nil
}

func (server *Server) HandleInSync(ctx context.Context) error {
	if server.inSync {
		// Check for reorged reverted txs
		for _, txid := range server.revertedTxs {
			itx, err := transactions.GetTx(ctx, server.MasterDB, txid, &server.Config.ChainParams, server.Config.IsTest)
			if err != nil {
				node.LogWarn(ctx, "Failed to get reverted tx : %s", err)
			}

			err = server.revertTx(ctx, itx)
			if err != nil {
				node.LogWarn(ctx, "Failed to revert tx : %s", err)
			}
		}
		server.revertedTxs = nil
		return nil // Only execute below on first sync
	}

	ctx = node.ContextWithOutLogSubSystem(ctx)
	node.Log(ctx, "Node is in sync")
	server.inSync = true

	// Send pending responses
	pending := server.pendingResponses
	server.pendingResponses = nil

	for _, pendingTx := range pending {
		node.Log(ctx, "Sending pending response: %s", pendingTx.TxHash().String())
		if err := server.sendTx(ctx, pendingTx); err != nil {
			if err != nil {
				node.LogWarn(ctx, "Failed to send tx : %s", err)
			}
			return nil // TODO Probably a fatal error
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
			node.LogWarn(ctx, "Failed to list votes : %s", err)
			return nil
		}
		for _, vt := range votes {
			if vt.CompletedAt.Nano() != 0 {
				continue // Already complete
			}

			// Retrieve voteTx
			var hash *chainhash.Hash
			hash, err = chainhash.NewHash(vt.VoteTxId.Bytes())
			if err != nil {
				node.LogWarn(ctx, "Failed to create tx hash : %s", err)
				return nil
			}
			voteTx, err := transactions.GetTx(ctx, server.MasterDB, hash, &server.Config.ChainParams, server.Config.IsTest)
			if err != nil {
				node.LogWarn(ctx, "Failed to retrieve vote tx : %s", err)
				return nil
			}

			// Schedule vote finalizer
			if err = server.Scheduler.ScheduleJob(ctx, NewVoteFinalizer(server.Handler, voteTx, vt.Expires)); err != nil {
				node.LogWarn(ctx, "Failed to schedule vote finalizer : %s", err)
				return nil
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
			node.LogWarn(ctx, "Failed to list transfers : %s", err)
			return nil
		}
		for _, pt := range transfers {
			// Retrieve transferTx
			var hash *chainhash.Hash
			hash, err = chainhash.NewHash(pt.TransferTxId.Bytes())
			if err != nil {
				node.LogWarn(ctx, "Failed to create tx hash : %s", err)
				return nil
			}
			transferTx, err := transactions.GetTx(ctx, server.MasterDB, hash, &server.Config.ChainParams, server.Config.IsTest)
			if err != nil {
				node.LogWarn(ctx, "Failed to retrieve transfer tx : %s", err)
				return nil
			}

			// Schedule transfer timeout
			if err = server.Scheduler.ScheduleJob(ctx, NewTransferTimeout(server.Handler, transferTx, pt.Timeout)); err != nil {
				node.LogWarn(ctx, "Failed to schedule transfer timeout : %s", err)
				return nil
			}
		}
	}

	return nil
}
