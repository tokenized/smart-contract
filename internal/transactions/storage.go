package transactions

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
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

	logger.Verbose(ctx, "Adding tx : %x", itx.Hash[:])
	return masterDb.Put(ctx, buildStoragePath(&itx.Hash), buf.Bytes())
}

func GetTx(ctx context.Context, masterDb *db.DB, txid *chainhash.Hash, netParams *chaincfg.Params, isTest bool) (*inspector.Transaction, error) {
	data, err := masterDb.Fetch(ctx, buildStoragePath(txid))
	if err != nil {
		if err == db.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	buf := bytes.NewBuffer(data)
	result := inspector.Transaction{}
	if err := result.Read(buf, netParams, isTest); err != nil {
		return nil, err
	}

	return &result, nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(txid *chainhash.Hash) string {
	return fmt.Sprintf("%s/%x", storageKey, txid[:])
}
