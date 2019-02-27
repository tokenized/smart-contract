package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// A filter to determine which transactions are relevant
type TxFilter interface {
	IsRelevant(context.Context, *wire.MsgTx) bool
}

func matchesFilter(ctx context.Context, tx *wire.MsgTx, filters []TxFilter) bool {
	if len(filters) == 0 {
		logger.Log(ctx, logger.Info, "No filters")
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
