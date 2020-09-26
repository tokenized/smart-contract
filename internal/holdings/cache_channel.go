package holdings

import (
	"context"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// CacheItem is a reference to an item in the cache that needs to be written to storage.
type CacheItem struct {
	contractHash *bitcoin.Hash20
	asset        *protocol.AssetCode
	addressHash  *bitcoin.Hash20
}

// NewCacheItem creates a new CacheItem.
func NewCacheItem(contractHash *bitcoin.Hash20, asset *protocol.AssetCode,
	addressHash *bitcoin.Hash20) *CacheItem {
	result := CacheItem{
		contractHash: contractHash,
		asset:        asset,
		addressHash:  addressHash,
	}
	return &result
}

// Write writes a cache item to storage.
func (ci *CacheItem) Write(ctx context.Context, dbConn *db.DB) error {
	return WriteCacheUpdate(ctx, dbConn, ci.contractHash, ci.asset, ci.addressHash)
}

// CacheChannel is a channel of items in cache waiting to be written to storage.
type CacheChannel struct {
	Channel chan *CacheItem
	lock    sync.Mutex
	open    bool
}

func (c *CacheChannel) Add(ci *CacheItem) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- ci
	return nil
}

func (c *CacheChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan *CacheItem, count)
	c.open = true
	return nil
}

func (c *CacheChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
