package state

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"

	"github.com/pkg/errors"
)

const (
	stateStorageKey = "chainstate"
	stateVersion    = uint8(0)
)

func GetNextMessageID(ctx context.Context, dbConn *db.DB) (*uint64, error) {
	b, err := dbConn.Fetch(ctx, stateStorageKey)
	if err != nil {
		if errors.Cause(err) == db.ErrNotFound {
			result := uint64(1)
			return &result, nil
		}
		return nil, errors.Wrap(err, "fetch")
	}

	r := bytes.NewReader(b)

	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, errors.Wrap(err, "version")
	}

	if version != 0 {
		return nil, errors.New("Wrong version")
	}

	var result uint64
	if err := binary.Read(r, binary.LittleEndian, &result); err != nil {
		return nil, errors.Wrap(err, "next message id")
	}

	return &result, nil
}

func SaveNextMessageID(ctx context.Context, dbConn *db.DB, nextMessageID uint64) error {
	node.Log(ctx, "Saving Spynode next message ID : %d", nextMessageID)

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, stateVersion); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := binary.Write(&buf, binary.LittleEndian, nextMessageID); err != nil {
		return errors.Wrap(err, "next message id")
	}

	if err := dbConn.Put(ctx, stateStorageKey, buf.Bytes()); err != nil {
		return errors.Wrap(err, "put")
	}

	return nil
}
