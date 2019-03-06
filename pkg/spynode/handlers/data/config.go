package data

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Config holds all configuration for the running service.
type Config struct {
	NodeAddress    string         // IP address of trusted external full node
	UserAgent      string         // User agent to send to external node
	StartHash      chainhash.Hash // Hash of first block to start processing on initial run
	UntrustedCount int            // The number of untrusted nodes to run for double spend monitoring
	SafeTxDelay    int            // Number of milliseconds without conflict before a tx is "safe"
}

// NewConfig returns a new Config populated from environment variables.
func NewConfig(host, useragent, starthash string, untrustedNodes, safeDelay int) (Config, error) {
	result := Config{
		NodeAddress:    host,
		UserAgent:      useragent,
		UntrustedCount: untrustedNodes,
		SafeTxDelay:    safeDelay,
	}

	hash, err := chainhash.NewHashFromStr(starthash)
	if err != nil {
		return result, err
	}
	result.StartHash = *hash
	return result, nil
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	pairs := map[string]string{
		"NodeAddress": c.NodeAddress,
		"UserAgent":   c.UserAgent,
		"StartHash":   c.StartHash.String(),
		"SafeTxDelay": fmt.Sprintf("%d ms", c.SafeTxDelay),
	}

	parts := []string{}

	for k, v := range pairs {
		parts = append(parts, fmt.Sprintf("%v:%v", k, v))
	}

	return fmt.Sprintf("{%v}", strings.Join(parts, " "))
}
