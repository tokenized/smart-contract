package identity

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/google/uuid"
)

type Oracle struct {
	// Oracle information
	ContractAddress bitcoin.RawAddress // Address of oracle's contract entity.
	URL             string
	PublicKey       bitcoin.PublicKey

	// Client information
	ClientID      uuid.UUID   // User ID of client
	ClientAuthKey bitcoin.Key // Key used to authorize/encrypt with oracle

	// TODO Implement retry functionality --ce
	// MaxRetries int
	// RetryDelay int
}

type BlockHashes interface {
	Hash(ctx context.Context, height int) (*bitcoin.Hash32, error)
}
