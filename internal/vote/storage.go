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

	// Prepare the contract object
	result := state.Vote{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, txid *protocol.TxId) string {
	return fmt.Sprintf("%v/%x/%x", storageKey, contractPKH.String(), txid.String())
}
