package identity

import (
	"context"

	"github.com/google/uuid"
	"github.com/tokenized/pkg/bitcoin"

	"github.com/pkg/errors"
)

// GetOracle fetches oracle data from the URL.
func GetOracle(ctx context.Context, baseURL string) (*Oracle, error) {
	result := &Oracle{
		URL: baseURL,
	}

	var response struct {
		Data struct {
			ContractAddress bitcoin.RawAddress `json:"contract_address"`
			PublicKey       bitcoin.PublicKey  `json:"public_key"`
		}
	}

	if err := get(result.URL+"/oracle/id", &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	result.ContractAddress = response.Data.ContractAddress
	result.PublicKey = response.Data.PublicKey

	return result, nil
}

// NewOracle creates an oracle from specified data.
func NewOracle(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (*Oracle, error) {
	return &Oracle{
		ContractAddress: contractAddress,
		URL:             url,
		PublicKey:       publicKey,
	}, nil
}

// GetContractAddress returns the oracle's contract address.
func (o *Oracle) GetContractAddress() bitcoin.RawAddress {
	return o.ContractAddress
}

// GetURL returns the oracle's service URL.
func (o *Oracle) GetURL() string {
	return o.URL
}

// GetPublicKey returns the oracle's public key.
func (o *Oracle) GetPublicKey() bitcoin.PublicKey {
	return o.PublicKey
}

// SetClientID sets the client's ID and authorization key.
func (o *Oracle) SetClientID(id uuid.UUID, key bitcoin.Key) {
	o.ClientID = id
	o.ClientAuthKey = key
}

// SetClientKey sets the client's authorization key.
func (o *Oracle) SetClientKey(key bitcoin.Key) {
	o.ClientAuthKey = key
}
