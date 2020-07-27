package contract

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
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
	c.FullOracles = make([]state.Oracle, 0, len(c.Oracles))
	for _, oracle := range c.Oracles {
		isIdentity := false
		for _, t := range oracle.OracleTypes {
			if t == actions.ServiceTypeIdentityOracle {
				isIdentity = true
				break
			}
		}

		if !isIdentity {
			c.FullOracles = append(c.FullOracles, state.Oracle{})
		}

		ra, err := bitcoin.DecodeRawAddress(oracle.EntityContract)
		if err != nil {
			return errors.Wrap(err, "entity address")
		}

		cf, err := FetchContractFormation(ctx, dbConn, ra, isTest)
		if err != nil {
			return errors.Wrap(err, "fetch entity")
		}

		found := false
		var url string
		var publicKey bitcoin.PublicKey
		for _, service := range cf.Services {
			if service.Type == actions.ServiceTypeIdentityOracle {
				found = true
				url = service.URL
				publicKey, err = bitcoin.PublicKeyFromBytes(service.PublicKey)
				if err != nil {
					return errors.Wrap(err, "identity key")
				}
				break
			}
		}

		if found {
			c.FullOracles = append(c.FullOracles, state.Oracle{
				Address:   ra,
				URL:       url,
				PublicKey: publicKey,
			})
		} else {
			c.FullOracles = append(c.FullOracles, state.Oracle{})
		}
	}
	return nil
}

// updateExpandedOracles updates expanded oracles that are in the cache.
func updateExpandedOracles(ctx context.Context, ra bitcoin.RawAddress, cf *actions.ContractFormation) error {
	isIdentity := false
	serviceIndex := 0
	for i, service := range cf.Services {
		if service.Type == actions.ServiceTypeIdentityOracle {
			isIdentity = true
			serviceIndex = i
			break
		}
	}

	if !isIdentity {
		return nil // Only identity oracles are used in expanded form.
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()
	for _, c := range cache {
		for i, oracle := range c.FullOracles {
			if !ra.Equal(oracle.Address) {
				continue
			}

			publicKey, err := bitcoin.PublicKeyFromBytes(cf.Services[serviceIndex].PublicKey)
			if err != nil {
				return errors.Wrap(err, "parse public key")
			}
			c.FullOracles[i].PublicKey = publicKey
			c.FullOracles[i].URL = cf.Services[serviceIndex].URL
		}
	}

	return nil
}
