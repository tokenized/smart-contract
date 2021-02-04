package listeners

import (
	"bytes"
	"context"
	"sort"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/transfer"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/spynode/pkg/client"

	"github.com/pkg/errors"
)

// Implement the SpyNode Client interface.

func (server *Server) HandleMessage(ctx context.Context, payload client.MessagePayload) {
	switch payload.(type) {
	case *client.AcceptRegister:
		logger.Info(ctx, "SpyNode registration accepted")

		if server.SpyNode != nil {
			if err := server.SpyNode.SubscribeContracts(ctx); err != nil {
				logger.Error(ctx, "Failed to subscribe to contracts : %s", err)
			}

			keys := server.wallet.ListAll()
			addresses := make([]bitcoin.RawAddress, len(keys))
			for i, key := range keys {
				addresses[i] = key.Address
			}

			logger.Info(ctx, "Subscribing to %d addresses", len(addresses))
			if err := client.SubscribeAddresses(ctx, addresses, server.SpyNode); err != nil {
				logger.Error(ctx, "Failed to subscribe to contract addresses : %s", err)
			}

			nextMessageID, err := state.GetNextMessageID(ctx, server.MasterDB)
			if err != nil {
				logger.Error(ctx, "Failed to get next message id : %s", err)
				return
			}

			if err := server.SpyNode.Ready(ctx, *nextMessageID); err != nil {
				logger.Error(ctx, "Failed to notify spynode ready : %s", err)
				return
			}

			logger.Info(ctx, "SpyNode client ready at next message %d", *nextMessageID)
		}
	}
}

func (server *Server) HandleTx(ctx context.Context, tx *client.Tx) {
	ctx = node.ContextWithOutLogSubSystem(ctx)
	txid := tx.Tx.TxHash()
	ctx = node.ContextWithLogTrace(ctx, txid.String())

	if tx.ID != 0 {
		if err := state.SaveNextMessageID(ctx, server.MasterDB, tx.ID+1); err != nil {
			logger.Error(ctx, "Failed to save next message id : %s", err)
		}
	}

	err := server.AddTx(ctx, tx, *txid)
	if err != nil {
		if errors.Cause(err) != ErrDuplicateTx {
			node.LogError(ctx, "Failed to add tx : %s", err)
		}
	}

	server.handleTxState(ctx, *txid, &tx.State)

	node.Log(ctx, "Handled tx")
}

func (server *Server) HandleTxUpdate(ctx context.Context, update *client.TxUpdate) {
	ctx = node.ContextWithOutLogSubSystem(ctx)
	ctx = node.ContextWithLogTrace(ctx, update.TxID.String())

	server.handleTxState(ctx, update.TxID, &update.State)

	node.Log(ctx, "Handled tx state")
}

func (server *Server) handleTxState(ctx context.Context, txid bitcoin.Hash32,
	state *client.TxState) {

	if state.UnSafe {
		node.Log(ctx, "Tx unsafe")
		server.MarkUnsafe(ctx, txid)
	} else if state.Cancelled {
		node.Log(ctx, "Tx cancel")

		if server.CancelPendingTx(ctx, txid) {
			return
		}

		itx, err := transactions.GetTx(ctx, server.MasterDB, &txid, server.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Failed to get cancelled tx : %s", err)
		}

		err = server.cancelTx(ctx, itx)
		if err != nil {
			node.LogWarn(ctx, "Failed to cancel tx : %s", err)
		}
	} else if state.MerkleProof != nil {
		node.Log(ctx, "Tx confirm")

		if server.removeFromReverted(ctx, &txid) {
			node.LogVerbose(ctx, "Tx reconfirmed in reorg")
			return // Already accepted. Reverted and reconfirmed by reorg
		}

		server.MarkConfirmed(ctx, txid)
	} else if state.Safe {
		node.Log(ctx, "Tx safe")

		if server.removeFromReverted(ctx, &txid) {
			node.LogVerbose(ctx, "Tx safe again after reorg")
			return // Already accepted. Reverted by reorg and safe again.
		}

		server.MarkSafe(ctx, txid)
	}
}

func (server *Server) HandleHeaders(ctx context.Context, headers *client.Headers) {
	ctx = node.ContextWithOutLogSubSystem(ctx)
	count := len(headers.Headers)
	node.Log(ctx, "New headers (%d) to height %d : %s", count, int(headers.StartHeight)+count-1,
		headers.Headers[count-1].BlockHash())
}

func (server *Server) HandleInSync(ctx context.Context) {
	ctx = node.ContextWithOutLogSubSystem(ctx)

	if server.IsInSync() {
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
		return // Only execute below on first sync
	}

	ctx = node.ContextWithLogTrace(ctx, "In Sync")
	node.Log(ctx, "Node is in sync")
	node.Log(ctx, "Processing pending : %d responses, %d requests", len(server.pendingResponses),
		len(server.pendingRequests))
	server.SetInSync()
	pendingResponses := server.pendingResponses
	server.pendingResponses = nil
	pendingRequests := server.pendingRequests
	server.pendingRequests = nil

	// Sort pending responses by timestamp, so they are handled in the same order as originally.
	sort.Sort(&pendingResponses)

	// Process pending responses
	for _, itx := range pendingResponses {
		ctx := node.ContextWithLogTrace(ctx, itx.Hash.String())
		node.Log(ctx, "Processing pending response")
		if err := server.Handler.Trigger(ctx, "SEE", itx); err != nil {
			node.LogError(ctx, "Failed to handle pending response tx : %s", err)
		}
	}

	// Process pending requests
	for _, tx := range pendingRequests {
		ctx := node.ContextWithLogTrace(ctx, tx.Itx.Hash.String())
		node.Log(ctx, "Processing pending request")
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
			return
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
				return
			}
			voteTx, err := transactions.GetTx(ctx, server.MasterDB, hash, server.Config.IsTest)
			if err != nil {
				node.LogWarn(ctx, "Failed to retrieve vote tx : %s", err)
				return
			}

			// Schedule vote finalizer
			if err = server.Scheduler.ScheduleJob(ctx, NewVoteFinalizer(server.Handler, voteTx, vt.Expires)); err != nil {
				node.LogWarn(ctx, "Failed to schedule vote finalizer : %s", err)
				return
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
			return
		}
		for _, pt := range transfers {
			// Retrieve transferTx
			var hash *bitcoin.Hash32
			hash, err = bitcoin.NewHash32(pt.TransferTxId.Bytes())
			if err != nil {
				node.LogWarn(ctx, "Failed to create tx hash : %s", err)
				return
			}
			transferTx, err := transactions.GetTx(ctx, server.MasterDB, hash, server.Config.IsTest)
			if err != nil {
				node.LogWarn(ctx, "Failed to retrieve transfer tx : %s", err)
				return
			}

			// Schedule transfer timeout
			if err = server.Scheduler.ScheduleJob(ctx, NewTransferTimeout(server.Handler, transferTx, pt.Timeout)); err != nil {
				node.LogWarn(ctx, "Failed to schedule transfer timeout : %s", err)
				return
			}
		}
	}
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
