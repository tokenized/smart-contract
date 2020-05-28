package vote

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const storageKey = "contracts"
const storageSubKey = "votes"

var cache []state.Vote
var cacheLock sync.Mutex

// Put a single vote in storage
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress, v *state.Vote) error {

	found := false
	cacheLock.Lock()
	for i, cv := range cache {
		if cv.VoteTxId.Equal(*v.VoteTxId) {
			found = true
			cache[i] = *v
			break
		}
	}
	if !found {
		cache = append(cache, *v)
	}
	cacheLock.Unlock()

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return errors.Wrap(err, "contract address hash")
	}
	key := buildStoragePath(contractHash, v.VoteTxId)

	// Update ballot list
	v.BallotList = make([]state.Ballot, 0, len(v.Ballots))
	for _, b := range v.Ballots {
		v.BallotList = append(v.BallotList, b)
	}

	// Save the contract
	data, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "json marshal vote")
	}
	v.BallotList = nil // Clear to save memory

	return dbConn.Put(ctx, key, data)
}

// Fetch a single vote from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteTxId *protocol.TxId) (*state.Vote, error) {

	cacheLock.Lock()
	for _, cv := range cache {
		if cv.VoteTxId.Equal(*voteTxId) {
			result := cv
			cacheLock.Unlock()
			return &result, nil
		}
	}
	cacheLock.Unlock()

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

	// Update ballot map
	result.Ballots = make(map[bitcoin.Hash20]state.Ballot)
	for _, b := range result.BallotList {
		hash, err := b.Address.Hash()
		if err != nil {
			return nil, errors.Wrap(err, "address hash")
		}
		result.Ballots[*hash] = b
	}
	result.BallotList = nil // Clear to save memory

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

func Reset(ctx context.Context) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20, txid *protocol.TxId) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractHash.String(), storageSubKey, txid.String())
}
