package instrument

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const storageKey = "contracts"
const storageSubKey = "assets" // Leave assets for backwards compatibility.

var cache map[bitcoin.Hash20]*state.Instrument

// Put a single instrument in storage
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrument *state.Instrument) error {

	data, err := serializeInstrument(instrument)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize instrument")
	}

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return err
	}
	if err := dbConn.Put(ctx, buildStoragePath(contractHash, instrument.Code), data); err != nil {
		return err
	}

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Instrument)
	}
	cache[*instrument.Code] = instrument
	return nil
}

// Fetch a single instrument from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20) (*state.Instrument, error) {
	if cache != nil {
		result, exists := cache[*instrumentCode]
		if exists {
			return result, nil
		}
	}

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}
	key := buildStoragePath(contractHash, instrumentCode)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch instrument")
	}

	// Prepare the instrument object
	instrument := state.Instrument{}
	if err := deserializeInstrument(bytes.NewReader(b), &instrument); err != nil {
		return nil, errors.Wrap(err, "Failed to deserialize instrument")
	}

	return &instrument, nil
}

func Reset(ctx context.Context) {
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20, instrument *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractHash, storageSubKey, instrument)
}

func serializeInstrument(as *state.Instrument) ([]byte, error) {
	var buf bytes.Buffer

	// Version
	if err := binary.Write(&buf, binary.LittleEndian, uint8(0)); err != nil {
		return nil, err
	}

	if err := as.Code.Serialize(&buf); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, as.Revision); err != nil {
		return nil, err
	}

	if err := as.CreatedAt.Serialize(&buf); err != nil {
		return nil, err
	}

	if err := as.UpdatedAt.Serialize(&buf); err != nil {
		return nil, err
	}

	if err := as.Timestamp.Serialize(&buf); err != nil {
		return nil, err
	}

	if err := serializeString(&buf, []byte(as.InstrumentType)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.InstrumentIndex); err != nil {
		return nil, err
	}
	if err := serializeString(&buf, as.InstrumentPermissions); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(as.TradeRestrictions))); err != nil {
		return nil, err
	}
	var tr [3]byte
	for _, rest := range as.TradeRestrictions {
		copy(tr[:], []byte(rest))
		if _, err := buf.Write(tr[:]); err != nil {
			return nil, err
		}
	}

	if err := binary.Write(&buf, binary.LittleEndian, as.EnforcementOrdersPermitted); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.VotingRights); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.VoteMultiplier); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AdministrationProposal); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.HolderProposal); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.InstrumentModificationGovernance); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AuthorizedTokenQty); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AdministrationProposal); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AdministrationProposal); err != nil {
		return nil, err
	}
	if err := serializeString(&buf, as.InstrumentPayload); err != nil {
		return nil, err
	}

	if err := as.FreezePeriod.Serialize(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeString(w io.Writer, v []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(v))); err != nil {
		return err
	}
	if _, err := w.Write(v); err != nil {
		return err
	}
	return nil
}

func deserializeInstrument(r io.Reader, as *state.Instrument) error {
	// Version
	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("Unknown version : %d", version)
	}

	as.Code = &bitcoin.Hash20{}
	if err := as.Code.Deserialize(r); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.Revision); err != nil {
		return err
	}

	var err error
	as.CreatedAt, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return err
	}
	as.UpdatedAt, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return err
	}
	as.Timestamp, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return err
	}
	data, err := deserializeString(r)
	if err != nil {
		return err
	}
	as.InstrumentType = string(data)
	if err := binary.Read(r, binary.LittleEndian, &as.InstrumentIndex); err != nil {
		return err
	}
	as.InstrumentPermissions, err = deserializeString(r)
	if err != nil {
		return err
	}

	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return err
	}
	for i := 0; i < int(length); i++ {
		var rest [3]byte
		if _, err := r.Read(rest[:]); err != nil {
			return err
		}
		as.TradeRestrictions = append(as.TradeRestrictions, string(rest[:]))
	}

	if err := binary.Read(r, binary.LittleEndian, &as.EnforcementOrdersPermitted); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.VotingRights); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.VoteMultiplier); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.HolderProposal); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.InstrumentModificationGovernance); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.AuthorizedTokenQty); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	as.InstrumentPayload, err = deserializeString(r)
	if err != nil {
		return err
	}
	as.FreezePeriod, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return err
	}

	return nil
}

func deserializeString(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	result := make([]byte, length)
	if _, err := r.Read(result); err != nil {
		return nil, err
	}
	return result, nil
}
