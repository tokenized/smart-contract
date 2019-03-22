package vote

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

const storageKey = "contracts"

// Put a single vote in storage
func Save(ctx context.Context, dbConn *db.DB, pkh *protocol.PublicKeyHash, v *state.Vote) error {

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

	// Initialize Vote map
	if c.Votes == nil {
		c.Votes = make(map[protocol.TxId]*state.Vote)
	}

	// Update the vote
	c.Votes[v.RefTxID] = v

	// Save the contract
	sb, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, sb)
}

// Fetch a single vote from storage
func Fetch(ctx context.Context, dbConn *db.DB, pkh *protocol.PublicKeyHash, voteID *protocol.TxId) (*state.Vote, error) {

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
		c.Votes = make(map[protocol.TxId]*state.Vote)
	}

	// Locate the vote
	vote, ok := c.Votes[*voteID]
	if !ok {
		return nil, ErrNotFound
	}

	return vote, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(id *protocol.PublicKeyHash) string {
	return fmt.Sprintf("%v/%x", storageKey, id)
}
