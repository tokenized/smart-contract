package contract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/pkg/errors"
)

const storageKey = "contracts"

var cache map[bitcoin.Hash20]*state.Contract

// Put a single contract in storage
func Save(ctx context.Context, dbConn *db.DB, contract *state.Contract) error {
	contractHash, err := contract.Address.Hash()
	if err != nil {
		return err
	}

	b, err := json.Marshal(contract)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal contract")
	}

	key := buildStoragePath(contractHash)

	if err := dbConn.Put(ctx, key, b); err != nil {
		return err
	}

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Contract)
	}
	cache[*contractHash] = contract
	return nil
}

// Fetch a single contract from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress) (*state.Contract, error) {
	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}
	if cache != nil {
		result, exists := cache[*contractHash]
		if exists {
			return result, nil
		}
	}

	key := buildStoragePath(contractHash)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch contract")
	}

	result := state.Contract{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal contract")
	}

	if err := ExpandOracles(ctx, &result); err != nil {
		return nil, err
	}

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Contract)
	}
	cache[*contractHash] = &result
	return &result, nil
}

func Reset(ctx context.Context) {
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/contract", storageKey, contractHash.String())
}

func ExpandOracles(ctx context.Context, data *state.Contract) error {
	logger.Info(ctx, "Expanding %d oracle public keys", len(data.Oracles))

	// Expand oracle public keys
	data.FullOracles = make([]bitcoin.PublicKey, 0, len(data.Oracles))
	for _, oracle := range data.Oracles {
		fullKey, err := bitcoin.PublicKeyFromBytes(oracle.PublicKey)
		if err != nil {
			return err
		}
		data.FullOracles = append(data.FullOracles, fullKey)
	}
	return nil
}
