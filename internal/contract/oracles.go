package contract

import (
	"context"

	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

// ExpandOracles pulls data for oracles used in the contract and makes it ready for request
// processing.
func ExpandOracles(ctx context.Context, dbConn *db.DB, c *state.Contract, isTest bool) error {
	logger.Info(ctx, "Expanding %d oracle public keys", len(c.Oracles))

	// Expand oracle public keys
	c.FullOracles = make([]*state.Oracle, len(c.Oracles))
	for i, oracle := range c.Oracles {
		if len(oracle.OracleTypes) == 0 || len(oracle.EntityContract) == 0 {
			continue
		}

		ra, err := bitcoin.DecodeRawAddress(oracle.EntityContract)
		if err != nil {
			return errors.Wrap(err, "entity address")
		}

		cf, err := FetchContractFormation(ctx, dbConn, ra, isTest)
		if err != nil {
			return errors.Wrap(err, "fetch entity")
		}

		fullOracle, err := buildFullOracle(ctx, ra, cf)
		if err != nil {
			return errors.Wrap(err, "build full oracle")
		}

		c.FullOracles[i] = fullOracle
	}

	return nil
}

func buildFullOracle(ctx context.Context, ra bitcoin.RawAddress,
	cf *actions.ContractFormation) (*state.Oracle, error) {

	fullOracle := &state.Oracle{
		Address: ra,
	}

	for _, service := range cf.Services {
		publicKey, err := bitcoin.PublicKeyFromBytes(service.PublicKey)
		if err != nil {
			return nil, errors.Wrap(err, "public key")
		}

		fullOracle.Services = append(fullOracle.Services, &state.Service{
			ServiceType: service.Type,
			URL:         service.URL,
			PublicKey:   publicKey,
		})
	}

	return fullOracle, nil
}

// updateExpandedOracles updates expanded oracles that are in the cache.
func updateExpandedOracles(ctx context.Context, ra bitcoin.RawAddress,
	cf *actions.ContractFormation) error {

	fullOracle, err := buildFullOracle(ctx, ra, cf)
	if err != nil {
		return errors.Wrap(err, "build full oracle")
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()
	for _, c := range cache {
		for i, oracle := range c.FullOracles {
			if ra.Equal(oracle.Address) {
				c.FullOracles[i] = fullOracle
			}
		}
	}

	return nil
}

func ContainsUint32(values []uint32, value uint32) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}
