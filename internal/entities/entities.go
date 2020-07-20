package entities

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

const storageKey = "entities"

func SaveContractFormation(ctx context.Context, dbConn *db.DB, itx *inspector.Transaction, isTest bool) error {

	if itx.MsgProto == nil {
		return errors.New("Empty action")
	}

	cf, ok := itx.MsgProto.(*actions.ContractFormation)
	if !ok {
		return errors.New("Not Contract Formation")
	}

	key := buildStoragePath(itx.Inputs[0].Address) // First input is smart contract address

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

	// TODO Keep list of all entity addresses currently used by active contracts under this smart
	// contract agent. Then update the contract data when a new contract formation is seen. --ce

	if err := dbConn.Put(ctx, key, itx.MsgTx.TxOut[itx.MsgProtoIndex].PkScript); err != nil {
		return err
	}

	return nil
}

func FetchEntity(ctx context.Context, dbConn *db.DB, ra bitcoin.RawAddress, isTest bool) (*actions.ContractFormation, error) {
	key := buildStoragePath(ra)
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

// Returns the storage path prefix for a given identifier.
func buildStoragePath(ra bitcoin.RawAddress) string {
	return fmt.Sprintf("%s/%x", storageKey, ra.Bytes())
}
