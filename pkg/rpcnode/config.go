package rpcnode

import (
	"fmt"
)

type Config struct {
	Host     string
	Username string
	Password string
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	return fmt.Sprintf("{Host:%v Username:%v Password:%v}", c.Host,
		c.Username,
		"****")
}
