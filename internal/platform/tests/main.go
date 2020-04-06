package tests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

type Test struct {
	Context         context.Context
	Headers         *mockHeaders
	RPCNode         *mockRpcNode
	NodeConfig      node.Config
	MasterKey       *wallet.Key
	ContractKey     *wallet.Key
	FeeKey          *wallet.Key
	Master2Key      *wallet.Key
	Contract2Key    *wallet.Key
	Fee2Key         *wallet.Key
	UTXOs           *utxos.UTXOs
	Wallet          *wallet.Wallet
	MasterDB        *db.DB
	Scheduler       *scheduler.Scheduler
	HoldingsChannel *holdings.CacheChannel
	schStarted      bool
	path            string
}

func New(logFileName string) *Test {

	// Random value used by helpers
	testHelperRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	// =========================================================================
	// Logging

	var ctx context.Context
	if len(logFileName) > 0 {
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file : %v\n", err)
			return nil
		}
		ctx = node.ContextWithDevelopmentLogger(NewContext(), logFile, "text")
	} else {
		ctx = node.ContextWithNoLogger(NewContext())
	}

	// ============================================================
	// Node

	nodeConfig := node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		DustLimit:          256,
		Net:                bitcoin.MainNet,
		FeeRate:            1.0,
		MinFeeRate:         0.5,
		RequestTimeout:     1000000000000,
		IsTest:             true,
	}

	feeKey, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate fee key : %v\n", err)
		return nil
	}

	fee2Key, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate fee 2 key : %v\n", err)
		return nil
	}

	nodeConfig.FeeAddress, err = bitcoin.NewRawAddressPKH(bitcoin.Hash160(feeKey.Key.PublicKey().Bytes()))
	if err != nil {
		fmt.Printf("main : Failed to create fee 2 address : %v\n", err)
		return nil
	}

	rpcNode := newMockRpcNode()

	// ============================================================
	// Database

	path := "./tmp"
	masterDB, err := db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   path,
	})
	if err != nil {
		fmt.Printf("main : Failed to create DB : %v\n", err)
		return nil
	}

	// ============================================================
	// Wallet

	testUTXOs, err := utxos.Load(ctx, masterDB)
	if err != nil {
		fmt.Printf("main : Failed to load UTXOs : %v\n", err)
		return nil
	}

	masterKey, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate master key : %v\n", err)
		return nil
	}

	contractKey, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate contract key : %v\n", err)
		return nil
	}

	testWallet := wallet.New()
	if err := testWallet.Add(contractKey); err != nil {
		fmt.Printf("main : Failed to add contract key to wallet : %v\n", err)
		return nil
	}

	master2Key, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate master 2 key : %v\n", err)
		return nil
	}

	contract2Key, err := GenerateKey(nodeConfig.Net)
	if err != nil {
		fmt.Printf("main : Failed to generate contract 2 key : %v\n", err)
		return nil
	}

	if err := testWallet.Add(contract2Key); err != nil {
		fmt.Printf("main : Failed to add contract 2 key to wallet : %v\n", err)
		return nil
	}

	// ============================================================
	// Scheduler

	testScheduler := &scheduler.Scheduler{}

	go func() {
		if err := testScheduler.Run(ctx); err != nil {
			fmt.Printf("Scheduler failed : %s\n", err)
		}
		node.Log(ctx, "Scheduler finished")
	}()

	// ============================================================
	// Result

	result := Test{
		Context:         ctx,
		Headers:         newMockHeaders(),
		RPCNode:         rpcNode,
		NodeConfig:      nodeConfig,
		MasterKey:       masterKey,
		ContractKey:     contractKey,
		FeeKey:          feeKey,
		Master2Key:      master2Key,
		Contract2Key:    contract2Key,
		Fee2Key:         fee2Key,
		Wallet:          testWallet,
		MasterDB:        masterDB,
		UTXOs:           testUTXOs,
		Scheduler:       testScheduler,
		schStarted:      true,
		path:            path,
		HoldingsChannel: &holdings.CacheChannel{},
	}

	return &result
}

// Reset is used to reset the test state complete.
func (test *Test) Reset(ctx context.Context) error {
	test.Headers.Reset()
	return test.ResetDB(ctx)
}

// ResetDB clears all the data in the database.
func (test *Test) ResetDB(ctx context.Context) error {
	return test.MasterDB.Clear(ctx, "")
}

// TearDown is used for shutting down tests. Calling this should be
// done in a defer immediately after calling New.
func (test *Test) TearDown() {
	if test.schStarted {
		test.Scheduler.Stop(test.Context)
	}
	if test.MasterDB != nil {
		test.MasterDB.Close()
	}
}

// Context returns an app level context for testing.
func NewContext() context.Context {
	values := node.Values{
		TraceID: uuid.New().String(),
		Now:     protocol.CurrentTimestamp(),
	}

	return context.WithValue(context.Background(), node.KeyValues, &values)
}

// GenerateKey generates a new wallet key.
func GenerateKey(net bitcoin.Network) (*wallet.Key, error) {
	key, err := bitcoin.GenerateKey(net)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate key")
	}

	result := wallet.Key{
		Key: key,
	}

	result.Address, err = key.RawAddress()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create key address")
	}

	return &result, nil
}

// Recover is used to prevent panics from allowing the test to cleanup.
func Recover(t testing.TB) {
	if r := recover(); r != nil {
		t.Fatal("Unhandled Exception:", string(debug.Stack()))
	}
}
