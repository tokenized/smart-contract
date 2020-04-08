package spynode

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// HandleTx processes a tx through spynode as if it came from the network.
// Used to feed "response" txs directly back through spynode.
func (node *Node) HandleTx(ctx context.Context, tx *wire.MsgTx) error {
	return node.unconfTxChannel.Add(handlers.TxData{Msg: tx, Trusted: true, Safe: true,
		ConfirmedHeight: -1})
}

func (node *Node) processConfirmedTx(ctx context.Context, tx handlers.TxData) error {
	hash := tx.Msg.TxHash()

	if tx.ConfirmedHeight == -1 {
		return errors.New("Process confirmed tx with no height")
	}

	// Send full tx to listener if we aren't in sync yet and don't have a populated mempool.
	// Or if it isn't in the mempool (not sent to listener yet).
	isRelevant := false
	if handlers.MatchesFilter(ctx, tx.Msg, node.txFilters) {
		for _, listener := range node.listeners {
			if rel, _ := listener.HandleTx(ctx, tx.Msg); rel {
				isRelevant = true
			}
		}
	}

	if isRelevant {
		// Notify of confirm
		node.txStateChannel.Add(handlers.TxState{
			handlers.ListenerMsgTxStateConfirm,
			*hash,
		})

		// Add to txs for block
		if _, _, err := node.txs.Add(ctx, *hash, tx.Trusted, tx.Safe, tx.ConfirmedHeight); err != nil {
			return err
		}
	}

	return nil
}

func (node *Node) processUnconfirmedTx(ctx context.Context, tx handlers.TxData) error {
	hash := tx.Msg.TxHash()

	if tx.ConfirmedHeight != -1 {
		return errors.New("Process unconfirmed tx with height")
	}

	node.txTracker.Remove(ctx, *hash)

	// The mempool is needed to track which transactions have been sent to listeners and to check
	//   for attempted double spends.
	conflicts, trusted, added := node.memPool.AddTransaction(ctx, tx.Msg, tx.Trusted)
	if !added {
		return nil // Already saw this tx
	}

	logger.Debug(ctx, "Tx mempool (added %t) (flagged trusted %t) (received trusted %t) : %s",
		added, trusted, tx.Trusted, hash.String())

	if trusted {
		// Was marked trusted in the mempool by a tx inventory from the trusted node.
		// tx.Trusted means the tx itself was received from the trusted node.
		tx.Trusted = trusted
	}

	if len(conflicts) > 0 {
		logger.Warn(ctx, "Found %d conflicts with %s", len(conflicts), hash)
		// Notify of attempted double spend
		for _, conflict := range conflicts {
			isRelevant, err := node.txs.MarkUnsafe(ctx, conflict)
			if err != nil {
				return errors.Wrap(err, "Failed to check tx repo")
			}
			if !isRelevant {
				continue // Only send for txs that previously matched filters.
			}

			node.txStateChannel.Add(handlers.TxState{
				handlers.ListenerMsgTxStateUnsafe,
				conflict,
			})
		}
	}

	// We have to succesfully add to tx repo because it is protected by a lock and will prevent
	//   processing the same tx twice at the same time.
	added, newlySafe, err := node.txs.Add(ctx, *hash, tx.Trusted, tx.Safe, -1)
	if err != nil {
		return errors.Wrap(err, "Failed to add to tx repo")
	}
	if !added {
		return nil // tx already processed
	}

	logger.Debug(ctx, "Tx repo (added %t) (newly safe %t) : %s", added, newlySafe, hash.String())

	isRelevant := false
	if !handlers.MatchesFilter(ctx, tx.Msg, node.txFilters) {
		if _, err := node.txs.Remove(ctx, *hash, -1); err != nil {
			return errors.Wrap(err, "Failed to remove from tx repo")
		}
		return nil // Filter out
	}

	// Notify of new tx
	for _, listener := range node.listeners {
		if rel, _ := listener.HandleTx(ctx, tx.Msg); rel {
			if !isRelevant {
				isRelevant = true
			}
		}
	}

	if !isRelevant {
		// Remove from tx repository
		if _, err := node.txs.Remove(ctx, *hash, -1); err != nil {
			return errors.Wrap(err, "Failed to remove from tx repo")
		}
		return nil
	}

	// Notify of conflicting txs
	if len(conflicts) > 0 {
		node.txs.MarkUnsafe(ctx, *hash)
		node.txStateChannel.Add(handlers.TxState{
			handlers.ListenerMsgTxStateUnsafe,
			*hash,
		})
	} else if (added && tx.Safe) || newlySafe {
		// Note: A tx can be marked safe without being marked trusted if it is created internally.
		node.txStateChannel.Add(handlers.TxState{
			handlers.ListenerMsgTxStateSafe,
			*hash,
		})
	}

	return nil
}

// processUnconfirmedTxs pulls txs from the unconfirmed tx channel and processes them.
func (node *Node) processUnconfirmedTxs(ctx context.Context) {
	for tx := range node.unconfTxChannel.Channel {
		if err := node.processUnconfirmedTx(ctx, tx); err != nil {
			logger.Error(ctx, "SpyNodeAborted to process unconfirmed tx : %s : %s", err,
				tx.Msg.TxHash().String())
			node.requestStop(ctx)
			break
		}
	}
}

// processConfirmedTxs pulls txs from the confiremd tx channel and processes them.
func (node *Node) processConfirmedTxs(ctx context.Context) {
	for tx := range node.confTxChannel.Channel {
		if err := node.processConfirmedTx(ctx, tx); err != nil {
			logger.Error(ctx, "SpyNodeAborted to process confirmed tx : %s : %s", err,
				tx.Msg.TxHash().String())
			node.requestStop(ctx)
			break
		}
	}
}

// processhandlers.TxStates pulls txs from the tx state channel and processes them.
func (node *Node) processTxStates(ctx context.Context) {
	for txState := range node.txStateChannel.Channel {
		for _, listener := range node.listeners {
			listener.HandleTxState(ctx, txState.MsgType, txState.TxId)
		}
	}
}
