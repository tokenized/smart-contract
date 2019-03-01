package rpcnode

import (
	"context"
	"os"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Prior to running test, set the following environment variables.
//   RPC_HOST
//   RPC_USERNAME
//   RPC_PASSWORD
//   TX_ID
func TestNode(test *testing.T) {
	ctx := context.Background()

	host := os.Getenv("RPC_HOST")
	user := os.Getenv("RPC_USERNAME")
	password := os.Getenv("RPC_PASSWORD")
	config := NewConfig(host, user, password)
	test.Logf("Connect to %s as %s password : %s", host, user, password)

	node, err := NewNode(config)
	if err != nil {
		test.Errorf("Failed to create node : %s", err.Error())
	}

	txid, err := chainhash.NewHashFromStr(os.Getenv("TX_ID"))
	test.Logf("Get Tx : %s", txid.String())

	if tx, err := node.GetTX(ctx, txid); err != nil {
		test.Errorf("Failed to get tx : %s", err.Error())
	} else {
		test.Logf("Tx : %s\n%+v", tx.TxHash().String(), tx)
	}
}
