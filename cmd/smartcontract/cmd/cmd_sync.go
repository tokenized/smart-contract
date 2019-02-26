package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/handlers"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/rpcnode"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
)

const (
	FlagDebugMode = "debug"
)

var cmdSync = &cobra.Command{
	Use:   "sync",
	Short: "Syncronize contract state with the network",
	RunE: func(c *cobra.Command, args []string) error {
		debugMode, _ := c.Flags().GetBool(FlagDebugMode)

		if debugMode {
			fmt.Println("Debug mode enabled!")
		}

		ctx := Context()

		// -------------------------------------------------------------------------
		// Configuration

		var cfg struct {
			Contract struct {
				PrivateKey   string `envconfig:"PRIV_KEY"`
				OperatorName string `envconfig:"OPERATOR_NAME"`
				Version      string `envconfig:"VERSION"`
				FeeAddress   string `envconfig:"FEE_ADDRESS"`
				FeeAmount    uint64 `envconfig:"FEE_VALUE"` // TODO rename FEE_AMOUNT
			}
			SpvNode struct {
				Address   string `envconfig:"NODE_ADDRESS"`
				UserAgent string `envconfig:"NODE_USER_AGENT"`
			}
			RpcNode struct {
				Host     string `envconfig:"RPC_HOST"`
				Username string `envconfig:"RPC_USERNAME"`
				Password string `envconfig:"RPC_PASSWORD"`
			}
			NodeStorage struct {
				Region    string `default:"ap-southeast-2" envconfig:"NODE_STORAGE_REGION"`
				AccessKey string `envconfig:"NODE_STORAGE_ACCESS_KEY"`
				Secret    string `envconfig:"NODE_STORAGE_SECRET"`
				Bucket    string `default:"standalone" envconfig:"NODE_STORAGE_BUCKET"`
				Root      string `default:"./tmp" envconfig:"NODE_STORAGE_ROOT"`
			}
			Storage struct {
				Region    string `default:"ap-southeast-2" envconfig:"CONTRACT_STORAGE_REGION"`
				AccessKey string `envconfig:"CONTRACT_STORAGE_ACCESS_KEY"`
				Secret    string `envconfig:"CONTRACT_STORAGE_SECRET"`
				Bucket    string `default:"standalone" envconfig:"CONTRACT_STORAGE_BUCKET"`
				Root      string `default:"./tmp" envconfig:"CONTRACT_STORAGE_ROOT"`
			}
		}

		if err := envconfig.Process("API", &cfg); err != nil {
			fmt.Printf("main : Parsing Config : %v", err)
		}

		// -------------------------------------------------------------------------
		// RPC Node

		rpcConfig := rpcnode.NewConfig(cfg.RpcNode.Host,
			cfg.RpcNode.Username,
			cfg.RpcNode.Password)

		rpcNode, err := rpcnode.NewNode(rpcConfig)
		if err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Wallet

		masterWallet := wallet.New()
		if err := masterWallet.Register(cfg.Contract.PrivateKey); err != nil {
			panic(err)
		}

		// -------------------------------------------------------------------------
		// Start Database / Storage

		fmt.Println("main : Started : Initialize Database")

		masterDB, err := db.New(&db.StorageConfig{
			Region:    cfg.Storage.Region,
			AccessKey: cfg.Storage.AccessKey,
			Secret:    cfg.Storage.Secret,
			Bucket:    cfg.Storage.Bucket,
			Root:      cfg.Storage.Root,
		})
		if err != nil {
			fmt.Printf("main : Register DB : %v", err)
		}
		defer masterDB.Close()

		// -------------------------------------------------------------------------
		// Node Config

		appConfig := &node.Config{
			ContractProviderID: cfg.Contract.OperatorName,
			Version:            cfg.Contract.Version,
			FeeAddress:         cfg.Contract.FeeAddress,
			FeeValue:           cfg.Contract.FeeAmount,
			DustLimit:          546,
		}

		// -------------------------------------------------------------------------
		// Register Hooks

		log := log.New(os.Stdout, "CLI : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

		txHandler := handlers.API(log, masterWallet, appConfig, masterDB)

		// Oldest -> Newest
		listResults, err := rpcNode.ListTransactions(ctx)
		if err != nil {
			return err
		}

		for _, rtx := range listResults {
			// now := time.Unix(rtx.Time, 0)

			hash, err := chainhash.NewHashFromStr(rtx.TxID)
			if err != nil {
				continue
			}

			fmt.Printf("Processing tx %v\n", hash)

			// Get transaction
			tx, err := rpcNode.GetTX(ctx, hash)
			if err != nil {
				continue
			}

			// Check if transaction relates to protocol
			itx, err := inspector.NewTransactionFromWire(ctx, tx)
			if err != nil {
				continue
			}

			// Prefilter out non-protocol messages, responses only
			if !itx.IsTokenized() || !itx.IsOutgoingMessageType() {
				continue
			}

			// Promote TX
			if err := itx.Promote(ctx, rpcNode); err != nil {
				return err
			}

			// Trigger "see" event
			if err := txHandler.Trigger(ctx, protomux.SEE, itx); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	cmdSync.Flags().Bool(FlagDebugMode, false, "Debug mode")
}
