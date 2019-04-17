package handlers

import (
	"bytes"
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/utxos"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/txbuilder"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ripemd160"
)

var responses []*wire.MsgTx
var nodeConfig node.Config
var cache cacheNode
var userKey wallet.RootKey
var issuerKey wallet.RootKey
var contractKey wallet.RootKey
var feeKey wallet.RootKey
var masterWallet *wallet.Wallet
var masterDB *db.DB
var tracer *listeners.Tracer
var sch scheduler.Scheduler
var utxoSet *utxos.UTXOs
var api protomux.Handler
var assetCode protocol.AssetCode
var voteTxId protocol.TxId
var voteResultTxId protocol.TxId
var freezeTxId protocol.TxId
var tokenQty uint64

func TestFull(t *testing.T) {
	ctx, span := trace.StartSpan(context.Background(), "Test.Full")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(os.Stdout)
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.Main.MinLevel = logger.LevelDebug
	logConfig.EnableSubSystem(txbuilder.SubSystem)

	ctx = logger.ContextWithLogConfig(ctx, logConfig)

	var err error
	err = setup(ctx, t)
	if err != nil {
		t.Errorf("Setup failed : %s", err)
		return
	}

	defer masterDB.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := sch.Run(ctx); err != nil {
			t.Errorf("Scheduler failed : %s", err)
		}
		t.Logf("Scheduler finished")
	}()

	if err := createContract(ctx, t); err != nil {
		t.Errorf("Failed to create contract : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := createAsset(ctx, t); err != nil {
		t.Errorf("Failed to create asset : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := transferTokens(ctx, t); err != nil {
		t.Errorf("Failed to transfer tokens : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := holderProposal(ctx, t); err != nil {
		t.Errorf("Failed holder proposal : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	if err := sendBallot(ctx, t, issuerPKH, "A"); err != nil {
		t.Errorf("Failed to submit issuer ballot : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if err := sendBallot(ctx, t, userPKH, "B"); err != nil {
		t.Errorf("Failed to submit user ballot : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := processVoteResult(ctx, t); err != nil {
		t.Errorf("Failed to process vote result : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := proposalAmendment(ctx, t); err != nil {
		t.Errorf("Failed proposal amendment : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := freezeOrder(ctx, t); err != nil {
		t.Errorf("Failed freeze order : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if err := thawOrder(ctx, t); err != nil {
		t.Errorf("Failed thaw order : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	sch.Stop(ctx)
	wg.Wait()
}

func setup(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.setup")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	// Setup message processing types.
	nodeConfig = node.Config{
		ContractProviderID: "TokenizedTest",
		Version:            "TestVersion",
		FeeValue:           10000,
		DustLimit:          256,
		ChainParams:        config.NewChainParams("mainnet"),
		FeeRate:            1.0,
		RequestTimeout:     1000000000000,
	}

	cache = cacheNode{params: &nodeConfig.ChainParams}

	// User Key
	var key *btcec.PrivateKey
	var err error
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return errors.Wrap(err, "Failed to generate user key")
	}
	userKey = wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(userKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	userKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &nodeConfig.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to generate user address")
	}
	t.Logf("User PKH : %x", userKey.Address.ScriptAddress())

	// Fee Key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return errors.Wrap(err, "Failed to generate fee key")
	}
	feeKey = wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(feeKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	feeKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &nodeConfig.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to generate fee address")
	}
	t.Logf("Fee PKH : %x", feeKey.Address.ScriptAddress())

	// Issuer Key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return errors.Wrap(err, "Failed to generate issuer key")
	}
	issuerKey = wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(issuerKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	issuerKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &nodeConfig.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to generate issuer address")
	}
	t.Logf("Issuer PKH : %x", issuerKey.Address.ScriptAddress())

	nodeConfig.FeePKH = protocol.PublicKeyHashFromBytes(feeKey.Address.ScriptAddress())

	// Contract key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		return errors.Wrap(err, "Failed to generate contract key")
	}
	contractKey = wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}

	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(contractKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	contractKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &nodeConfig.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to generate contract address")
	}

	masterWallet = wallet.New()
	wif, err := btcutil.NewWIF(contractKey.PrivateKey, &nodeConfig.ChainParams, true)
	if err := masterWallet.Register(wif.String()); err != nil {
		return errors.Wrap(err, "Failed to create wallet")
	}
	contractKey = *masterWallet.ListAll()[0]
	t.Logf("Contract PKH : %x", contractKey.Address.ScriptAddress())

	for _, walletKey := range masterWallet.ListAll() {
		t.Logf("Wallet Key : %s", walletKey.Address.String())
	}

	masterDB, err = db.New(&db.StorageConfig{
		Bucket: "standalone",
		Root:   "./tmp",
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create DB")
	}

	tracer = listeners.NewTracer()
	sch = scheduler.Scheduler{}

	utxoSet, err = utxos.Load(ctx, masterDB)
	if err != nil {
		return errors.Wrap(err, "Failed to load UTXOs")
	}

	api, err = API(ctx, masterWallet, &nodeConfig, masterDB, tracer, &sch, nil, utxoSet)
	if err != nil {
		return errors.Wrap(err, "Failed to configure API")
	}
	api.SetResponder(respondTx)

	tokenQty = 1000
	return nil
}

func createContract(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.ContractOffer")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	// ********************************************************************************************
	// Create Contract Offer message
	offerData := protocol.ContractOffer{
		ContractName:           "Test Name",
		BodyOfAgreementType:    0,
		SupportingDocsFileType: 1,
		BodyOfAgreement:        []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: protocol.Entity{
			Type:           'I',
			Administration: []protocol.Administrator{protocol.Administrator{Type: 1, Name: "John Smith"}},
		},
		VotingSystems:  []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000}},
		HolderProposal: true,
	}

	// Define permissions for contract fields
	permissions := make([]protocol.Permission, 21)
	for i, _ := range permissions {
		permissions[i].Permitted = false      // Issuer can't update field without proposal
		permissions[i].IssuerProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = true  // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	var err error
	offerData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize contract auth flags")
	}

	// Create funding tx
	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	var offerInputHash chainhash.Hash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&offerInputHash, 0), make([]byte, 130)))

	// To contract
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&offerData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize offer")
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create itx")
	}

	err = offerItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote itx")
	}

	cache.AddTX(ctx, offerTx)

	err = api.Trigger(ctx, protomux.SEE, offerItx)
	if err == nil {
		return errors.New("Accepted invalid contract offer")
	}

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		return errors.New("Handle contract offer created no reject response")
	}

	var responseMsg protocol.OpReturnMessage
	response := responses[0].Copy()
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		return errors.New("Contract offer response doesn't contain tokenized op return")
	}
	if responseMsg.Type() != "M2" {
		return fmt.Errorf("Contract offer response not a reject : %s", responseMsg.Type())
	}
	reject, ok := responseMsg.(*protocol.Rejection)
	if !ok {
		return errors.New("Failed to convert response to rejection")
	}
	if reject.RejectionCode != protocol.RejectMsgMalformed {
		return errors.New("Wrong reject code for contract offer reject")
	}

	t.Logf("Invalid Contract offer rejection : (%d) %s", reject.RejectionCode, reject.Message)

	// ********************************************************************************************
	// Correct Contract Offer
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	offerData.BodyOfAgreementType = 2

	// Reserialize and update tx
	script, err = protocol.Serialize(&offerData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize offer")
	}
	offerTx.TxOut[1].PkScript = script

	offerItx, err = inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create itx")
	}

	err = offerItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote itx")
	}

	cache.AddTX(ctx, offerTx)

	// Resubmit to handler
	err = api.Trigger(ctx, protomux.SEE, offerItx)
	if err != nil {
		return errors.Wrap(err, "Failed to handle contract offer")
	}

	t.Logf("Contract offer accepted")

	if err := checkResponse(ctx, t, "C2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	return nil
}

func createAsset(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.AssetDefinition")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	contractPKH := protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, masterDB, contractPKH)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100001, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	assetCode = *protocol.AssetCodeFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})

	// Create AssetDefinition message
	assetData := protocol.AssetDefinition{
		AssetType:                  protocol.CodeShareCommon,
		AssetCode:                  assetCode,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		IssuerProposal:             true,
		TokenQty:                   1000,
	}

	assetPayloadData := protocol.ShareCommon{
		Ticker:      "TST",
		Description: "Test common shares",
	}
	assetData.AssetPayload, err = assetPayloadData.Serialize()
	if err != nil {
		return errors.Wrap(err, "Failed to serialize asset payload")
	}

	// Define permissions for asset fields
	permissions := make([]protocol.Permission, 13)
	for i, _ := range permissions {
		permissions[i].Permitted = false      // Issuer can't update field without proposal
		permissions[i].IssuerProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	assetData.AssetAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize asset auth flags")
	}

	// Build asset definition transaction
	assetTx := wire.NewMsgTx(2)

	var assetInputHash chainhash.Hash
	assetInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&assetInputHash, 0), make([]byte, 130)))

	// To contract
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&assetData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize offer")
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create asset itx")
	}

	err = assetItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote asset itx")
	}

	cache.AddTX(ctx, assetTx)

	err = api.Trigger(ctx, protomux.SEE, assetItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept asset definition")
	}

	t.Logf("Asset definition accepted")

	if err := checkResponse(ctx, t, "A2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	// Check issuer balance
	contractPKH = protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetData.AssetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != assetData.TokenQty {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, assetData.TokenQty)
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
	return nil
}

func transferTokens(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.Transfer")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100002, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     assetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, protocol.TokenReceiver{Index: 1, Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	var transferInputHash chainhash.Hash
	transferInputHash = fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// To user
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize transfer")
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create transfer itx")
	}

	err = transferItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote transfer itx")
	}

	cache.AddTX(ctx, transferTx)

	err = api.Trigger(ctx, protomux.SEE, transferItx)
	if err == nil {
		return errors.New("Accepted transfer with insufficient value")
	}

	if len(responses) != 0 {
		return errors.New("Handle asset transfer created reject response")
	}

	t.Logf("Underfunded asset transfer rejected with no response")

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 1000

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create transfer itx")
	}

	err = transferItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote transfer itx")
	}

	// Resubmit
	cache.AddTX(ctx, transferTx)

	err = api.Trigger(ctx, protomux.SEE, transferItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept transfer")
	}

	t.Logf("Transfer accepted")

	if err := checkResponse(ctx, t, "T2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != tokenQty-transferAmount {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, tokenQty-transferAmount)
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != transferAmount {
		return fmt.Errorf("User token balance incorrect : %d != %d", userBalance, transferAmount)
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
	t.Logf("User asset balance : %d", userBalance)
	return nil
}

func holderProposal(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.Proposal")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100003, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	proposalData := protocol.Proposal{
		Initiator:           1,
		AssetSpecificVote:   false,
		VoteSystem:          0,
		Specific:            true,
		VoteOptions:         "AB",
		VoteMax:             1,
		ProposalDescription: "Change contract name",
		VoteCutOffTimestamp: protocol.NewTimestamp(v.Now.Nano() + 500000000),
	}

	proposalData.ProposedAmendments = append(proposalData.ProposedAmendments, protocol.Amendment{
		FieldIndex: 0,
		Data:       []byte("Test Name 2"),
	})

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	var proposalInputHash chainhash.Hash
	proposalInputHash = fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(51000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&proposalData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize proposal")
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create proposal itx")
	}

	err = proposalItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote proposal itx")
	}

	cache.AddTX(ctx, proposalTx)

	err = api.Trigger(ctx, protomux.SEE, proposalItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept proposal")
	}

	t.Logf("Proposal accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		voteTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, "G2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	return nil
}

func sendBallot(ctx context.Context, t *testing.T, pkh *protocol.PublicKeyHash, vote string) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100006, txbuilder.P2PKHScriptForPKH(pkh.Bytes())))
	cache.AddTX(ctx, fundingTx)

	ballotData := protocol.BallotCast{
		VoteTxId: voteTxId,
		Vote:     vote,
	}

	// Build transaction
	ballotTx := wire.NewMsgTx(2)

	var ballotInputHash chainhash.Hash
	ballotInputHash = fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&ballotInputHash, 0), make([]byte, 130)))

	// To contract
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&ballotData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize ballot")
	}
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(0, script))

	ballotItx, err := inspector.NewTransactionFromWire(ctx, ballotTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create ballot itx")
	}

	err = ballotItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote ballot itx")
	}

	cache.AddTX(ctx, ballotTx)

	err = api.Trigger(ctx, protomux.SEE, ballotItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept ballot")
	}

	t.Logf("Ballot accepted")

	if err := checkResponse(ctx, t, "G4"); err != nil {
		return errors.Wrap(err, "Failed to check ballot counted result")
	}

	return nil
}

func processVoteResult(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		voteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, "G5"); err != nil {
		return errors.Wrap(err, "Failed to check vote result")
	}

	return nil
}

func proposalAmendment(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100007, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	amendmentData := protocol.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          voteResultTxId,
	}

	amendmentData.Amendments = append(amendmentData.Amendments, protocol.Amendment{
		FieldIndex: 0,
		Data:       []byte("Test Name 2"),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	var amendmentInputHash chainhash.Hash
	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&amendmentData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize amendment")
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create amendment itx")
	}

	err = amendmentItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote amendment itx")
	}

	cache.AddTX(ctx, amendmentTx)

	err = api.Trigger(ctx, protomux.SEE, amendmentItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept amendment")
	}

	t.Logf("Amendment accepted")

	if err := checkResponse(ctx, t, "C2"); err != nil {
		return errors.Wrap(err, "Failed to check amendment response")
	}

	return nil
}

func freezeOrder(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.freezeOrder")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100008, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        assetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize order")
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create order itx")
	}

	err = orderItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	cache.AddTX(ctx, orderTx)

	err = api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Freeze order accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		freezeTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, "E2"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if asset.CheckBalanceFrozen(ctx, as, userPKH, 100, v.Now) {
		return errors.New("User unfrozen balance too high")
	}

	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 50, v.Now) {
		return errors.New("User unfrozen balance not high enough")
	}

	return nil
}

func thawOrder(ctx context.Context, t *testing.T) error {
	ctx, span := trace.StartSpan(ctx, "Test.freezeOrder")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionThaw,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        assetCode,
		FreezeTxId:       freezeTxId,
		Message:          "Court order released",
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize order")
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		return errors.Wrap(err, "Failed to create order itx")
	}

	err = orderItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	cache.AddTX(ctx, orderTx)

	err = api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Thaw order accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		freezeTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, "E3"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 250, v.Now) {
		return errors.New("User balance not unfrozen")
	}

	return nil
}

func checkResponse(ctx context.Context, t *testing.T, responseCode string) error {
	ctx, span := trace.StartSpan(ctx, "Test.CheckResponse")
	defer span.End()

	v := node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	if len(responses) != 1 {
		return fmt.Errorf("%s Response not created", responseCode)
	}

	response := responses[0]
	responses = nil
	var responseMsg protocol.OpReturnMessage
	var err error
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		return fmt.Errorf("%s Response doesn't contain tokenized op return", responseCode)
	}
	if responseMsg.Type() != responseCode {
		return fmt.Errorf("Response is the wrong type : %s != %s", responseMsg.Type(), responseCode)
	}

	// Submit response
	responseItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		return errors.Wrap(err, "Failed to create response itx")
	}

	err = responseItx.Promote(ctx, &cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote response itx")
	}

	cache.AddTX(ctx, response)

	err = api.Trigger(ctx, protomux.SEE, responseItx)
	if err != nil {
		return errors.Wrap(err, "Failed to process response")
	}

	if len(responses) != 0 {
		return errors.New("Response created a response")
	}

	t.Logf("Response processed : %s", responseCode)
	return nil
}

func respondTx(ctx context.Context, tx *wire.MsgTx) error {
	responses = append(responses, tx)
	return nil
}

type cacheNode struct {
	txs    []*wire.MsgTx
	params *chaincfg.Params
}

func (cache *cacheNode) AddTX(ctx context.Context, tx *wire.MsgTx) error {
	cache.txs = append(cache.txs, tx)
	return nil
}

func (cache *cacheNode) GetTX(ctx context.Context, txid *chainhash.Hash) (*wire.MsgTx, error) {
	for _, tx := range cache.txs {
		hash := tx.TxHash()
		if bytes.Equal(hash[:], txid[:]) {
			return tx, nil
		}
	}
	return nil, errors.New("Couldn't find tx in cache")
}

func (cache *cacheNode) GetChainParams() *chaincfg.Params {
	return cache.params
}
