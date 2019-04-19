package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"
)

const storageKey = "contracts"
const storageSubKey = "transfers"

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Pending transfer not found")
)

// Put a single pending transfer in storage
func Save(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, t *state.PendingTransfer) error {
	key := buildStoragePath(contractPKH, &t.TransferTxId)

	// Save the contract
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	return dbConn.Put(ctx, key, data)
}

// Fetch a single pending transfer from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, transferTxId *protocol.TxId) (*state.PendingTransfer, error) {
	key := buildStoragePath(contractPKH, transferTxId)

	data, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	// Prepare the pending transfer object
	result := state.PendingTransfer{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func Remove(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, transferTxId *protocol.TxId) error {
	err := dbConn.Remove(ctx, buildStoragePath(contractPKH, transferTxId))
	if err != nil {
		if err == db.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// List all pending transfer for a specified contract.
func List(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash) ([]*state.PendingTransfer, error) {

	data, err := dbConn.List(ctx, fmt.Sprintf("%s/%s/%s", storageKey, contractPKH.String(), storageSubKey))
	if err != nil {
		return nil, err
	}

	result := make([]*state.PendingTransfer, 0, len(data))
	for _, b := range data {
		pendingTransfer := state.PendingTransfer{}

		if err := json.Unmarshal(b, &pendingTransfer); err != nil {
			return nil, err
		}

		result = append(result, &pendingTransfer)
	}

	return result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, txid *protocol.TxId) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractPKH.String(), storageSubKey, txid.String())
}
