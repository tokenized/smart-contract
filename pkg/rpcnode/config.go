package rpcnode

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
)

type Config struct {
	Host        string
	Username    string
	Password    string
	ChainParams *chaincfg.Params
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	return fmt.Sprintf("{Host:%v Username:%v Password:%v}", c.Host,
		c.Username,
		"****")
}
