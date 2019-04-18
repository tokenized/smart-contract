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
const storageSubKey = "votes"

// Put a single vote in storage
func Save(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, v *state.Vote) error {
	key := buildStoragePath(contractPKH, &v.VoteTxId)

	// Save the contract
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, data)
}

// Fetch a single vote from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, voteTxId *protocol.TxId) (*state.Vote, error) {
	key := buildStoragePath(contractPKH, voteTxId)

	data, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	// Prepare the vote object
	result := state.Vote{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// List all votes for a specified contract.
func List(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash) ([]*state.Vote, error) {

	data, err := dbConn.List(ctx, fmt.Sprintf("%s/%s/%s", storageKey, contractPKH.String(), storageSubKey))
	if err != nil {
		return nil, err
	}

	result := make([]*state.Vote, 0, len(data))
	for _, b := range data {
		vote := state.Vote{}

		if err := json.Unmarshal(b, &vote); err != nil {
			return nil, err
		}

		result = append(result, &vote)
	}

	return result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, txid *protocol.TxId) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractPKH.String(), storageSubKey, txid.String())
}
