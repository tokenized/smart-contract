package bitcoin

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type UTXO struct {
	Hash          Hash32 `db:"hash" json:"hash"`
	Index         uint32 `db:"index" json:"index"`
	Value         uint64 `db:"value" json:"value"`
	LockingScript []byte `db:"locking_script" json:"locking_script"`

	// Optional identifier for external use to track the key needed to spend the UTXO.
	KeyID string `json:"key_id,omitempty"`
}

func (u UTXO) ID() string {
	return fmt.Sprintf("%s:%d", u.Hash.String(), u.Index)
}

func (u UTXO) Address() (RawAddress, error) {
	return RawAddressFromLockingScript(u.LockingScript)
}

func (utxo UTXO) Write(buf *bytes.Buffer) error {
	if err := utxo.Hash.Serialize(buf); err != nil {
		return err
	}

	scriptSize := uint32(len(utxo.LockingScript))
	if err := binary.Write(buf, binary.LittleEndian, &scriptSize); err != nil {
		return err
	}

	if _, err := buf.Write(utxo.LockingScript); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}

func (utxo *UTXO) Read(buf *bytes.Reader) error {
	hash, err := DeserializeHash32(buf)
	if err != nil {
		return err
	}
	utxo.Hash = *hash

	var scriptSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &scriptSize); err != nil {
		return err
	}

	utxo.LockingScript = make([]byte, int(scriptSize))
	if _, err := buf.Read(utxo.LockingScript); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.LittleEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.LittleEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}
