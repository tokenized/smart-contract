package rpcnode

import (
	"context"
	"os"
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

// Prior to running test, set the following environment variables.
//   RPC_HOST
//   RPC_USERNAME
//   RPC_PASSWORD
//   TX_ID
func ManualTestNode(test *testing.T) {
	ctx := context.Background()

	config := &Config{
		Host:     os.Getenv("RPC_HOST"),
		Username: os.Getenv("RPC_USERNAME"),
		Password: os.Getenv("RPC_PASSWORD"),
	}
	test.Logf("Connect to %s as %s password : %s", config.Host, config.Username, config.Password)

	node, err := NewNode(config)
	if err != nil {
		test.Errorf("Failed to create node : %s", err.Error())
	}

	txid, err := bitcoin.NewHash32FromStr(os.Getenv("TX_ID"))
	test.Logf("Get Tx : %s", txid.String())

	if tx, err := node.GetTX(ctx, txid); err != nil {
		test.Errorf("Failed to get tx : %s", err.Error())
	} else {
		test.Logf("Tx : %s\n%+v", tx.TxHash().String(), tx)
	}
}
