package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// A filter to determine which transactions should be passed through to the listeners.
// If no filters are specified, then all txs will be passed to listeners.
type TxFilter interface {
	IsRelevant(context.Context, *wire.MsgTx) bool
}

func matchesFilter(ctx context.Context, tx *wire.MsgTx, filters []TxFilter) bool {
	if len(filters) == 0 {
		return true // No filters means all tx are "relevant"
	}

	// Check filters
	for _, filter := range filters {
		if filter.IsRelevant(ctx, tx) {
			return true
		}
	}

	return false
}
