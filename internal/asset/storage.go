package asset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

const storageKey = "contracts"

// Put a single asset in storage
func Save(ctx context.Context, dbConn *db.DB, pkh protocol.PublicKeyHash, a state.Asset) error {

	// Fetch the contract
	key := buildStoragePath(pkh)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return err
	}

	// Prepare the contract object
	c := state.Contract{}
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	// Initialize Asset map
	if c.Assets == nil {
		c.Assets = map[protocol.AssetCode]state.Asset{}
	}

	// Update the asset
	c.Assets[a.ID] = a

	// Save the contract
	sb, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, sb)
}

// Fetch a single asset from storage
func Fetch(ctx context.Context, dbConn *db.DB, pkh protocol.PublicKeyHash, assetCode protocol.AssetCode) (*state.Asset, error) {

	// Fetch the contract
	key := buildStoragePath(pkh)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	// Prepare the contract object
	c := state.Contract{}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	// Initialize Asset map
	if c.Assets == nil {
		c.Assets = map[protocol.AssetCode]state.Asset{}
	}

	// Locate the asset
	asset, ok := c.Assets[assetCode]
	if !ok {
		return nil, ErrNotFound
	}

	return &asset, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(id protocol.PublicKeyHash) string {
	return fmt.Sprintf("%v/%x", storageKey, id)
}
