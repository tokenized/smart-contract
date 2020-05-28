package authority

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

type Oracle struct {
	BaseURL string

	// Oracle information
	OracleKey    bitcoin.PublicKey
	OracleEntity actions.EntityField

	// TODO Implement retry functionality --ce
	// MaxRetries int
	// RetryDelay int
}
