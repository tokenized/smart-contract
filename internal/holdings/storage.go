package holdings

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
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

var cache map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate
var cacheLock sync.Mutex

// Save puts a single holding in cache. A CacheItem is returned and should be put in a CacheChannel
//   to be written to storage asynchronously, or be synchronously written to storage by immediately
//   calling Write.
func Save(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20, h *state.Holding) (*CacheItem, error) {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
	}
	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "contract hash")
	}
	contract, exists := cache[*contractHash]
	if !exists {
		nc := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
		cache[*contractHash] = &nc
		contract = &nc
	}
	instrument, exists := (*contract)[*instrumentCode]
	if !exists {
		na := make(map[bitcoin.Hash20]*cacheUpdate)
		(*contract)[*instrumentCode] = &na
		instrument = &na
	}

	addressHash, err := h.Address.Hash()
	if err != nil {
		return nil, err
	}
	cu, exists := (*instrument)[*addressHash]

	if exists {
		cu.lock.Lock()
		cu.h = h
		cu.modified = true
		cu.lock.Unlock()
	} else {
		(*instrument)[*addressHash] = &cacheUpdate{h: h, modified: true}
	}

	return NewCacheItem(contractHash, instrumentCode, addressHash), nil
}

// List provides a list of all holdings in storage for a specified instrument.
func List(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20) ([]string, error) {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "contract hash")
	}

	// Merge cache and storage data
	path := fmt.Sprintf("%s/%s/%s/%s",
		storageKey,
		contractHash.String(),
		storageSubKey,
		instrumentCode.String())
	result := make([]string, 0)
	resultKeys := make(map[string]bool)

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
	}
	contract, exists := cache[*contractHash]
	if !exists {
		nc := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
		cache[*contractHash] = &nc
		contract = &nc
	}
	instrument, exists := (*contract)[*instrumentCode]
	if !exists {
		na := make(map[bitcoin.Hash20]*cacheUpdate)
		(*contract)[*instrumentCode] = &na
		instrument = &na
	}

	for addressHash, _ := range *instrument {
		key := path + "/" + addressHash.String()
		result = append(result, key)
		resultKeys[key] = true
	}

	// Add storage
	keys, err := dbConn.List(ctx, path)
	if err != nil {
		return nil, errors.Wrap(err, "db list")
	}

	for _, key := range keys {
		_, exists := resultKeys[key]
		if !exists {
			result = append(result, key)
		}
	}

	return result, nil
}

// FetchAll fetches all holdings from storage for a specified instrument.
func FetchAll(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20) ([]*state.Holding, error) {

	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "contract hash")
	}

	// Merge cache and storage data
	path := fmt.Sprintf("%s/%s/%s/%s",
		storageKey,
		contractHash.String(),
		storageSubKey,
		instrumentCode.String())
	result := make([]*state.Holding, 0)
	resultKeys := make(map[string]bool)

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
	}
	contract, exists := cache[*contractHash]
	if !exists {
		nc := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
		cache[*contractHash] = &nc
		contract = &nc
	}
	instrument, exists := (*contract)[*instrumentCode]
	if !exists {
		na := make(map[bitcoin.Hash20]*cacheUpdate)
		(*contract)[*instrumentCode] = &na
		instrument = &na
	}

	for addressHash, h := range *instrument {
		key := path + "/" + addressHash.String()
		result = append(result, h.h)
		resultKeys[key] = true
	}

	// Add storage
	keys, err := dbConn.List(ctx, path)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		_, exists := resultKeys[key]
		if !exists {
			b, err := dbConn.Fetch(ctx, key)
			if err != nil {
				if err == db.ErrNotFound {
					return nil, ErrNotFound
				}

				return nil, errors.Wrap(err, "Failed to fetch holding")
			}

			// Prepare the instrument object
			readResult, err := deserializeHolding(bytes.NewReader(b))
			if err != nil {
				return nil, errors.Wrap(err, "Failed to deserialize holding")
			}

			result = append(result, readResult)
		}
	}

	return result, nil
}

// Fetch fetches a single holding from storage and places it in the cache.
func Fetch(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20, address bitcoin.RawAddress) (*state.Holding, error) {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
	}
	contractHash, err := contractAddress.Hash()
	if err != nil {
		return nil, errors.Wrap(err, "contract hash")
	}
	contract, exists := cache[*contractHash]
	if !exists {
		nc := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
		cache[*contractHash] = &nc
		contract = &nc
	}
	instrument, exists := (*contract)[*instrumentCode]
	if !exists {
		na := make(map[bitcoin.Hash20]*cacheUpdate)
		(*contract)[*instrumentCode] = &na
		instrument = &na
	}
	addressHash, err := address.Hash()
	if err != nil {
		return nil, err
	}
	cu, exists := (*instrument)[*addressHash]
	if exists {
		// Copy so the object in cache will not be unintentionally modified (by reference)
		// We don't want it to be modified unless Save is called.
		cu.lock.Lock()
		defer cu.lock.Unlock()
		return copyHolding(cu.h), nil
	}

	key := buildStoragePath(contractHash, instrumentCode, addressHash)

	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "Failed to fetch holding")
	}

	// Prepare the instrument object
	readResult, err := deserializeHolding(bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to deserialize holding")
	}

	(*instrument)[*addressHash] = &cacheUpdate{h: readResult, modified: false}

	return copyHolding(readResult), nil
}

// ProcessCacheItems waits for items on the cache channel and writes them to storage. It exits when
//   the channel is closed.
func ProcessCacheItems(ctx context.Context, dbConn *db.DB, ch *CacheChannel) error {
	for ci := range ch.Channel {
		if err := ci.Write(ctx, dbConn); err != nil && err != ErrNotInCache {
			return err
		}
	}

	return nil
}

func copyHolding(h *state.Holding) *state.Holding {
	result := *h
	result.HoldingStatuses = make(map[bitcoin.Hash32]*state.HoldingStatus)
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

	for contractHash, instruments := range cache {
		for instrumentCode, instrumentHoldings := range *instruments {
			for addressHash, cu := range *instrumentHoldings {
				cu.lock.Lock()
				if cu.modified {
					if err := write(ctx, dbConn, &contractHash, &instrumentCode, &addressHash,
						cu.h); err != nil {
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

// WriteCacheUpdate updates storage for an item from the cache if it has been modified since the
// last write.
func WriteCacheUpdate(ctx context.Context, dbConn *db.DB, contractHash *bitcoin.Hash20,
	instrumentCode *bitcoin.Hash20, addressHash *bitcoin.Hash20) error {

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if cache == nil {
		cache = make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
	}
	contract, exists := cache[*contractHash]
	if !exists {
		nc := make(map[bitcoin.Hash20]*map[bitcoin.Hash20]*cacheUpdate)
		cache[*contractHash] = &nc
		contract = &nc
	}
	instrument, exists := (*contract)[*instrumentCode]
	if !exists {
		na := make(map[bitcoin.Hash20]*cacheUpdate)
		(*contract)[*instrumentCode] = &na
		instrument = &na
	}
	cu, exists := (*instrument)[*addressHash]
	if !exists {
		return ErrNotInCache
	}

	cu.lock.Lock()
	defer cu.lock.Unlock()

	if !cu.modified {
		return nil
	}

	if err := write(ctx, dbConn, contractHash, instrumentCode, addressHash, cu.h); err != nil {
		return err
	}

	cu.modified = false
	return nil
}

func write(ctx context.Context, dbConn *db.DB, contractHash *bitcoin.Hash20,
	instrumentCode *bitcoin.Hash20, addressHash *bitcoin.Hash20, h *state.Holding) error {

	data, err := serializeHolding(h)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize holding")
	}

	if err := dbConn.Put(ctx, buildStoragePath(contractHash, instrumentCode, addressHash),
		data); err != nil {
		return err
	}

	return nil
}

// Returns the storage path prefix for a given identifier.
func buildStoragePath(contractHash *bitcoin.Hash20, instrumentCode *bitcoin.Hash20,
	addressHash *bitcoin.Hash20) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", storageKey, contractHash.String(), storageSubKey,
		instrumentCode.String(), addressHash.String())
}

func serializeHolding(h *state.Holding) ([]byte, error) {
	var buf bytes.Buffer

	// Version
	if err := binary.Write(&buf, binary.LittleEndian, uint8(0)); err != nil {
		return nil, err
	}

	if err := h.Address.Serialize(&buf); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, h.PendingBalance); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, h.FinalizedBalance); err != nil {
		return nil, err
	}

	if err := h.CreatedAt.Serialize(&buf); err != nil {
		return nil, err
	}
	if err := h.UpdatedAt.Serialize(&buf); err != nil {
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

func serializeHoldingStatus(w io.Writer, hs *state.HoldingStatus) error {
	if err := binary.Write(w, binary.LittleEndian, hs.Code); err != nil {
		return err
	}

	if err := hs.Expires.Serialize(w); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, hs.Amount); err != nil {
		return err
	}

	if err := hs.TxId.Serialize(w); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, hs.SettleQuantity); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, hs.Posted); err != nil {
		return err
	}

	return nil
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

func deserializeHolding(r io.Reader) (*state.Holding, error) {
	var result state.Holding

	// Version
	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return &result, err
	}
	if version != 0 {
		return &result, fmt.Errorf("Unknown version : %d", version)
	}

	err := result.Address.Deserialize(r)
	if err != nil {
		return &result, err
	}

	if err := binary.Read(r, binary.LittleEndian, &result.PendingBalance); err != nil {
		return &result, err
	}
	if err := binary.Read(r, binary.LittleEndian, &result.FinalizedBalance); err != nil {
		return &result, err
	}
	result.CreatedAt, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return &result, err
	}
	result.UpdatedAt, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return &result, err
	}

	result.HoldingStatuses = make(map[bitcoin.Hash32]*state.HoldingStatus)
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return &result, err
	}
	for i := 0; i < int(length); i++ {
		var hs state.HoldingStatus
		if err := deserializeHoldingStatus(r, &hs); err != nil {
			return &result, err
		}
		result.HoldingStatuses[*hs.TxId] = &hs
	}

	return &result, nil
}

func deserializeHoldingStatus(r io.Reader, hs *state.HoldingStatus) error {
	if err := binary.Read(r, binary.LittleEndian, &hs.Code); err != nil {
		return err
	}

	var err error
	hs.Expires, err = protocol.DeserializeTimestamp(r)
	if err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &hs.Amount); err != nil {
		return err
	}

	hs.TxId = &bitcoin.Hash32{}
	if err := hs.TxId.Deserialize(r); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &hs.SettleQuantity); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &hs.Posted); err != nil {
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
