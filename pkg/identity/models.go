package identity

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
)

type Oracle struct {
	BaseURL string

	// Oracle information
	OracleKey    bitcoin.PublicKey
	OracleEntity actions.EntityField

	// Client information
	ClientAuthKey bitcoin.Key // Key used to authorize/encrypt with oracle
	ClientID      string      // User ID of client

	// TODO Implement retry functionality --ce
	// MaxRetries int
	// RetryDelay int
}

type BlockHashes interface {
	Hash(ctx context.Context, height int) (*bitcoin.Hash32, error)
}
