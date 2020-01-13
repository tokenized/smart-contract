package rpcnode

import (
	"fmt"
)

type Config struct {
	Host     string
	Username string
	Password string

	// Retry attempts when calls fail.
	MaxRetries int
	RetryDelay int
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	return fmt.Sprintf("{Host:%v Username:%v Password:%v MaxRetries:%d RetryDelay:%d ms}", c.Host,
		c.Username, "****", c.MaxRetries, c.RetryDelay)
}
