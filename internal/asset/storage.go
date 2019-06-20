package asset

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const storageKey = "contracts"
const storageSubKey = "assets"

var cache map[protocol.AssetCode]*state.Asset

// Put a single asset in storage
func Save(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, asset *state.Asset) error {
	data, err := serializeAsset(asset)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize asset")
	}

	if err := dbConn.Put(ctx, buildStoragePath(contractPKH, &asset.ID), data); err != nil {
		return err
	}

	if cache == nil {
		cache = make(map[protocol.AssetCode]*state.Asset)
	}
	cache[asset.ID] = asset
	return nil
}

// Fetch a single asset from storage
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode) (*state.Asset, error) {
	if cache != nil {
		result, exists := cache[*assetCode]
		if exists {
			return result, nil
		}
	}

	key := buildStoragePath(contractPKH, assetCode)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch asset")
	}

	// Prepare the asset object
	asset := state.Asset{}
	if err := deserializeAsset(bytes.NewBuffer(b), &asset); err != nil {
		return nil, errors.Wrap(err, "Failed to deserialize asset")
	}

	return &asset, nil
}

func Reset(ctx context.Context) {
	cache = nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, asset *protocol.AssetCode) string {
	return fmt.Sprintf("%s/%s/%s/%s", storageKey, contractPKH.String(), storageSubKey, asset.String())
}

func serializeAsset(as *state.Asset) ([]byte, error) {
	var buf bytes.Buffer

	// Version
	if err := binary.Write(&buf, binary.LittleEndian, uint8(0)); err != nil {
		return nil, err
	}

	data, err := as.ID.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, as.Revision); err != nil {
		return nil, err
	}

	data, err = as.CreatedAt.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	data, err = as.UpdatedAt.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	data, err = as.Timestamp.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	if err := serializeString(&buf, []byte(as.AssetType)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.AssetIndex); err != nil {
		return nil, err
	}
	if err := serializeString(&buf, as.AssetAuthFlags); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, as.TransfersPermitted); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(as.TradeRestrictions))); err != nil {
		return nil, err
	}
	for _, rest := range as.TradeRestrictions {
		if err := binary.Write(&buf, binary.LittleEndian, rest); err != nil {
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
	if err := binary.Write(&buf, binary.LittleEndian, as.TokenQty); err != nil {
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

	data, err = as.FreezePeriod.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func serializeString(buf *bytes.Buffer, v []byte) error {
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(v))); err != nil {
		return err
	}
	if _, err := buf.Write(v); err != nil {
		return err
	}
	return nil
}

func deserializeAsset(buf *bytes.Buffer, as *state.Asset) error {
	// Version
	var version uint8
	if err := binary.Read(buf, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("Unknown version : %d", version)
	}
	if err := as.ID.Write(buf); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.Revision); err != nil {
		return err
	}
	if err := as.CreatedAt.Write(buf); err != nil {
		return err
	}
	if err := as.UpdatedAt.Write(buf); err != nil {
		return err
	}
	if err := as.Timestamp.Write(buf); err != nil {
		return err
	}
	data, err := deserializeString(buf)
	if err != nil {
		return err
	}
	as.AssetType = string(data)
	if err := binary.Read(buf, binary.LittleEndian, &as.AssetIndex); err != nil {
		return err
	}
	as.AssetAuthFlags, err = deserializeString(buf)
	if err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.TransfersPermitted); err != nil {
		return err
	}

	var length uint32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}
	for i := 0; i < int(length); i++ {
		var rest [3]byte
		if _, err := buf.Read(rest[:]); err != nil {
			return err
		}
		as.TradeRestrictions = append(as.TradeRestrictions, rest)
	}

	if err := binary.Read(buf, binary.LittleEndian, &as.EnforcementOrdersPermitted); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.VotingRights); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.VoteMultiplier); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.HolderProposal); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.AssetModificationGovernance); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.TokenQty); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &as.AdministrationProposal); err != nil {
		return err
	}
	as.AssetPayload, err = deserializeString(buf)
	if err != nil {
		return err
	}
	if err := as.FreezePeriod.Write(buf); err != nil {
		return err
	}

	return nil
}

func deserializeString(buf *bytes.Buffer) ([]byte, error) {
	var length uint32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	result := make([]byte, length)
	if _, err := buf.Read(result); err != nil {
		return nil, err
	}
	return result, nil
}
