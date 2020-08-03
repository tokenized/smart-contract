package operator

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type MockFactory struct {
	clients []*MockClient
}

func NewMockFactory() *MockFactory {
	return &MockFactory{}
}

func (f *MockFactory) NewClient(contractAddress bitcoin.RawAddress, url string,
	publicKey bitcoin.PublicKey) (Client, error) {
	// Find setup mock oracle
	for _, client := range f.clients {
		if client.ContractAddress.Equal(contractAddress) {
			return client, nil
		}
	}

	return nil, errors.New("Client contract address not found")
}

// SetupOracle prepares a mock client in the factory. This must be called before calling NewClient.
func (f *MockFactory) SetupOracle(contractAddress bitcoin.RawAddress, url string, key bitcoin.Key,
	contractFee uint64) {
	// Find setup mock oracle
	f.clients = append(f.clients, &MockClient{
		ContractAddress: contractAddress,
		URL:             url,
		Key:             key,
		contractFee:     contractFee,
	})
}

type MockClient struct {
	// Oracle information
	ContractAddress bitcoin.RawAddress // Address of oracle's contract entity.
	URL             string
	Key             bitcoin.Key

	// Client information
	ClientID  uuid.UUID   // User ID of client
	ClientKey bitcoin.Key // Key used to authorize/encrypt with oracle

	contractFee uint64
	keys        []*bitcoin.Key
}

func (c *MockClient) FetchContractAddress(ctx context.Context) (bitcoin.RawAddress, uint64, error) {
	k, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, errors.Wrap(err, "generate key")
	}

	ra, err := k.RawAddress()
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, errors.Wrap(err, "create address")
	}

	return ra, c.contractFee, nil
}

// GetContractAddress returns the oracle's contract address.
func (c *MockClient) GetContractAddress() bitcoin.RawAddress {
	return c.ContractAddress
}

// GetURL returns the oracle's URL.
func (c *MockClient) GetURL() string {
	return c.URL
}

// GetPublicKey returns the oracle's public key.
func (c *MockClient) GetPublicKey() bitcoin.PublicKey {
	return c.Key.PublicKey()
}

// SetClientID sets the client's ID and authorization key.
func (c *MockClient) SetClientID(id uuid.UUID, key bitcoin.Key) {
	c.ClientID = id
	c.ClientKey = key
}

// SetClientKey sets the client's authorization key.
func (c *MockClient) SetClientKey(key bitcoin.Key) {
	c.ClientKey = key
}
