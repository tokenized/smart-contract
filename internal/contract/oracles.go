package contract

import (
	"context"

	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/platform/state"
)

func ExpandOracles(ctx context.Context, data *state.Contract) error {
	logger.Info(ctx, "Expanding %d oracle public keys", len(data.Oracles))

	// Expand oracle public keys
	// data.FullOracles = make([]Oracle, 0, len(data.Oracles))
	// for _, oracle := range data.Oracles {
	// 	fullKey, err := bitcoin.PublicKeyFromBytes(oracle.PublicKey)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	data.FullOracles = append(data.FullOracles, fullKey)
	// }
	return nil
}
