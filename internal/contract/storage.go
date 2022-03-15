package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/pkg/errors"
)

const storageKey = "contracts"

var cache map[bitcoin.Hash20]*state.Contract
var cacheLock sync.Mutex

// Put a single contract in storage
func Save(ctx context.Context, dbConn *db.DB, contract *state.Contract, isTest bool) error {
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

	cacheLock.Lock()
	defer cacheLock.Unlock()
	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Contract)
	}
	cache[*contractHash] = contract
	return nil
}

// Fetch a single contract from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	isTest bool) (*state.Contract, error) {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}
	cacheLock.Lock()
	defer cacheLock.Unlock()
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

	if err := ExpandOracles(ctx, dbConn, &result, isTest); err != nil {
		return nil, err
	}

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Contract)
	}
	cache[*contractHash] = &result
	return &result, nil
}

func Reset(ctx context.Context) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/contract", storageKey, contractHash.String())
}
