package utxos

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/tokenized/smart-contract/internal/platform/db"
)

const (
	storageKey = "utxos"
)

func (us *UTXOs) Save(ctx context.Context, masterDb *db.DB) error {
	var buf bytes.Buffer

	count := uint32(len(us.list))
	if err := binary.Write(&buf, binary.BigEndian, &count); err != nil {
		return err
	}

	for _, utxo := range us.list {
		if err := utxo.Write(&buf); err != nil {
			return err
		}
	}

	return masterDb.Put(ctx, storageKey, buf.Bytes())
}

func Load(ctx context.Context, masterDb *db.DB) (*UTXOs, error) {
	result := UTXOs{}
	data, err := masterDb.Fetch(ctx, storageKey)
	if err != nil {
		if err == db.ErrNotFound {
			return &result, nil // No UTXOs yet
		}
		return nil, err
	}

	buf := bytes.NewBuffer(data)

	var count uint32
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return nil, err
	}

	result.list = make([]*UTXO, count)
	for i, _ := range result.list {
		utxo := UTXO{}
		result.list[i] = &utxo
		if err := utxo.Read(buf); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func (utxo *UTXO) Write(buf *bytes.Buffer) error {
	if _, err := buf.Write(utxo.OutPoint.Hash[:]); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &utxo.OutPoint.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &utxo.Output.Value); err != nil {
		return err
	}

	scriptSize := uint32(len(utxo.Output.PkScript))
	if err := binary.Write(buf, binary.BigEndian, &scriptSize); err != nil {
		return err
	}

	if _, err := buf.Write(utxo.Output.PkScript); err != nil {
		return err
	}

	if _, err := buf.Write(utxo.SpentBy[:]); err != nil {
		return err
	}

	return nil
}

func (utxo *UTXO) Read(buf *bytes.Buffer) error {
	if _, err := buf.Read(utxo.OutPoint.Hash[:]); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &utxo.OutPoint.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &utxo.Output.Value); err != nil {
		return err
	}

	var scriptSize uint32
	if err := binary.Read(buf, binary.BigEndian, &scriptSize); err != nil {
		return err
	}

	utxo.Output.PkScript = make([]byte, int(scriptSize))
	if _, err := buf.Read(utxo.Output.PkScript); err != nil {
		return err
	}

	if _, err := buf.Read(utxo.SpentBy[:]); err != nil {
		return err
	}

	return nil
}
