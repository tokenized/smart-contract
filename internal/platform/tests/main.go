package tests

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"os"
	"runtime/debug"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/pkg/errors"
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
	"golang.org/x/crypto/ripemd160"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

type Test struct {
	Context     context.Context
	RPCNode     *mockRpcNode
	NodeConfig  node.Config
	ContractKey *wallet.RootKey
	FeeKey      *wallet.RootKey
	Wallet      *wallet.Wallet
	MasterDB    *db.DB
	UTXOs       *utxos.UTXOs
	Scheduler   *scheduler.Scheduler
	schStarted  bool
}

func New() *Test {

	// =========================================================================
	// Logging

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(os.Stdout)
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)

	ctx := logger.ContextWithLogConfig(NewContext(), logConfig)

	// ============================================================
	// Node

	nodeConfig := node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		DustLimit:          256,
		ChainParams:        config.NewChainParams("mainnet"),
		FeeRate:            1.0,
		RequestTimeout:     1000000000000,
		FeeValue:           10000,
	}

	feeKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate fee key : %v", err)
	}

	nodeConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeKey.Address.ScriptAddress())

	rpcNode := &mockRpcNode{
		params: &nodeConfig.ChainParams,
	}

	// ============================================================
	// Database

	masterDB, err := db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   "./tmp",
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

	contractKey, err := GenerateKey(nodeConfig.ChainParams)
	if err != nil {
		logger.Fatal(ctx, "main : Failed to generate contract key : %v", err)
	}

	testWallet := wallet.New()
	if err := testWallet.Add(contractKey); err != nil {
		logger.Fatal(ctx, "main : Failed to create wallet : %v", err)
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
		Context:     ctx,
		RPCNode:     rpcNode,
		NodeConfig:  nodeConfig,
		ContractKey: contractKey,
		FeeKey:      feeKey,
		Wallet:      testWallet,
		MasterDB:    masterDB,
		UTXOs:       testUTXOs,
		Scheduler:   testScheduler,
		schStarted:  true,
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
