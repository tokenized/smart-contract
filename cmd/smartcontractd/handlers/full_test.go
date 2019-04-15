package handlers

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/txbuilder"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

func TestFull(t *testing.T) {
	ctx, span := trace.StartSpan(context.Background(), "TestFull")
	defer span.End()

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(os.Stdout)
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.Main.MinLevel = logger.LevelDebug

	ctx = logger.ContextWithLogConfig(ctx, logConfig)

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	// Setup message processing types.
	config := node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		FeeValue:           10000,
		DustLimit:          256,
		ChainParams:        config.NewChainParams("mainnet"),
		FeeRate:            1.0,
		RequestTimeout:     1000000000000,
	}

	// Fee Key
	var key *btcec.PrivateKey
	var err error
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to generate fee key : %s", err)
		return
	}
	feeKey := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(feeKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	feeKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &config.ChainParams)
	if err != nil {
		t.Errorf("Failed to create fee address : %s", err)
		return
	}
	logger.Info(ctx, "Fee PKH : %x", feeKey.Address.ScriptAddress())

	// Issuer Key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to generate issuer key : %s", err)
		return
	}
	issuerKey := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(issuerKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	issuerKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &config.ChainParams)
	if err != nil {
		t.Errorf("Failed to create issuer address : %s", err)
		return
	}
	logger.Info(ctx, "Issuer PKH : %x", issuerKey.Address.ScriptAddress())

	config.FeePKH = protocol.PublicKeyHashFromBytes(feeKey.Address.ScriptAddress())

	// Contract key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to generate contract key : %s", err)
		return
	}
	contractKey := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}

	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(contractKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	contractKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &config.ChainParams)
	if err != nil {
		t.Errorf("Failed to create address : %s", err)
		return
	}
	logger.Info(ctx, "Contract PKH : %x", contractKey.Address.ScriptAddress())

	masterWallet := wallet.New()
	wif, err := btcutil.NewWIF(contractKey.PrivateKey, &config.ChainParams, true)
	if err := masterWallet.Register(wif.String()); err != nil {
		t.Errorf("Failed to create wallet : %s", err)
		return
	}

	masterDB, err := db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   "./tmp",
	})
	if err != nil {
		t.Errorf("Failed to register DB : %s", err)
	}
	defer masterDB.Close()

	tracer := listeners.NewTracer()
	sch := scheduler.Scheduler{}
	utxos, err := utxos.Load(ctx, masterDB)
	if err != nil {
		t.Errorf("Failed to load UTXOs : %s", err)
		return
	}

	api, err := API(ctx, masterWallet, &config, masterDB, tracer, &sch, nil, utxos)
	if err != nil {
		t.Errorf("Failed to configure API : %s", err)
		return
	}

	responseWriter := node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}

	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	// Create Offer message
	offerData := protocol.ContractOffer{
		ContractName: "Test Name",
		BodyOfAgreementType: 0,
		BodyOfAgreement: []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: protocol.Entity{Administration: []protocol.Administrator{
			protocol.Administrator{Type: 1, Name: "John Smith"},
		}},
		VotingSystems: []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000}},
	}

	// Define permissions for contract fields
	permissions := make([]protocol.Permission, 21)
	for i, _ := range permissions {
		permissions[i].Permitted = false      // Issuer can't update field without proposal
		permissions[i].IssuerProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	offerData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		t.Errorf("Failed to serialize contract auth flags : %s", err)
		return
	}

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	var offerInputHash chainhash.Hash

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&offerInputHash, 0), make([]byte, 130)))

	// To contract
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&offerData)
	if err != nil {
		t.Errorf("Serialize offer : %s", err)
		return
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	offerItx := inspector.Transaction{
		Hash:     offerTx.TxHash(),
		MsgTx:    offerTx,
		MsgProto: &offerData,
	}

	offerItxInput := inspector.Input{
		Address: issuerKey.Address,
		Index:   0,
		Value:   101000,
		UTXO: inspector.UTXO{
			Hash:     offerInputHash,
			PkScript: offerTx.TxOut[0].PkScript,
			Index:    0,
			Value:    101000,
		},
		FullTx: offerTx,
	}
	offerItx.Inputs = append(offerItx.Inputs, offerItxInput)

	offerItxOutput := inspector.Output{
		Address: contractKey.Address,
		Index:   0,
		Value:   100000,
		UTXO: inspector.UTXO{
			Hash:     offerTx.TxHash(),
			PkScript: offerTx.TxOut[0].PkScript,
			Index:    0,
			Value:    100000,
		},
	}
	offerItx.Outputs = append(offerItx.Outputs, offerItxOutput)

	offerItxMsgOutput := inspector.Output{
		Address: nil,
		Index:   1,
		Value:   0,
		UTXO: inspector.UTXO{
			Hash:     offerTx.TxHash(),
			PkScript: offerTx.TxOut[1].PkScript,
			Index:    1,
			Value:    0,
		},
	}
	offerItx.Outputs = append(offerItx.Outputs, offerItxMsgOutput)

	contractHandler := Contract{
		MasterDB: masterDB,
		Config:   &config,
	}
	c := &contractHandler

	err = c.OfferRequest(ctx, &responseWriter, &offerItx, &contractKey)
	if err != nil {
		t.Errorf("Handle contract offer : %s", err)
		return
	}

	// Check response
	if len(responses) != 1 {
		t.Errorf("Handle contract offer created no response : %s", err)
		return
	}
}

var responses []*wire.MsgTx

func respondTx(ctx context.Context, tx *wire.MsgTx) error {
	responses = append(responses, tx)
	return nil
}
