package identity

import (
	"context"

	"github.com/google/uuid"
	"github.com/tokenized/pkg/bitcoin"

	"github.com/pkg/errors"
)

// GetOracle fetches an HTTP oracle client's data from the URL.
func GetHTTPOracle(ctx context.Context, baseURL string) (*HTTPClient, error) {
	result := &HTTPClient{
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

// NewHTTPClient creates an HTTP oracle client from specified data.
func NewHTTPClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (*HTTPClient, error) {
	return &HTTPClient{
		ContractAddress: contractAddress,
		URL:             url,
		PublicKey:       publicKey,
	}, nil
}

// GetContractAddress returns the oracle's contract address.
func (o *HTTPClient) GetContractAddress() bitcoin.RawAddress {
	return o.ContractAddress
}

// GetURL returns the oracle's service URL.
func (o *HTTPClient) GetURL() string {
	return o.URL
}

// GetPublicKey returns the oracle's public key.
func (o *HTTPClient) GetPublicKey() bitcoin.PublicKey {
	return o.PublicKey
}

// SetClientID sets the client's ID and authorization key.
func (o *HTTPClient) SetClientID(id uuid.UUID, key bitcoin.Key) {
	o.ClientID = id
	o.ClientKey = key
}

// SetClientKey sets the client's authorization key.
func (o *HTTPClient) SetClientKey(key bitcoin.Key) {
	o.ClientKey = key
}
