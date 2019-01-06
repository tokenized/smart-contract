package state

/**
 * State Kit
 *
 * What is my purpose?
 * - You store the state for contracts
 * - You harden state based on blockchain confirmations
 */

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/storage"
)

const (
	ContractPrefix = "contracts"
	StateSoft      = "soft"
	StateHard      = "hard"
)

var ErrContractNotFound = errors.New("Contract not found")

type StateService struct {
	Storage storage.ReadWriter
}

func NewStateService(store storage.ReadWriter) StateService {
	return StateService{
		Storage: store,
	}
}

func (r StateService) WriteHard(ctx context.Context, c contract.Contract) error {
	return r.write(ctx, c, StateHard)
}

func (r StateService) ReadHard(ctx context.Context, addr string) (*contract.Contract, error) {
	return r.read(ctx, addr, StateHard)
}

func (r StateService) Write(ctx context.Context, c contract.Contract) error {
	return r.write(ctx, c, StateSoft)
}

func (r StateService) Read(ctx context.Context, addr string) (*contract.Contract, error) {
	return r.read(ctx, addr, StateSoft)
}

func (r StateService) write(ctx context.Context,
	c contract.Contract, stateStore string) error {
	defer logger.Elapsed(ctx, time.Now(), "StateService.Write")

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := r.buildPath(c.ID) + "-" + stateStore

	return r.Storage.Write(ctx, key, data, nil)
}

func (r StateService) read(ctx context.Context,
	addr string, stateStore string) (*contract.Contract, error) {

	defer logger.Elapsed(ctx, time.Now(), "StateService.Read")

	key := r.buildPath(addr) + "-" + stateStore

	b, err := r.Storage.Read(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = ErrContractNotFound
		}

		return nil, err
	}

	// we have found a matching key
	c := contract.Contract{}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func (r StateService) buildPath(id string) string {
	return fmt.Sprintf("%v/%v", ContractPrefix, id)
}
