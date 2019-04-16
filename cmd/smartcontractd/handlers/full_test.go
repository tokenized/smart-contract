package handlers

import (
	"bytes"
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
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
	"github.com/btcsuite/btcd/chaincfg"
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
	logConfig.EnableSubSystem(txbuilder.SubSystem)

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

	cache := cacheNode{params: &config.ChainParams}

	// User Key
	var key *btcec.PrivateKey
	var err error
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to generate user key : %s", err)
		return
	}
	userKey := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(userKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	userKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &config.ChainParams)
	if err != nil {
		t.Errorf("Failed to create user address : %s", err)
		return
	}
	t.Logf("User PKH : %x", userKey.Address.ScriptAddress())

	// Fee Key
	key, err = btcec.NewPrivateKey(elliptic.P256())
	if err != nil {
		t.Errorf("Failed to generate fee key : %s", err)
		return
	}
	feeKey := wallet.RootKey{
		PrivateKey: key,
		PublicKey:  key.PubKey(),
	}
	hash160 = ripemd160.New()
	hash256 = sha256.Sum256(feeKey.PublicKey.SerializeCompressed())
	hash160.Write(hash256[:])
	feeKey.Address, err = btcutil.NewAddressPubKeyHash(hash160.Sum(nil), &config.ChainParams)
	if err != nil {
		t.Errorf("Failed to create fee address : %s", err)
		return
	}
	t.Logf("Fee PKH : %x", feeKey.Address.ScriptAddress())

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
	t.Logf("Issuer PKH : %x", issuerKey.Address.ScriptAddress())

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
	t.Logf("Contract PKH : %x", contractKey.Address.ScriptAddress())

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

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := sch.Run(ctx); err != nil {
			t.Errorf("Scheduler failed : %s", err)
		}
		t.Logf("Scheduler finished")
	}()

	utxos, err := utxos.Load(ctx, masterDB)
	if err != nil {
		t.Errorf("Failed to load UTXOs : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	api, err := API(ctx, masterWallet, &config, masterDB, tracer, &sch, nil, utxos)
	if err != nil {
		t.Errorf("Failed to configure API : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	responseWriter := node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	// Create funding tx
	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

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

	offerData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		t.Errorf("Failed to serialize contract auth flags : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	var offerInputHash chainhash.Hash
	offerInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&offerInputHash, 0), make([]byte, 130)))

	// To contract
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&offerData)
	if err != nil {
		t.Errorf("Failed to serialize offer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		t.Errorf("Failed to create itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = offerItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	contractHandler := Contract{
		MasterDB: masterDB,
		Config:   &config,
	}
	c := &contractHandler

	cache.AddTX(ctx, offerTx)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = c.OfferRequest(ctx, &responseWriter, offerItx, &contractKey)
	if err == nil {
		t.Errorf("Accepted invalid contract offer")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Errorf("Handle contract offer created no reject response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	var responseMsg protocol.OpReturnMessage
	response := responses[0]
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Errorf("Contract offer response doesn't contain tokenized op return")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if responseMsg.Type() != "M2" {
		t.Errorf("Contract offer response not a reject : %s", responseMsg.Type())
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	reject, ok := responseMsg.(*protocol.Rejection)
	if !ok {
		t.Errorf("Failed to convert response to rejection")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if reject.RejectionCode != protocol.RejectMsgMalformed {
		t.Errorf("Wrong reject code for contract offer reject")
		sch.Stop(ctx)
		wg.Wait()
		return
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
		t.Errorf("Serialize offer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	offerTx.TxOut[1].PkScript = script

	offerItx, err = inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		t.Errorf("Failed to create itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = offerItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	cache.AddTX(ctx, offerTx)

	// Resubmit to handler
	err = c.OfferRequest(ctx, &responseWriter, offerItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to handle contract offer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Contract offer accepted")

	if len(responses) != 1 {
		t.Errorf("Handle contract offer created no response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	response = responses[0]
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Errorf("Contract offer response doesn't contain tokenized op return")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if responseMsg.Type() != "C2" {
		t.Errorf("Contract offer response not a formation : %s", responseMsg.Type())
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	formation, ok := responseMsg.(*protocol.ContractFormation)
	if !ok {
		t.Errorf("Failed to convert response to formation")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	t.Logf("Contract formation processed : %s", formation.ContractName)

	// ********************************************************************************************
	// Submit contract formation response
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	formationItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Errorf("Failed to create formation itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = formationItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote formation itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	cache.AddTX(ctx, response)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = c.FormationResponse(ctx, &responseWriter, formationItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to handle contract formation : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if len(responses) != 0 {
		t.Errorf("Handle contract formation created a response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// ********************************************************************************************
	// Create asset
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx = wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100001, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	// Create AssetDefinition message
	assetData := protocol.AssetDefinition{
		AssetType:                  protocol.CodeShareCommon,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		IssuerProposal:             true,
		TokenQty:                   1000,
	}

	//assetData.AssetCode

	assetPayloadData := protocol.ShareCommon{
		Ticker:      "TST",
		Description: "Test common shares",
	}
	assetData.AssetPayload, err = assetPayloadData.Serialize()
	if err != nil {
		t.Errorf("Failed to serialize asset payload : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Define permissions for asset fields
	permissions = make([]protocol.Permission, 13)
	for i, _ := range permissions {
		permissions[i].Permitted = false      // Issuer can't update field without proposal
		permissions[i].IssuerProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	assetData.AssetAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		t.Errorf("Failed to serialize asset auth flags : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Build offer transaction
	assetTx := wire.NewMsgTx(2)

	var assetInputHash chainhash.Hash
	assetInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&assetInputHash, 0), make([]byte, 130)))

	// To contract
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(contractKey.Address.ScriptAddress())))

	// Data output
	script, err = protocol.Serialize(&assetData)
	if err != nil {
		t.Errorf("Failed to serialize offer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx)
	if err != nil {
		t.Errorf("Failed to create asset itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = assetItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote asset itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	assetHandler := Asset{
		MasterDB: masterDB,
		Config:   &config,
	}
	a := &assetHandler

	cache.AddTX(ctx, assetTx)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = a.DefinitionRequest(ctx, &responseWriter, assetItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to accept asset definition : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Asset definition accepted")

	if len(responses) != 1 {
		t.Errorf("Handle asset creation created no response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	response = responses[0]
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Errorf("Asset definition response doesn't contain tokenized op return")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if responseMsg.Type() != "A2" {
		t.Errorf("Asset definition response not a asset creation : %s", responseMsg.Type())
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	_, ok = responseMsg.(*protocol.AssetCreation)
	if !ok {
		t.Errorf("Failed to convert response to asset creation")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// ********************************************************************************************
	// Submit asset creation response
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	creationItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Errorf("Failed to create asset creation itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = creationItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote asset creation itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	cache.AddTX(ctx, response)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = a.CreationResponse(ctx, &responseWriter, creationItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to process asset creation : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Asset creation processed")

	if len(responses) != 0 {
		t.Errorf("Handle asset creation created a response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Check issuer balance
	contractPKH := protocol.PublicKeyHashFromBytes(contractKey.Address.ScriptAddress())
	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, masterDB, contractPKH, &assetData.AssetCode)
	if err != nil {
		t.Errorf("Failed to retrieve asset : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != assetData.TokenQty {
		t.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, assetData.TokenQty)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)

	// ********************************************************************************************
	// Transfer some tokens to user
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx = wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100002, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     assetData.AssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, protocol.TokenReceiver{Index: 1, Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build offer transaction
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
	script, err = protocol.Serialize(&transferData)
	if err != nil {
		t.Errorf("Failed to serialize transfer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Errorf("Failed to create transfer itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = transferItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote transfer itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	transferHandler := Transfer{
		handler:   api,
		MasterDB:  masterDB,
		Config:    &config,
		Headers:   nil,
		Tracer:    tracer,
		Scheduler: &sch,
	}
	th := &transferHandler

	cache.AddTX(ctx, transferTx)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = th.TransferRequest(ctx, &responseWriter, transferItx, &contractKey)
	if err == nil {
		t.Errorf("Accepted transfer with insufficient value")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	if len(responses) != 0 {
		t.Errorf("Handle asset transfer created reject response")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Underfunded asset transfer rejected with no response")

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 1000

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Errorf("Failed to create transfer itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = transferItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote transfer itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Resubmit
	cache.AddTX(ctx, transferTx)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = th.TransferRequest(ctx, &responseWriter, transferItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to accept transfer : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Transfer accepted")

	if len(responses) != 1 {
		t.Errorf("Handle transfer created no response : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	response = responses[0]
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Errorf("Transfer response doesn't contain tokenized op return")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if responseMsg.Type() != "T2" {
		t.Errorf("Transfer response not a settlement : %s", responseMsg.Type())
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	_, ok = responseMsg.(*protocol.Settlement)
	if !ok {
		t.Errorf("Failed to convert response to settlement")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Submit settlement response
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	settlementItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Errorf("Failed to create settlement itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = settlementItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote settlement itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	cache.AddTX(ctx, response)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = th.SettlementResponse(ctx, &responseWriter, settlementItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to process settlement : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Settlement processed")

	if len(responses) != 0 {
		t.Errorf("Handle settlement created a response : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// Check issuer and user balance
	as, err = asset.Retrieve(ctx, masterDB, contractPKH, &assetData.AssetCode)
	if err != nil {
		t.Errorf("Failed to retrieve asset : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	issuerBalance = asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != assetData.TokenQty-transferAmount {
		t.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, assetData.TokenQty-transferAmount)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != transferAmount {
		t.Errorf("User token balance incorrect : %d != %d", userBalance, transferAmount)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
	t.Logf("User asset balance : %d", userBalance)

	// ********************************************************************************************
	// Create holder proposal message
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	fundingTx = wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100003, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))
	cache.AddTX(ctx, fundingTx)

	proposalData := protocol.Proposal{
		Initiator:           1,
		AssetSpecificVote:   false,
		VoteSystem:          0,
		Specific:            true,
		VoteOptions:         "AB",
		VoteMax:             1,
		ProposalDescription: "Change contract name to Test Name 2",
		VoteCutOffTimestamp: protocol.NewTimestamp(v.Now.Nano() + 1000000000),
	}

	proposalData.ProposedAmendments = append(proposalData.ProposedAmendments, protocol.Amendment{
		FieldIndex: 0,
		Data:       []byte("Test Name 2"),
	})

	// Build offer transaction
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
	script, err = protocol.Serialize(&proposalData)
	if err != nil {
		t.Errorf("Failed to serialize proposal : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx)
	if err != nil {
		t.Errorf("Failed to create proposal itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = proposalItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote proposal itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	governanceHandler := Governance{
		handler:   api,
		MasterDB:  masterDB,
		Config:    &config,
		Scheduler: &sch,
	}
	g := &governanceHandler

	cache.AddTX(ctx, proposalTx)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = g.ProposalRequest(ctx, &responseWriter, proposalItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to accept proposal : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Proposal accepted")

	if len(responses) != 1 {
		t.Errorf("Handle proposal created no response : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	response = responses[0]
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Errorf("Proposal response doesn't contain tokenized op return")
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	if responseMsg.Type() != "G2" {
		t.Errorf("Proposal response not a vote : %s", responseMsg.Type())
		sch.Stop(ctx)
		wg.Wait()
		return
	}
	_, ok = responseMsg.(*protocol.Vote)
	if !ok {
		t.Errorf("Failed to convert response to vote")
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// ********************************************************************************************
	// Submit vote response
	v = node.Values{
		TraceID: span.SpanContext().TraceID.String(),
		Now:     protocol.CurrentTimestamp(),
	}
	ctx = context.WithValue(ctx, node.KeyValues, &v)

	voteItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Errorf("Failed to create vote itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	err = voteItx.Promote(ctx, &cache)
	if err != nil {
		t.Errorf("Failed to promote vote itx : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	cache.AddTX(ctx, response)

	responseWriter = node.ResponseWriter{
		Config: &config,
		Mux:    api,
	}
	// Set responder
	responseWriter.Mux.SetResponder(respondTx)

	err = g.VoteResponse(ctx, &responseWriter, voteItx, &contractKey)
	if err != nil {
		t.Errorf("Failed to process vote : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	t.Logf("Vote processed")

	if len(responses) != 0 {
		t.Errorf("Handle vote created a response : %s", err)
		sch.Stop(ctx)
		wg.Wait()
		return
	}

	// ********************************************************************************************
	// Submit ballot from issuer

	// ********************************************************************************************
	// Submit ballot from user

	// ********************************************************************************************
	// Check result
	// Wait for expiration
	time.Sleep(2 * time.Second)
	// TODO Run scheduler in another thread

}

var responses []*wire.MsgTx

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
