package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/storage"

	"github.com/pkg/errors"
)

const (
	peersPath    = "spynode/peers"
	peersVersion = 2
)

// Peer address database. Used to find Tx Peers.
type Peer struct {
	Address  string
	Score    int32
	LastTime uint32
}

// TxRepository is used for managing Block data
type PeerRepository struct {
	store     storage.Storage
	lookup    map[string]*Peer
	list      []*Peer
	LastSaved time.Time
	mutex     sync.Mutex
}

// Creates a new PeerRepository
func NewPeerRepository(store storage.Storage) *PeerRepository {
	result := PeerRepository{
		store:     store,
		lookup:    make(map[string]*Peer),
		list:      make([]*Peer, 0),
		LastSaved: time.Now(),
	}
	return &result
}

// Loads peers from storage
func (repo *PeerRepository) Load(ctx context.Context) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Clear
	repo.list = make([]*Peer, 0)
	repo.lookup = make(map[string]*Peer)

	// Get current data
	data, err := repo.store.Read(ctx, peersPath)
	if err == storage.ErrNotFound {
		return nil // Leave empty
	}
	if err != nil {
		return err
	}

	// Parse peers
	buffer := bytes.NewBuffer(data)
	var version int32
	if err := binary.Read(buffer, binary.LittleEndian, &version); err != nil {
		return errors.Wrap(err, "Failed to read peers version")
	}

	var count int32
	if err := binary.Read(buffer, binary.LittleEndian, &count); err != nil {
		return errors.Wrap(err, "Failed to read peers count")
	}

	// Reset
	repo.list = make([]*Peer, 0, count)

	// Parse peers
	for {
		peer, err := readPeer(buffer, version)
		if err != nil {
			break
		}

		// Add peer
		repo.list = append(repo.list, &peer)
		repo.lookup[peer.Address] = &peer
	}

	logger.Verbose(ctx, "Loaded %d peers", len(repo.list))

	repo.LastSaved = time.Now()
	return nil
}

func (repo *PeerRepository) Count() int {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return len(repo.list)
}

// Adds a peer if it isn't already in the database
// Returns true if it was added
func (repo *PeerRepository) Add(ctx context.Context, address string) (bool, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	_, exists := repo.lookup[address]
	if exists {
		return false, nil
	}

	// Add peer
	peer := Peer{Address: address, Score: 0}
	repo.list = append(repo.list, &peer)
	repo.lookup[peer.Address] = &peer
	return true, nil
}

// Get returns all peers at or above the specified score
func (repo *PeerRepository) Get(ctx context.Context, minScore int32) ([]*Peer, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	result := make([]*Peer, 0, 500)
	for _, peer := range repo.list {
		if peer.Score >= minScore {
			result = append(result, peer)
		}
	}

	return result, nil
}

// GetUnchecked returns all peers with a zero score
func (repo *PeerRepository) GetUnchecked(ctx context.Context) ([]*Peer, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	result := make([]*Peer, 0, 500)
	now := time.Now()
	cutoff := uint32(now.Unix()) - 86400 // 24 hours
	for _, peer := range repo.list {
		if peer.Score == 0 && peer.LastTime < cutoff {
			result = append(result, peer)
		}
	}

	return result, nil
}

// Modifies the score of a peer
// Returns true if found and updated
func (repo *PeerRepository) UpdateScore(ctx context.Context, address string, delta int32) bool {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	peer, exists := repo.lookup[address]
	if exists {
		now := time.Now()
		peer.LastTime = uint32(now.Unix())
		peer.Score += delta
		return true
	}

	return false
}

// Modifies the score of a peer
// Returns true if found and updated
func (repo *PeerRepository) UpdateTime(ctx context.Context, address string) bool {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	peer, exists := repo.lookup[address]
	if exists {
		now := time.Now()
		peer.LastTime = uint32(now.Unix())
		return true
	}

	return false
}

// Saves the peers to storage
func (repo *PeerRepository) Save(ctx context.Context) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var buffer bytes.Buffer

	// Write version
	if err := binary.Write(&buffer, binary.LittleEndian, int32(peersVersion)); err != nil {
		return err
	}

	logger.Verbose(ctx, "Saving %d peers", len(repo.list))

	// Write count
	if err := binary.Write(&buffer, binary.LittleEndian, int32(len(repo.list))); err != nil {
		return err
	}

	// Write peers
	for _, peer := range repo.list {
		if err := peer.write(&buffer); err != nil {
			return err
		}
	}

	if err := repo.store.Write(ctx, peersPath, buffer.Bytes(), nil); err != nil {
		return err
	}

	repo.LastSaved = time.Now()
	return nil
}

// Clears all peers from the database
func (repo *PeerRepository) Clear(ctx context.Context) error {
	repo.list = make([]*Peer, 0)
	repo.lookup = make(map[string]*Peer)
	repo.LastSaved = time.Now()
	return repo.store.Remove(ctx, peersPath)
}

func readPeer(input io.Reader, version int32) (Peer, error) {
	result := Peer{}

	// Read address
	var addressSize int32
	if err := binary.Read(input, binary.LittleEndian, &addressSize); err != nil {
		return result, err
	}

	addressData := make([]byte, addressSize)
	_, err := input.Read(addressData) // Read until string terminator
	if err != nil {
		return result, err
	}
	result.Address = string(addressData)

	// Read score
	if err := binary.Read(input, binary.LittleEndian, &result.Score); err != nil {
		return result, err
	}

	if version > 1 {
		// Read score
		if err := binary.Read(input, binary.LittleEndian, &result.LastTime); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (peer *Peer) write(output io.Writer) error {
	// Write address
	err := binary.Write(output, binary.LittleEndian, int32(len(peer.Address)))
	if err != nil {
		return err
	}
	_, err = output.Write([]byte(peer.Address))
	if err != nil {
		return err
	}

	// Write score
	err = binary.Write(output, binary.LittleEndian, peer.Score)
	if err != nil {
		return err
	}

	// Write time
	err = binary.Write(output, binary.LittleEndian, peer.LastTime)
	if err != nil {
		return err
	}

	return nil
}
