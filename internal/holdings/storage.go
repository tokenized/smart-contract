package holdings

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var (
	ErrNotInCache = errors.New("Not in cache")
)

// Options
//   Periodically write holdings that haven't been written for x seconds
//   Build deltas and only write deltas
//     holding statuses are fixed size
//

const storageKey = "contracts"
const storageSubKey = "holdings"

type cacheUpdate struct {
	h        *state.Holding
	modified bool // true when modified since last write to storage.
	lock     sync.Mutex
}

var cache map[protocol.PublicKeyHash]map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate
var cacheLock sync.Mutex

// Save puts a single holding in cache. A CacheItem is returned and should be put in a CacheChannel
//   to be written to storage asynchronously, or be synchronously written to storage by immediately
//   calling Write.
func Save(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode, h *state.Holding) (*CacheItem, error) {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[protocol.PublicKeyHash]map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
	}
	contract, exists := cache[*contractPKH]
	if !exists {
		contract = make(map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
		cache[*contractPKH] = contract
	}
	asset, exists := cache[*contractPKH][*assetCode]
	if !exists {
		asset = make(map[protocol.PublicKeyHash]*cacheUpdate)
		cache[*contractPKH][*assetCode] = asset
	}

	cu, exists := asset[h.PKH]

	if exists {
		cu.lock.Lock()
		cu.h = h
		cu.modified = true
		cu.lock.Unlock()
	} else {
		asset[h.PKH] = &cacheUpdate{h: h, modified: true}
	}

	return NewCacheItem(contractPKH, assetCode, &h.PKH), nil
}

// List provides a list of all holdings in storage for a specified asset.
func List(ctx context.Context,
	dbConn *db.DB,
	contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode) ([]string, error) {

	path := fmt.Sprintf("%s/%s/%s/%s",
		storageKey,
		contractPKH.String(),
		storageSubKey,
		assetCode.String())

	return dbConn.List(ctx, path)
}

// Fetch fetches a single holding from storage and places it in the cache.
func Fetch(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode, pkh *protocol.PublicKeyHash) (*state.Holding, error) {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[protocol.PublicKeyHash]map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
	}
	contract, exists := cache[*contractPKH]
	if !exists {
		contract = make(map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
		cache[*contractPKH] = contract
	}
	asset, exists := contract[*assetCode]
	if !exists {
		asset = make(map[protocol.PublicKeyHash]*cacheUpdate)
		contract[*assetCode] = asset
	}
	cu, exists := asset[*pkh]
	if exists {
		// Copy so the object in cache will not be unintentionally modified (by reference)
		// We don't want it to be modified unless Save is called.
		cu.lock.Lock()
		defer cu.lock.Unlock()
		return copyHolding(cu.h), nil
	}

	key := buildStoragePath(contractPKH, assetCode, pkh)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch holding")
	}

	// Prepare the asset object
	readResult, err := deserializeHolding(bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to deserialize holding")
	}

	asset[*pkh] = &cacheUpdate{h: readResult, modified: false}

	return readResult, nil
}

func copyHolding(h *state.Holding) *state.Holding {
	result := *h
	result.HoldingStatuses = make(map[protocol.TxId]*state.HoldingStatus)
	for key, val := range h.HoldingStatuses {
		result.HoldingStatuses[key] = val
	}
	return &result
}

func Reset(ctx context.Context) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	cache = nil
}

func WriteCache(ctx context.Context, dbConn *db.DB) error {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		return nil
	}

	for contractPKH, assets := range cache {
		for assetCode, assetHoldings := range assets {
			for _, cu := range assetHoldings {
				cu.lock.Lock()
				if cu.modified {
					if err := write(ctx, dbConn, &contractPKH, &assetCode, cu.h); err != nil {
						cu.lock.Unlock()
						return err
					}
					cu.modified = false
				}
				cu.lock.Unlock()
			}
		}
	}
	return nil
}

// Write updates storage for an item from the cache if it has been modified since the last write.
func WriteCacheUpdate(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode, pkh *protocol.PublicKeyHash) error {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[protocol.PublicKeyHash]map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
	}
	contract, exists := cache[*contractPKH]
	if !exists {
		contract = make(map[protocol.AssetCode]map[protocol.PublicKeyHash]*cacheUpdate)
		cache[*contractPKH] = contract
	}
	asset, exists := contract[*assetCode]
	if !exists {
		asset = make(map[protocol.PublicKeyHash]*cacheUpdate)
		contract[*assetCode] = asset
	}
	cu, exists := asset[*pkh]
	if !exists {
		return ErrNotInCache
	}

	cu.lock.Lock()
	defer cu.lock.Unlock()

	if !cu.modified {
		return nil
	}

	if err := write(ctx, dbConn, contractPKH, assetCode, cu.h); err != nil {
		return err
	}

	cu.modified = false
	return nil
}

func write(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	assetCode *protocol.AssetCode, h *state.Holding) error {

	data, err := serializeHolding(h)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize holding")
	}

	if err := dbConn.Put(ctx, buildStoragePath(contractPKH, assetCode, &h.PKH), data); err != nil {
		return err
	}

	return nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractPKH *protocol.PublicKeyHash, assetCode *protocol.AssetCode,
	pkh *protocol.PublicKeyHash) string {

	return fmt.Sprintf("%s/%s/%s/%s/%s", storageKey, contractPKH.String(), storageSubKey,
		assetCode.String(), pkh.String())
}

func serializeHolding(h *state.Holding) ([]byte, error) {
	var buf bytes.Buffer

	// Version
	if err := binary.Write(&buf, binary.LittleEndian, uint8(0)); err != nil {
		return nil, err
	}

	data, err := h.PKH.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err = buf.Write(data); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, h.PendingBalance); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, h.FinalizedBalance); err != nil {
		return nil, err
	}
	data, err = h.CreatedAt.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}
	data, err = h.UpdatedAt.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(data); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(h.HoldingStatuses))); err != nil {
		return nil, err
	}

	for _, value := range h.HoldingStatuses {
		if err := serializeHoldingStatus(&buf, value); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func serializeHoldingStatus(buf *bytes.Buffer, hs *state.HoldingStatus) error {
	if err := binary.Write(buf, binary.LittleEndian, hs.Code); err != nil {
		return err
	}

	data, err := hs.Expires.Serialize()
	if err != nil {
		return err
	}
	if _, err := buf.Write(data); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, hs.Amount); err != nil {
		return err
	}

	data, err = hs.TxId.Serialize()
	if err != nil {
		return err
	}
	if _, err := buf.Write(data); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, hs.SettleQuantity); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, hs.Posted); err != nil {
		return err
	}

	return nil
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

func deserializeHolding(buf *bytes.Buffer) (*state.Holding, error) {
	var result state.Holding

	// Version
	var version uint8
	if err := binary.Read(buf, binary.LittleEndian, &version); err != nil {
		return &result, err
	}
	if version != 0 {
		return &result, fmt.Errorf("Unknown version : %d", version)
	}

	if err := result.PKH.Write(buf); err != nil {
		return &result, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &result.PendingBalance); err != nil {
		return &result, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &result.FinalizedBalance); err != nil {
		return &result, err
	}
	if err := result.CreatedAt.Write(buf); err != nil {
		return &result, err
	}
	if err := result.UpdatedAt.Write(buf); err != nil {
		return &result, err
	}

	result.HoldingStatuses = make(map[protocol.TxId]*state.HoldingStatus)
	var length uint32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return &result, err
	}
	for i := 0; i < int(length); i++ {
		var hs state.HoldingStatus
		if err := deserializeHoldingStatus(buf, &hs); err != nil {
			return &result, err
		}
		result.HoldingStatuses[hs.TxId] = &hs
	}

	return &result, nil
}

func deserializeHoldingStatus(buf *bytes.Buffer, hs *state.HoldingStatus) error {
	if err := binary.Read(buf, binary.LittleEndian, &hs.Code); err != nil {
		return err
	}

	if err := hs.Expires.Write(buf); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &hs.Amount); err != nil {
		return err
	}
	if err := hs.TxId.Write(buf); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &hs.SettleQuantity); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &hs.Posted); err != nil {
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
