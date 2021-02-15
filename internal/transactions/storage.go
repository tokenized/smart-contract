package transactions

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/spynode/pkg/client"
)

const (
	storageKey      = "txs"
	storageStateKey = "txstates"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Transaction not found")
)

func AddTx(ctx context.Context, masterDb *db.DB, itx *inspector.Transaction) error {
	var buf bytes.Buffer
	if err := itx.Write(&buf); err != nil {
		return err
	}

	logger.Verbose(ctx, "Adding tx : %s", itx.Hash)
	return masterDb.Put(ctx, buildStoragePath(itx.Hash), buf.Bytes())
}

func GetTx(ctx context.Context, masterDb *db.DB, txid *bitcoin.Hash32, isTest bool) (*inspector.Transaction, error) {
	data, err := masterDb.Fetch(ctx, buildStoragePath(txid))
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	buf := bytes.NewReader(data)
	result := inspector.Transaction{}
	if err := result.Read(buf, isTest); err != nil {
		return nil, err
	}

	return &result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(txid *bitcoin.Hash32) string {
	return fmt.Sprintf("%s/%s", storageKey, txid.String())
}

func AddTxState(ctx context.Context, masterDb *db.DB, txid *bitcoin.Hash32,
	state *client.TxState) error {

	var buf bytes.Buffer
	if err := state.Serialize(&buf); err != nil {
		return err
	}

	logger.Verbose(ctx, "Adding tx state : %s", txid)
	return masterDb.Put(ctx, buildStateStoragePath(txid), buf.Bytes())
}

func GetTxState(ctx context.Context, masterDb *db.DB,
	txid *bitcoin.Hash32) (*client.TxState, error) {

	data, err := masterDb.Fetch(ctx, buildStateStoragePath(txid))
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	buf := bytes.NewReader(data)
	result := &client.TxState{}
	if err := result.Deserialize(buf); err != nil {
		return nil, err
	}

	return result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStateStoragePath(txid *bitcoin.Hash32) string {
	return fmt.Sprintf("%s/%s", storageStateKey, txid)
}
