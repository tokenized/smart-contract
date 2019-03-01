package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// TXHandler exists to handle the tx command.
type TXHandler struct {
	ready     StateReady
	memPool   *data.MemPool
	txs       *storage.TxRepository
	listeners []Listener
	txFilters []TxFilter
}

// NewTXHandler returns a new TXHandler with the given Config.
func NewTXHandler(ready StateReady, memPool *data.MemPool, txs *storage.TxRepository, listeners []Listener, txFilters []TxFilter) *TXHandler {
	result := TXHandler{
		ready:     ready,
		memPool:   memPool,
		txs:       txs,
		listeners: listeners,
		txFilters: txFilters,
	}
	return &result
}

type TxKey int

var DirectTxKey TxKey = 1 // Used in context to flag when a tx is from the system

func (handler *TXHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgTx)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgTx")
	}

	// Only notify of transactions when in sync or they might be duplicated
	if !handler.ready.IsReady() && ctx.Value(DirectTxKey) == nil {
		return nil, nil
	}

	// The mempool is needed to track which transactions have been sent to listeners and to check
	//   for attempted double spends.
	conflicts, added := handler.memPool.AddTransaction(msg)

	if !added {
		return nil, nil // Already saw this tx
	}

	if len(conflicts) > 0 {
		logger.Log(ctx, logger.Warn, "Found %d conflicts with %s", len(conflicts), msg.TxHash().String())
		// Notify of attempted double spend
		for _, conflict := range conflicts {
			contains, err := handler.txs.Contains(ctx, *conflict, -1)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to check tx repo")
			}
			if contains { // Only send for txs that previously matched filters.
				for _, listener := range handler.listeners {
					listener.Handle(ctx, ListenerMsgTxUnsafe, *conflict)
				}
			}
		}
	}

	if !matchesFilter(ctx, msg, handler.txFilters) {
		return nil, nil // Filter out
	}

	// Add to tx repository as "relevant" unconfirmed tx
	repoAdded, err := handler.txs.Add(ctx, msg.TxHash(), -1)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to add to tx repo")
	}
	if !repoAdded { // Not added to tx repo, so already seen
		return nil, nil
	}

	// Notify of new tx
	for _, listener := range handler.listeners {
		listener.Handle(ctx, ListenerMsgTx, *msg)
	}
	return nil, nil
}
