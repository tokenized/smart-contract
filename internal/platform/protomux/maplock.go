package protomux

import (
	"sync"
)

// To ensure multiple messages do not modify the same Contract in
// parallel, use a mutex to prevent parallel access on a contract
// address.
// mtx := h.mapLock.get(h.Wallet.PublicAddress)
// mtx.Lock()
// defer mtx.Unlock()

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
