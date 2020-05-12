package db

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var (
	// ErrInvalidDBProvided is returned in the event that an uninitialized db is
	// used to perform actions against.
	ErrInvalidDBProvided = errors.New("Invalid DB provided")

	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Entity not found")
)

// DB is a collection of support for different DB technologies. Currently
// only documentstorage has been implemented. We want to be able to access
// the raw database support for the given DB so an interface does not work.
// Each database is too different.
type DB struct {
	storage storage.Storage
	lock    sync.RWMutex
}

// StorageConfig is geared towards "bucket" style storage, where you have a
// specific root (the Bucket).
type StorageConfig struct {
	Bucket     string
	Root       string
	MaxRetries int
	RetryDelay int // Milliseconds between retries
}

// New returns a new DB value for use with document storage based on a
// registered master session.
func New(sc *StorageConfig) (*DB, error) {

	// S3 Storage
	var store storage.Storage
	if sc != nil {
		storeConfig := storage.NewConfig(sc.Bucket, sc.Root)
		if strings.ToLower(sc.Bucket) == "standalone" {
			store = storage.NewFilesystemStorage(storeConfig)
		} else {
			store = storage.NewS3Storage(storeConfig)
		}
	}

	db := DB{
		storage: store,
	}

	return &db, nil
}

// StatusCheck validates the DB status good.
func (db *DB) StatusCheck(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "platform.DB.StatusCheck")
	defer span.End()

	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage != nil {
		// Generate a random key that is almost certain not to exist.
		uid, _ := uuid.NewRandom()
		ts := time.Now().UnixNano()
		k := fmt.Sprintf("healthcheck/%v/%v", uid, ts)

		// We should receive a "not found" error for a non-existant key.
		if _, err := db.Fetch(ctx, k); err != ErrNotFound {
			return err
		}
	}

	return nil
}

// Close closes a DB value being used.
func (db *DB) Close() {
	db.lock.Lock()
	db.storage = nil
	db.lock.Unlock()
}

// -------------------------------------------------------------------------
// Storage

// Put something in storage
func (db *DB) Put(ctx context.Context, key string, body []byte) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage == nil {
		return errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	return db.storage.Write(ctx, key, body, nil)
}

// Fetch something from storage
func (db *DB) Fetch(ctx context.Context, key string) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage == nil {
		return nil, errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	b, err := db.storage.Read(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = ErrNotFound
		}

		return nil, err
	}

	return b, nil
}

// Remove something from storage
func (db *DB) Remove(ctx context.Context, key string) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage == nil {
		return errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	return db.storage.Remove(ctx, key)
}

// Search for things in storage
func (db *DB) Search(ctx context.Context, keyStart string) ([][]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage == nil {
		return nil, errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}
	query := map[string]string{
		"path": keyStart,
	}

	return db.storage.Search(ctx, query)
}

// List returns the keys under a given path.
func (db *DB) List(ctx context.Context, key string) ([]string, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.storage == nil {
		return nil, errors.Wrap(ErrInvalidDBProvided, "storage == nil")
	}

	return db.storage.List(ctx, key)
}

func (db *DB) Clear(ctx context.Context, keyStart string) error {
	query := map[string]string{
		"path": keyStart,
	}

	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.storage.Clear(ctx, query)
}
