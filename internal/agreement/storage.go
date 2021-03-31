package agreement

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/pkg/errors"
)

const storageKey = "contracts"

// Put a single agreement in storage
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	agreement *state.Agreement) error {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return err
	}

	b, err := json.Marshal(agreement)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal agreement")
	}

	key := buildStoragePath(contractHash)

	if err := dbConn.Put(ctx, key, b); err != nil {
		return err
	}

	return nil
}

// Fetch a single agreement from storage
func Fetch(ctx context.Context, dbConn *db.DB,
	contractAddress bitcoin.RawAddress) (*state.Agreement, error) {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}

	key := buildStoragePath(contractHash)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch agreement")
	}

	result := state.Agreement{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal agreement")
	}

	return &result, nil
}

func Reset(ctx context.Context) {}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/agreement", storageKey, contractHash)
}
