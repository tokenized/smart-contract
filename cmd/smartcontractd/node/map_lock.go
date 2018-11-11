package node

import (
	"sync"
)

// mapLock is used to manage multiple locks for various keys.
type mapLock struct {
	mu    *sync.Mutex
	locks map[string]*sync.Mutex
}

// newMapLock returns a new mapLock.
func newMapLock() mapLock {
	return mapLock{
		mu:    &sync.Mutex{},
		locks: map[string]*sync.Mutex{},
	}
}

// get returns a Mutex for a given key. It is up to the caller to Lock and
// Unlock the Mutex.
func (m mapLock) get(key string) *sync.Mutex {
	// top level mutex ensuring only one process accesses the map of locks.
	m.mu.Lock()
	defer m.mu.Unlock()

	mu, ok := m.locks[key]
	if ok {
		return mu
	}

	mu = &sync.Mutex{}
	m.locks[key] = mu

	return mu
}
