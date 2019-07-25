package tests

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var a protomux.Handler
var test *tests.Test

// Information about the handlers we have created for testing.
var responses []*wire.MsgTx
var responseLock sync.Mutex

var userKey *wallet.Key
var user2Key *wallet.Key
var issuerKey *wallet.Key
var oracleKey *wallet.Key
var authorityKey *wallet.Key

var testTokenQty uint64
var testToken2Qty uint64
var testAssetType string
var testAsset2Type string
var testAssetCode protocol.AssetCode
var testAsset2Code protocol.AssetCode
var testVoteTxId protocol.TxId
var testVoteResultTxId protocol.TxId

// TestMain is the entry point for testing.
func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	test = tests.New(false)
	if test == nil {
		return 1
	}
	defer test.TearDown()

	// =========================================================================
	// Locals

	testTokenQty = 1000

	// =========================================================================
	// API

	tracer := filters.NewTracer()

	var err error
	a, err = handlers.API(
		test.Context,
		test.Wallet,
		&test.NodeConfig,
		test.MasterDB,
		tracer,
		test.Scheduler,
		test.Headers,
		test.UTXOs,
	)

	if err != nil {
		panic(err)
	}

	a.SetResponder(respondTx)
	a.SetReprocessor(reprocessTx)

	// =========================================================================
	// Keys

	userKey, err = tests.GenerateKey(wire.BitcoinNet(test.NodeConfig.ChainParams.Net))
	if err != nil {
		panic(err)
	}

	user2Key, err = tests.GenerateKey(wire.BitcoinNet(test.NodeConfig.ChainParams.Net))
	if err != nil {
		panic(err)
	}

	issuerKey, err = tests.GenerateKey(wire.BitcoinNet(test.NodeConfig.ChainParams.Net))
	if err != nil {
		panic(err)
	}

	oracleKey, err = tests.GenerateKey(wire.BitcoinNet(test.NodeConfig.ChainParams.Net))
	if err != nil {
		panic(err)
	}

	authorityKey, err = tests.GenerateKey(wire.BitcoinNet(test.NodeConfig.ChainParams.Net))
	if err != nil {
		panic(err)
	}

	return m.Run()
}

func respondTx(ctx context.Context, tx *wire.MsgTx) error {
	responseLock.Lock()
	responses = append(responses, tx)
	responseLock.Unlock()
	return nil
}

func reprocessTx(ctx context.Context, itx *inspector.Transaction) error {
	return a.Trigger(ctx, "REPROCESS", itx)
}

func getResponse() *wire.MsgTx {
	responseLock.Lock()
	defer responseLock.Unlock()

	if len(responses) == 0 {
		return nil
	}

	result := responses[0]
	responses = responses[1:]
	return result
}

func responseType(tx *wire.MsgTx) string {
	for _, output := range tx.TxOut {
		msg, err := protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			return msg.Type()
		}
	}
	return ""
}

// checkResponse fails the test if the respons is not of the specified type
func checkResponse(t testing.TB, responseCode string) *wire.MsgTx {
	ctx := test.Context

	responseLock.Lock()

	if len(responses) != 1 {
		responseLock.Unlock()
		t.Fatalf("\t%s\t%s Response not created", tests.Failed, responseCode)
	}

	response := responses[0]
	responses = nil
	responseLock.Unlock()

	var responseMsg protocol.OpReturnMessage
	var err error
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\t%s Response doesn't contain tokenized op return", tests.Failed, responseCode)
	}
	if responseMsg.Type() != responseCode {
		t.Fatalf("\t%s\tResponse is the wrong type : %s != %s", tests.Failed, responseMsg.Type(), responseCode)
	}

	// Submit response
	responseItx, err := inspector.NewTransactionFromWire(ctx, response, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create response itx : %v", tests.Failed, err)
	}

	err = responseItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote response itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	responseLock.Lock()
	if len(responses) != 0 {
		responseLock.Unlock()
		t.Fatalf("\t%s\tResponse created a response", tests.Failed)
	}
	responseLock.Unlock()

	t.Logf("\t%s\tResponse processed : %s", tests.Success, responseCode)
	return response
}

// checkResponses fails the test if all responses are not of the specified type
func checkResponses(t testing.TB, responseCode string) {
	var responseMsg protocol.OpReturnMessage
	var err error

	responseLock.Lock()
	responsesToProcess := responses
	responses = nil
	responseLock.Unlock()

	for i, response := range responsesToProcess {
		for _, output := range response.TxOut {
			responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
			if err == nil {
				break
			}
		}
		if responseMsg == nil {
			t.Fatalf("\t%s\t%s Response %d doesn't contain tokenized op return", tests.Failed, responseCode, i)
		}
		if responseMsg.Type() != responseCode {
			t.Fatalf("\t%s\tResponse %d is the wrong type : %s != %s", tests.Failed, i, responseMsg.Type(), responseCode)
		}
	}
}

func resetTest(ctx context.Context) error {
	responseLock.Lock()
	responses = nil
	responseLock.Unlock()
	asset.Reset(ctx)
	contract.Reset(ctx)
	holdings.Reset(ctx)
	return test.Reset(ctx)
}
