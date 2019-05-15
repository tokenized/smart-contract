package tests

import (
	"context"
	"crypto/sha256"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

type Test struct {
	Context      context.Context
	Headers      *mockHeaders
	RPCNode      *mockRpcNode
	NodeConfig   node.Config
	MasterKey    *wallet.RootKey
	ContractKey  *wallet.RootKey
	FeeKey       *wallet.RootKey
	Master2Key   *wallet.RootKey
	Contract2Key *wallet.RootKey
	Fee2Key      *wallet.RootKey
	UTXOs        *utxos.UTXOs
	Wallet       *wallet.Wallet
	MasterDB     *db.DB
	Scheduler    *scheduler.Scheduler
	schStarted   bool
	path         string
}

func New(logToStdOut bool) *Test {

	// Random value used by helpers
	testHelperRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	// =========================================================================
	// Logging

	var ctx context.Context
	if logToStdOut {
		ctx = node.ContextWithProductionLogger(NewContext(), os.Stdout)
	} else {
		ctx = node.ContextWithNoLogger(NewContext())
	}

	// ============================================================
	// Node

	nodeConfig := node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		DustLimit:          256,
		ChainParams:        config.NewChainParams("mainnet"),
		FeeRate:            1.0,
		RequestTimeout:     1000000000000,
		IsTest:             true,
	}

	feeKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate fee key : %v", err)
		return nil
	}

	fee2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate fee 2 key : %v", err)
		return nil
	}

	nodeConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeKey.Address.ScriptAddress())

	rpcNode := newMockRpcNode(&nodeConfig.ChainParams)

	// ============================================================
	// Database

	path := "./tmp"
	masterDB, err := db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   path,
	})
	if err != nil {
		node.LogError(ctx, "main : Failed to create DB : %v", err)
		return nil
	}

	// ============================================================
	// Wallet

	testUTXOs, err := utxos.Load(ctx, masterDB)
	if err != nil {
		node.LogError(ctx, "main : Failed to load UTXOs : %v", err)
		return nil
	}

	masterKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate master key : %v", err)
		return nil
	}

	contractKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate contract key : %v", err)
		return nil
	}

	testWallet := wallet.New()
	if err := testWallet.Add(contractKey); err != nil {
		node.LogError(ctx, "main : Failed to add contract key to wallet : %v", err)
		return nil
	}

	master2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate master 2 key : %v", err)
		return nil
	}

	contract2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		node.LogError(ctx, "main : Failed to generate contract 2 key : %v", err)
		return nil
	}

	if err := testWallet.Add(contract2Key); err != nil {
		node.LogError(ctx, "main : Failed to add contract 2 key to wallet : %v", err)
		return nil
	}

	// ============================================================
	// Scheduler

	testScheduler := &scheduler.Scheduler{}

	go func() {
		if err := testScheduler.Run(ctx); err != nil {
			node.LogError(ctx, "Scheduler failed : %s", err)
		}
		node.Log(ctx, "Scheduler finished")
	}()

	// ============================================================
	// Result

	return &Test{
		Context:      ctx,
		Headers:      newMockHeaders(),
		RPCNode:      rpcNode,
		NodeConfig:   nodeConfig,
		MasterKey:    masterKey,
		ContractKey:  contractKey,
		FeeKey:       feeKey,
		Master2Key:   master2Key,
		Contract2Key: contract2Key,
		Fee2Key:      fee2Key,
		Wallet:       testWallet,
		MasterDB:     masterDB,
		UTXOs:        testUTXOs,
		Scheduler:    testScheduler,
		schStarted:   true,
		path:         path,
	}
}

// Reset is used to reset the test state complete.
func (test *Test) Reset() error {
	test.Headers.Reset()
	return test.ResetDB()
}

// ResetDB clears all the data in the database.
func (test *Test) ResetDB() error {
	return os.RemoveAll(filepath.FromSlash(test.path))
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

// GenerateKey does something
func GenerateKey(chainParams chaincfg.Params) (*wallet.RootKey, error) {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate key")
	}

	result := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}

	hash256 := sha256.Sum256(result.PublicKey.SerializeCompressed())
	hash160 := ripemd160.New()
	hash160.Write(hash256[:])
	result.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &chainParams)
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
