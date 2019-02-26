package vote

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
)

const storageKey = "contracts"

// Put a single vote in storage
func Save(ctx context.Context, dbConn *db.DB, pkh string, v state.Vote) error {

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

	// Initialize Vote map
	if c.Votes == nil {
		c.Votes = map[string]state.Vote{}
	}

	// Update the vote
	c.Votes[a.ID] = a

	// Save the contract
	sb, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, sb)
}

// Fetch a single vote from storage
func Fetch(ctx context.Context, dbConn *db.DB, pkh string, voteID string) (*state.Vote, error) {

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

	// Initialize Vote map
	if c.Votes == nil {
		c.Votes = map[string]state.Vote{}
	}

	// Locate the vote
	vote, ok := c.Votes[voteID]
	if !ok {
		return nil, ErrNotFound
	}

	return &vote, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(id string) string {
	return fmt.Sprintf("%v/%v", storageKey, id)
}
