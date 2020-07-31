package identity

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/google/uuid"
)

type Factory interface {
	NewClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (Client, error)
}

type Client interface {
	// RegisterUser registers a user with the identity oracle.
	RegisterUser(ctx context.Context, entity actions.EntityField, xpubs []bitcoin.ExtendedKeys) (uuid.UUID, error)

	// RegisterXPub registers an xpub under a user with an identity oracle.
	RegisterXPub(ctx context.Context, path string, xpubs bitcoin.ExtendedKeys, requiredSigners int) error

	// ApproveReceive requests an approval signature for a receiver from an identity oracle.
	ApproveReceive(ctx context.Context, contract, asset string, oracleIndex int, quantity uint64,
		xpubs bitcoin.ExtendedKeys, index uint32, requiredSigners int) (*actions.AssetReceiverField, bitcoin.Hash32, error)

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

type HTTPClient struct {
	// Oracle information
	ContractAddress bitcoin.RawAddress // Address of oracle's contract entity.
	URL             string
	PublicKey       bitcoin.PublicKey

	// Client information
	ClientID  uuid.UUID   // User ID of client
	ClientKey bitcoin.Key // Key used to authorize/encrypt with oracle

	// TODO Implement retry functionality --ce
	// MaxRetries int
	// RetryDelay int
}

type BlockHashes interface {
	Hash(ctx context.Context, height int) (*bitcoin.Hash32, error)
}
