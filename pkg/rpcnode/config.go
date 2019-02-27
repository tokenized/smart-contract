package rpcnode

import (
	"fmt"
)

type Config struct {
	Host     string
	Username string
	Password string
}

func NewConfig(host, username, password string) Config {
	return Config{
		Host:     host,
		Username: username,
		Password: password,
	}
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	return fmt.Sprintf("{Host:%v Username:%v Password:%v}", c.Host,
		c.Username,
		"****")
}
