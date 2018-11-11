package spvnode

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/tokenized/smart-contract/pkg/storage"
)

type State struct {
	LastSeen Block `json:"last_seen"`
}

var ErrStateNotFound = errors.New("State not found")

type StateRepository struct {
	Storage storage.Storage
}

func NewStateRepository(store storage.Storage) StateRepository {
	return StateRepository{
		Storage: store,
	}
}

func (r StateRepository) Write(ctx context.Context, c State) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := r.buildPath()

	return r.Storage.Write(ctx, key, b, nil)
}

func (r StateRepository) Read(ctx context.Context, id string) (*State, error) {
	key := r.buildPath()

	b, err := r.Storage.Read(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = ErrStateNotFound
		}

		return nil, err
	}

	// we have found a matching key
	c := State{}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func (r StateRepository) buildPath() string {
	return "state.json"
}
