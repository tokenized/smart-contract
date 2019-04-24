package tests

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var a protomux.Handler
var test *tests.Test

// Information about the handlers we have created for testing.
var responses []*wire.MsgTx
var headers mockHeaders
var r *rand.Rand

var userKey *wallet.RootKey
var user2Key *wallet.RootKey
var issuerKey *wallet.RootKey
var oracleKey *wallet.RootKey
var authorityKey *wallet.RootKey

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
	test = tests.New(true)
	defer test.TearDown()

	// =========================================================================
	// Locals

	testTokenQty = 1000

	// =========================================================================
	// API

	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	tracer := listeners.NewTracer()

	var err error
	a, err = handlers.API(
		test.Context,
		test.Wallet,
		&test.NodeConfig,
		test.MasterDB,
		tracer,
		test.Scheduler,
		&headers,
		test.UTXOs,
	)

	if err != nil {
		panic(err)
	}

	a.SetResponder(respondTx)

	// =========================================================================
	// Keys

	userKey, err = tests.GenerateKey(test.NodeConfig.ChainParams)
	if err != nil {
		panic(err)
	}

	user2Key, err = tests.GenerateKey(test.NodeConfig.ChainParams)
	if err != nil {
		panic(err)
	}

	issuerKey, err = tests.GenerateKey(test.NodeConfig.ChainParams)
	if err != nil {
		panic(err)
	}

	oracleKey, err = tests.GenerateKey(test.NodeConfig.ChainParams)
	if err != nil {
		panic(err)
	}

	authorityKey, err = tests.GenerateKey(test.NodeConfig.ChainParams)
	if err != nil {
		panic(err)
	}

	return m.Run()
}

func respondTx(ctx context.Context, tx *wire.MsgTx) error {
	responses = append(responses, tx)
	return nil
}

func checkResponse(t *testing.T, responseCode string) {
	ctx := test.Context

	if len(responses) != 1 {
		t.Fatalf("\t%s\t%s Response not created", tests.Failed, responseCode)
	}

	response := responses[0]
	responses = nil
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

	test.RPCNode.AddTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	if len(responses) != 0 {
		t.Fatalf("\t%s\tResponse created a response", tests.Failed)
	}

	t.Logf("\t%s\tResponse processed : %s", tests.Success, responseCode)
}

func resetTest() {
	test.ResetDB()
	responses = nil
	headers.height = 0
	headers.hashes = nil
	headers.times = nil
}

type mockHeaders struct {
	height int
	hashes []*chainhash.Hash
	times  []uint32
}

func (h *mockHeaders) LastHeight(ctx context.Context) int {
	return h.height
}

func (h *mockHeaders) Hash(ctx context.Context, height int) (*chainhash.Hash, error) {
	if height > h.height {
		return nil, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return nil, errors.New("Hash unavailable")
	}
	return h.hashes[h.height-height], nil
}

func (h *mockHeaders) Time(ctx context.Context, height int) (uint32, error) {
	if height > h.height {
		return 0, errors.New("Above current height")
	}
	if h.height-height >= len(h.hashes) {
		return 0, errors.New("Time unavailable")
	}
	return h.times[h.height-height], nil
}

func randomHash() *chainhash.Hash {
	data := make([]byte, 32)
	for i, _ := range data {
		data[i] = byte(r.Intn(256))
	}
	result, _ := chainhash.NewHash(data)
	return result
}

func mockUpHeaderHashes(ctx context.Context, height, count int) error {
	headers.height = height
	headers.hashes = nil
	headers.times = nil

	timestamp := uint32(time.Now().Unix())
	for i := 0; i < count; i++ {
		headers.hashes = append(headers.hashes, randomHash())
		headers.times = append(headers.times, timestamp)
		timestamp -= 600
	}
	return nil
}
