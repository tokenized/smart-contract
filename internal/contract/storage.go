package contract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const storageKey = "contracts"

// Put a single contract in storage
func Save(ctx context.Context, dbConn *db.DB, contract *state.Contract) error {
	b, err := json.Marshal(contract)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal contract")
	}

	key := buildStoragePath(&contract.ID)

	return dbConn.Put(ctx, key, b)
}

// Fetch a single contract from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash) (*state.Contract, error) {
	key := buildStoragePath(contractPKH)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch contract")
	}

	contract := state.Contract{}
	if err := json.Unmarshal(b, &contract); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal contract")
	}

	return &contract, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash) string {
	return fmt.Sprintf("%s/%s/contract", storageKey, contractPKH.String())
}
