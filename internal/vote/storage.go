package vote

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/protocol"
)

const storageKey = "contracts"
const storageSubKey = "votes"

// Put a single vote in storage
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress, v *state.Vote) error {
	contractHash, err := contractAddress.Hash()
	if err != nil {
		return err
	}
	key := buildStoragePath(contractHash, v.VoteTxId)

	// Save the contract
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, data)
}

// Fetch a single vote from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteTxId *protocol.TxId) (*state.Vote, error) {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}
	key := buildStoragePath(contractHash, voteTxId)

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
func List(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress) ([]*state.Vote, error) {
	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}

	// TODO: This should probably use dbConn.List for greater efficiency
	data, err := dbConn.Search(ctx, fmt.Sprintf("%s/%s/%s", storageKey, contractHash.String(),
		storageSubKey))
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
func buildStoragePath(contractHash *bitcoin.Hash20, txid *protocol.TxId) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractHash.String(), storageSubKey, txid.String())
}
