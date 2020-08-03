package operator

import (
	"context"

	"github.com/google/uuid"
	"github.com/tokenized/pkg/bitcoin"
)

type Factory interface {
	NewClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (Client, error)
}

type Client interface {
	// FetchContractAddress requests a hosted smart contract agent address from the operator.
	FetchContractAddress(context.Context) (bitcoin.RawAddress, uint64, error)

	// GetContractAddress returns the oracle's contract address.
	GetContractAddress() bitcoin.RawAddress

	// GetURL returns the oracle's URL.
	GetURL() string

	// GetPublicKey returns the oracle's public key.
	GetPublicKey() bitcoin.PublicKey

	// SetClientID sets the client's ID and authorization key.
	SetClientID(id uuid.UUID, key bitcoin.Key)

	// SetClientKey sets the client's authorization key.
	SetClientKey(key bitcoin.Key)
}
