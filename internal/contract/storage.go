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

var cache map[protocol.PublicKeyHash]*state.Contract

// Put a single contract in storage
func Save(ctx context.Context, dbConn *db.DB, contract *state.Contract) error {
	b, err := json.Marshal(contract)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal contract")
	}

	key := buildStoragePath(&contract.ID)

	if err := dbConn.Put(ctx, key, b); err != nil {
		return err
	}

	if cache == nil {
		cache = make(map[protocol.PublicKeyHash]*state.Contract)
	}
	cache[contract.ID] = contract
	return nil
}

// Fetch a single contract from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash) (*state.Contract, error) {
	if cache != nil {
		result, exists := cache[*contractPKH]
		if exists {
			return result, nil
		}
	}

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

func Reset(ctx context.Context) {
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash) string {
	return fmt.Sprintf("%s/%s/contract", storageKey, contractPKH.String())
}
