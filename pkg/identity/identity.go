package identity

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

// GetOracle fetches oracle data from the URL.
func GetOracle(ctx context.Context, baseURL string, clientAuthKey bitcoin.Key) (*Oracle, error) {
	result := &Oracle{
		BaseURL:       baseURL,
		ClientAuthKey: clientAuthKey,
	}

	var response struct {
		Data struct {
			Entity    actions.EntityField `json:"entity"`
			URL       string              `json:"url"`
			PublicKey bitcoin.PublicKey   `json:"public_key"`
		}
	}

	if err := get(result.BaseURL+"/oracle/id", &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	result.OracleEntity = response.Data.Entity
	result.OracleKey = response.Data.PublicKey

	return result, nil
}

// NewOracle creates an oracle from existing data.
func NewOracle(baseURL string, oracleKey bitcoin.PublicKey, oracleEntity actions.EntityField,
	clientID string, clientAuthKey bitcoin.Key) (*Oracle, error) {
	return &Oracle{
		BaseURL:       baseURL,
		OracleKey:     oracleKey,
		OracleEntity:  oracleEntity,
		ClientID:      clientID,
		ClientAuthKey: clientAuthKey,
	}, nil
}

// GetOracleEntity returns the oracle's entity.
func (o *Oracle) GetOracleEntity() actions.EntityField {
	return o.OracleEntity
}

// GetOracleKey returns the oracle's public key.
func (o *Oracle) GetOracleKey() bitcoin.PublicKey {
	return o.OracleKey
}

// SetClientID sets the client's ID.
func (o *Oracle) SetClientID(id string) {
	o.ClientID = id
}
