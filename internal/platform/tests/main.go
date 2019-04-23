package tests

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
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

	// =========================================================================
	// Logging

	var ctx context.Context
	if logToStdOut {
		logConfig := logger.NewDevelopmentConfig()
		logConfig.Main.SetWriter(os.Stdout)
		logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
		logConfig.Main.MinLevel = logger.LevelDebug
		logConfig.EnableSubSystem(txbuilder.SubSystem)
		logConfig.EnableSubSystem(spynode.SubSystem)

		ctx = logger.ContextWithLogConfig(NewContext(), logConfig)
	} else {
		ctx = NewContext()
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
	}

	feeKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate fee key : %v", err)
	}

	fee2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate fee 2 key : %v", err)
	}

	nodeConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeKey.Address.ScriptAddress())

	rpcNode := &mockRpcNode{
		params: &nodeConfig.ChainParams,
	}

	// ============================================================
	// Database

	path := "./tmp"
	masterDB, err := db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   path,
	})
	if err != nil {
		logger.Fatal(ctx, "main : Failed to create DB : %v", err)
	}

	// ============================================================
	// Wallet

	testUTXOs, err := utxos.Load(ctx, masterDB)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to load UTXOs : %v", err)
	}

	masterKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate master key : %v", err)
	}

	contractKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate contract key : %v", err)
	}

	testWallet := wallet.New()
	if err := testWallet.Add(contractKey); err != nil {
		logger.Fatal(ctx, "main : Failed to add contract key to wallet : %v", err)
	}

	master2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate master 2 key : %v", err)
	}

	contract2Key, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate contract 2 key : %v", err)
	}

	if err := testWallet.Add(contract2Key); err != nil {
		logger.Fatal(ctx, "main : Failed to add contract 2 key to wallet : %v", err)
	}

	// ============================================================
	// Scheduler

	testScheduler := &scheduler.Scheduler{}

	go func() {
		if err := testScheduler.Run(ctx); err != nil {
			logger.Error(ctx, "Scheduler failed : %s", err)
		}
		logger.Info(ctx, "Scheduler finished")
	}()

	// ============================================================
	// Result

	return &Test{
		Context:      ctx,
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
	key, err := btcec.NewPrivateKey(elliptic.P256())
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
func Recover(t *testing.T) {
	if r := recover(); r != nil {
		t.Fatal("Unhandled Exception:", string(debug.Stack()))
	}
}

// ResetDB clears all the data in the database.
func (test *Test) ResetDB() {
	os.RemoveAll(filepath.FromSlash(test.path))
}
