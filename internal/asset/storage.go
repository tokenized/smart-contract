package asset

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
const storageSubKey = "assets"

var cache map[bitcoin.Hash20]*state.Asset

// Put a single asset in storage
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	asset *state.Asset) error {

	data, err := serializeAsset(asset)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize asset")
	}

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return err
	}
	if err := dbConn.Put(ctx, buildStoragePath(contractHash, asset.Code), data); err != nil {
		return err
	}

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*state.Asset)
	}
	cache[*asset.Code] = asset
	return nil
}

// Fetch a single asset from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	assetCode *bitcoin.Hash20) (*state.Asset, error) {
	if cache != nil {
		result, exists := cache[*assetCode]
		if exists {
			return result, nil
		}
	}

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, err
	}
	key := buildStoragePath(contractHash, assetCode)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch asset")
	}

	// Prepare the asset object
	asset := state.Asset{}
	if err := deserializeAsset(bytes.NewReader(b), &asset); err != nil {
		return nil, errors.Wrap(err, "Failed to deserialize asset")
	}

	return &asset, nil
}

func Reset(ctx context.Context) {
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20, asset *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractHash, storageSubKey, asset)
}

func serializeAsset(as *state.Asset) ([]byte, error) {
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

	if err := serializeString(&buf, []byte(as.AssetType)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AssetIndex); err != nil {
		return nil, err
	}
	if err := serializeString(&buf, as.AssetPermissions); err != nil {
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
	if err := binary.Write(&buf, binary.LittleEndian, as.AssetModificationGovernance); err != nil {
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
	if err := serializeString(&buf, as.AssetPayload); err != nil {
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

func deserializeAsset(r io.Reader, as *state.Asset) error {
	// Version
	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("Unknown version : %d", version)
	}

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
	as.AssetType = string(data)
	if err := binary.Read(r, binary.LittleEndian, &as.AssetIndex); err != nil {
		return err
	}
	as.AssetPermissions, err = deserializeString(r)
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
	if err := binary.Read(r, binary.LittleEndian, &as.AssetModificationGovernance); err != nil {
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
	as.AssetPayload, err = deserializeString(r)
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
