package listeners

import (
	"bytes"
	"context"
	"sort"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
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

func (server *Server) removeFromReverted(ctx context.Context, txid *bitcoin.Hash32) bool {
	for i, id := range server.revertedTxs {
		if bytes.Equal(id[:], txid[:]) {
			server.revertedTxs = append(server.revertedTxs[:i], server.revertedTxs[i+1:]...)
			return true
		}
	}

	return false
}

func (server *Server) HandleTxState(ctx context.Context, msgType int, txid bitcoin.Hash32) error {
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

		itx, err := transactions.GetTx(ctx, server.MasterDB, &txid, server.Config.IsTest)
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
			itx, err := transactions.GetTx(ctx, server.MasterDB, txid, server.Config.IsTest)
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
	pendingResponses := server.pendingResponses
	server.pendingResponses = nil
	pendingRequests := server.pendingRequests
	server.pendingRequests = nil

	// Sort pending responses by timestamp, so they are handled in the same order as originally.
	sort.Sort(&pendingResponses)

	// Process pending responses
	for _, itx := range pendingResponses {
		node.Log(ctx, "Processing pending response: %s", itx.Hash.String())
		if err := server.Handler.Trigger(ctx, "SEE", itx); err != nil {
			node.LogError(ctx, "Failed to handle pending response tx : %s", err)
		}
	}

	// Process pending requests
	for _, tx := range pendingRequests {
		node.Log(ctx, "Processing pending request: %s", tx.Itx.Hash.String())
		if err := server.Handler.Trigger(ctx, "SEE", tx.Itx); err != nil {
			node.LogError(ctx, "Failed to handle pending request tx : %s", err)
		}
	}

	// -------------------------------------------------------------------------
	// Schedule vote finalizers
	// Iterate through votes for each contract and if they aren't complete schedule a finalizer.
	keys := server.wallet.ListAll()
	for _, key := range keys {
		votes, err := vote.List(ctx, server.MasterDB, key.Address)
		if err != nil {
			node.LogWarn(ctx, "Failed to list votes : %s", err)
			return nil
		}
		for _, vt := range votes {
			if vt.CompletedAt.Nano() != 0 {
				continue // Already complete
			}

			// Retrieve voteTx
			var hash *bitcoin.Hash32
			hash, err = bitcoin.NewHash32(vt.VoteTxId.Bytes())
			if err != nil {
				node.LogWarn(ctx, "Failed to create tx hash : %s", err)
				return nil
			}
			voteTx, err := transactions.GetTx(ctx, server.MasterDB, hash, server.Config.IsTest)
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
		transfers, err := transfer.List(ctx, server.MasterDB, key.Address)
		if err != nil {
			node.LogWarn(ctx, "Failed to list transfers : %s", err)
			return nil
		}
		for _, pt := range transfers {
			// Retrieve transferTx
			var hash *bitcoin.Hash32
			hash, err = bitcoin.NewHash32(pt.TransferTxId.Bytes())
			if err != nil {
				node.LogWarn(ctx, "Failed to create tx hash : %s", err)
				return nil
			}
			transferTx, err := transactions.GetTx(ctx, server.MasterDB, hash, server.Config.IsTest)
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
