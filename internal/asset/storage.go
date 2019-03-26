package asset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/pkg/errors"
)

const storageKey = "contracts"

// Put a single asset in storage
func Save(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, asset *state.Asset) error {
	data, err := json.Marshal(asset)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal asset")
	}

	return dbConn.Put(ctx, buildStoragePath(contractPKH, &asset.ID), data)
}

// Fetch a single asset from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode) (*state.Asset, error) {
	key := buildStoragePath(contractPKH, assetCode)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch asset")
	}

	// Prepare the contract object
	asset := state.Asset{}
	if err := json.Unmarshal(b, &asset); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal asset")
	}

	return &asset, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, asset *protocol.AssetCode) string {
	return fmt.Sprintf("%v/%s/%s", storageKey, contractPKH.String(), asset.String())
}
