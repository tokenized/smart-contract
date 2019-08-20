package holdings

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// CacheItem is a reference to an item in the cache that needs to be written to storage.
type CacheItem struct {
	contract bitcoin.RawAddress
	asset    *protocol.AssetCode
	address  bitcoin.RawAddress
}

// NewCacheItem creates a new CacheItem.
func NewCacheItem(contract bitcoin.RawAddress, asset *protocol.AssetCode,
	address bitcoin.RawAddress) *CacheItem {
	result := CacheItem{
		contract: contract,
		asset:    asset,
		address:  address,
	}
	return &result
}

// Write writes a cache item to storage.
func (ci *CacheItem) Write(ctx context.Context, dbConn *db.DB) error {
	return WriteCacheUpdate(ctx, dbConn, ci.contract, ci.asset, ci.address)
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
