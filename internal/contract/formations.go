package contract

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const cfStorageKey = "formations"

func SaveContractFormation(ctx context.Context, dbConn *db.DB, ra bitcoin.RawAddress,
	cf *actions.ContractFormation, isTest bool) error {

	key := buildCFStoragePath(ra) // First input is smart contract address

	// Check existing timestamp
	b, err := dbConn.Fetch(ctx, key)
	if err == nil {
		existingAction, err := protocol.Deserialize(b, isTest)
		if err != nil {
			return errors.Wrap(err, "deserialize existing")
		}

		existingCF, ok := existingAction.(*actions.ContractFormation)
		if !ok {
			return errors.New("Not Contract Formation")
		}

		if existingCF.Timestamp > cf.Timestamp {
			// Existing timestamp is after this timestamp so don't overwrite it.
			return nil
		}
	} else if errors.Cause(err) != db.ErrNotFound {
		return errors.Wrap(err, "Failed to fetch contract")
	}

	// Update any existing contracts that reference this contract.
	if err := updateExpandedOracles(ctx, ra, cf); err != nil {
		return errors.Wrap(err, "update expanded oracles")
	}

	b, err = protocol.Serialize(cf, isTest)
	if err != nil {
		return errors.Wrap(err, "serialize")
	}

	if err := dbConn.Put(ctx, key, b); err != nil {
		return errors.Wrap(err, "update storage")
	}

	return nil
}

func FetchContractFormation(ctx context.Context, dbConn *db.DB, ra bitcoin.RawAddress, isTest bool) (*actions.ContractFormation, error) {
	key := buildCFStoragePath(ra)
	b, err := dbConn.Fetch(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch contract")
	}

	action, err := protocol.Deserialize(b, isTest)
	if err != nil {
		return nil, errors.Wrap(err, "deserialize action")
	}

	cf, ok := action.(*actions.ContractFormation)
	if !ok {
		return nil, errors.New("Not Contract Formation")
	}

	return cf, nil
}

func GetIdentityOracleKey(cf *actions.ContractFormation) (bitcoin.PublicKey, error) {
	for _, service := range cf.Services {
		if service.Type == actions.OracleTypeIdentity {
			return bitcoin.PublicKeyFromBytes(service.PublicKey)
		}
	}

	return bitcoin.PublicKey{}, errors.New("Not Found")
}

// buildCFStoragePath returns the storage path prefix for a given identifier.
func buildCFStoragePath(ra bitcoin.RawAddress) string {
	return fmt.Sprintf("%s/%x", cfStorageKey, ra.Bytes())
}
