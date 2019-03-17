package contract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

const storageKey = "contracts"

// Put a single contract in storage
func Save(ctx context.Context, dbConn *db.DB, c state.Contract) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := buildStoragePath(c.ID)

	return dbConn.Put(ctx, key, b)
}

// Fetch a single contract from storage
func Fetch(ctx context.Context, dbConn *db.DB, pkh protocol.PublicKeyHash) (*state.Contract, error) {
	key := buildStoragePath(pkh)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	c := state.Contract{}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	// Initialize Asset map
	if c.Assets == nil {
		c.Assets = map[protocol.AssetCode]state.Asset{}
	}

	// Initialize Vote map
	if c.Votes == nil {
		c.Votes = map[protocol.TxId]state.Vote{}
	}

	return &c, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(id protocol.PublicKeyHash) string {
	return fmt.Sprintf("%v/%x", storageKey, id)
}
