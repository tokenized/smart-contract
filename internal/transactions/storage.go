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
)

const (
	storageKey = "txs"
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

	logger.Verbose(ctx, "Adding tx : %s", itx.Hash.String())
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
