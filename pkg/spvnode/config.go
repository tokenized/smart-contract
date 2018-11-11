package spvnode

import (
	"fmt"
	"strings"
)

// Config holds all configuration for the running service.
type Config struct {
	NodeAddress string
	UserAgent   string
}

// NewConfig returns a new Config populated from environment variables.
func NewConfig(host, useragent string) Config {
	c := Config{
		NodeAddress: host,
		UserAgent:   useragent,
	}

	return c
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	pairs := map[string]string{
		"NodeAddress": c.NodeAddress,
		"UserAgent":   c.UserAgent,
	}

	parts := []string{}

	for k, v := range pairs {
		parts = append(parts, fmt.Sprintf("%v:%v", k, v))
	}

	return fmt.Sprintf("{%v}", strings.Join(parts, " "))
}
