package tests

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"golang.org/x/crypto/ripemd160"
)

type Test struct {
	logConfig   *logger.Config
	NodeConfig  node.Config
	ContractKey *wallet.RootKey
	FeeKey      *wallet.RootKey
	Wallet      *wallet.Wallet
	DB          *db.DB
	UTXOs       *utxos.UTXOs
	schStarted  bool
	Scheduler   scheduler.Scheduler
}

func (test *Test) Setup(ctx context.Context) error {
	test.logConfig = logger.NewDevelopmentConfig()
	test.logConfig.Main.SetWriter(os.Stdout)
	test.logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	test.logConfig.Main.MinLevel = logger.LevelDebug
	test.logConfig.EnableSubSystem(txbuilder.SubSystem)
	test.logConfig.EnableSubSystem(spynode.SubSystem)

	ctx = logger.ContextWithLogConfig(ctx, test.logConfig)

	test.NodeConfig = node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		DustLimit:          256,
		ChainParams:        config.NewChainParams("mainnet"),
		FeeRate:            1.0,
		RequestTimeout:     1000000000000,
		FeeValue:           10000,
	}

	var err error
	test.FeeKey, err = test.GenerateKey()
	if err != nil {
		return errors.Wrap(err, "Failed to generate fee key")
	}
	test.NodeConfig.FeePKH = protocol.PublicKeyHashFromBytes(test.FeeKey.Address.ScriptAddress())

	test.DB, err = db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   "./tmp",
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create DB")
	}

	test.UTXOs, err = utxos.Load(ctx, test.DB)
	if err != nil {
		return errors.Wrap(err, "Failed to load UTXOs")
	}

	test.ContractKey, err = test.GenerateKey()
	if err != nil {
		return errors.Wrap(err, "Failed to generate contract key")
	}
	test.Wallet = wallet.New()
	if err := test.Wallet.Add(test.ContractKey); err != nil {
		return errors.Wrap(err, "Failed to create wallet")
	}

	test.schStarted = true
	go func() {
		if err := test.Scheduler.Run(ctx); err != nil {
			logger.Error(ctx, "Scheduler failed : %s", err)
		}
		logger.Info(ctx, "Scheduler finished")
	}()

	return nil
}

func (test *Test) Close(ctx context.Context) {
	if test.schStarted {
		test.Scheduler.Stop(ctx)
	}
	if test.DB != nil {
		test.DB.Close()
	}
}

func (test *Test) Context(ctx context.Context, traceID string) context.Context {
	v := node.Values{
		TraceID: traceID,
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	return logger.ContextWithLogConfig(ctx, test.logConfig)
}

func (test *Test) GenerateKey() (*wallet.RootKey, error) {
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
	result.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &test.NodeConfig.ChainParams)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create key address")
	}

	return &result, nil
}
