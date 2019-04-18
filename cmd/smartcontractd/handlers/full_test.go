package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
)

var responses []*wire.MsgTx

type Test struct {
	tests.Test

	userKey   *wallet.RootKey
	issuerKey *wallet.RootKey

	cache  cacheNode
	tracer *listeners.Tracer
	api    protomux.Handler

	tokenQty       uint64
	assetCode      protocol.AssetCode
	voteTxId       protocol.TxId
	voteResultTxId protocol.TxId
	freezeTxId     protocol.TxId
}

func TestFull(t *testing.T) {
	ctx, span := trace.StartSpan(context.Background(), "Test.Full")
	defer span.End()

	test := &Test{}
	err := test.Setup(ctx)
	if err != nil {
		t.Errorf("Setup failed : %s", err)
		if test != nil {
			test.Close(ctx)
		}
		return
	}

	test.tracer = listeners.NewTracer()
	test.cache.params = &test.NodeConfig.ChainParams

	test.api, err = API(ctx, test.Wallet, &test.NodeConfig, test.DB, test.tracer, &test.Scheduler, nil, test.UTXOs)
	if err != nil {
		test.Close(ctx)
		t.Errorf("Failed to configure api : %s", err)
	}

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())
	defer test.Close(ctx)

	test.userKey, err = test.GenerateKey()
	if err != nil {
		t.Errorf("Failed to generate user key : %s", err)
		return
	}

	test.issuerKey, err = test.GenerateKey()
	if err != nil {
		t.Errorf("Failed to generate issuer key : %s", err)
		return
	}

	t.Logf("User PKH : %x", test.userKey.Address.ScriptAddress())
	t.Logf("Issuer PKH : %x", test.issuerKey.Address.ScriptAddress())
	t.Logf("Contract PKH : %x", test.ContractKey.Address.ScriptAddress())

	test.api.SetResponder(respondTx)
	test.tokenQty = 1000

	if err := createContract(ctx, t, test); err != nil {
		t.Errorf("Failed to create contract : %s", err)
		return
	}

	if err := createAsset(ctx, t, test); err != nil {
		t.Errorf("Failed to create asset : %s", err)
		return
	}

	if err := transferTokens(ctx, t, test); err != nil {
		t.Errorf("Failed to transfer tokens : %s", err)
		return
	}

	if err := holderProposal(ctx, t, test); err != nil {
		t.Errorf("Failed holder proposal : %s", err)
		return
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	if err := sendBallot(ctx, t, test, issuerPKH, "A"); err != nil {
		t.Errorf("Failed to submit issuer ballot : %s", err)
		return
	}

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	if err := sendBallot(ctx, t, test, userPKH, "B"); err != nil {
		t.Errorf("Failed to submit user ballot : %s", err)
		return
	}

	if err := processVoteResult(ctx, t, test); err != nil {
		t.Errorf("Failed to process vote result : %s", err)
		return
	}

	if err := proposalAmendment(ctx, t, test); err != nil {
		t.Errorf("Failed proposal amendment : %s", err)
		return
	}

	if err := freezeOrder(ctx, t, test); err != nil {
		t.Errorf("Failed freeze order : %s", err)
		return
	}

	if err := thawOrder(ctx, t, test); err != nil {
		t.Errorf("Failed thaw order : %s", err)
		return
	}

	if err := confiscateOrder(ctx, t, test); err != nil {
		t.Errorf("Failed confiscate order : %s", err)
		return
	}

	if err := reconcileOrder(ctx, t, test); err != nil {
		t.Errorf("Failed reconcile order : %s", err)
		return
	}

	if err := assetAmendment(ctx, t, test); err != nil {
		t.Errorf("Failed asset amendment : %s", err)
		return
	}
}

func createContract(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.ContractOffer")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

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
		permissions[i].Permitted = false      // Issuer can update field without proposal
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
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	var offerInputHash chainhash.Hash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&offerInputHash, 0), make([]byte, 130)))

	// To contract
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = offerItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote itx")
	}

	test.cache.AddTX(ctx, offerTx)

	err = test.api.Trigger(ctx, protomux.SEE, offerItx)
	if err == nil {
		return errors.New("Accepted invalid contract offer")
	}
	t.Logf("Rejected invalid contract offer : %s", err)

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

	err = offerItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote itx")
	}

	test.cache.AddTX(ctx, offerTx)

	// Resubmit to handler
	err = test.api.Trigger(ctx, protomux.SEE, offerItx)
	if err != nil {
		return errors.Wrap(err, "Failed to handle contract offer")
	}

	t.Logf("Contract offer accepted")

	if err := checkResponse(ctx, t, test, "C2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	return nil
}

func createAsset(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.AssetDefinition")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, test.DB, contractPKH)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100001, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	test.assetCode = *protocol.AssetCodeFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})

	// Create AssetDefinition message
	assetData := protocol.AssetDefinition{
		AssetType:                  protocol.CodeShareCommon,
		AssetCode:                  test.assetCode,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
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
		permissions[i].Permitted = true       // Issuer can update field without proposal
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
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = assetItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote asset itx")
	}

	test.cache.AddTX(ctx, assetTx)

	err = test.api.Trigger(ctx, protomux.SEE, assetItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept asset definition")
	}

	t.Logf("Asset definition accepted")

	if err := checkResponse(ctx, t, test, "A2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	// Check issuer balance
	contractPKH = protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &assetData.AssetCode)
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

func transferTokens(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.Transfer")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100002, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     test.assetCode,
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
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To user
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(test.userKey.Address.ScriptAddress())))

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

	err = transferItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote transfer itx")
	}

	test.cache.AddTX(ctx, transferTx)

	err = test.api.Trigger(ctx, protomux.SEE, transferItx)
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

	err = transferItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote transfer itx")
	}

	// Resubmit
	test.cache.AddTX(ctx, transferTx)

	err = test.api.Trigger(ctx, protomux.SEE, transferItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept transfer")
	}

	t.Logf("Transfer accepted")

	if err := checkResponse(ctx, t, test, "T2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != test.tokenQty-transferAmount {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, test.tokenQty-transferAmount)
	}

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != transferAmount {
		return fmt.Errorf("User token balance incorrect : %d != %d", userBalance, transferAmount)
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
	t.Logf("User asset balance : %d", userBalance)
	return nil
}

func holderProposal(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.Proposal")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100003, txbuilder.P2PKHScriptForPKH(test.userKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	v := ctx.Value(node.KeyValues).(*node.Values)

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
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(51000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = proposalItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote proposal itx")
	}

	test.cache.AddTX(ctx, proposalTx)

	err = test.api.Trigger(ctx, protomux.SEE, proposalItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept proposal")
	}

	t.Logf("Proposal accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		test.voteTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, test, "G2"); err != nil {
		return errors.Wrap(err, "Failed to check proposal response")
	}

	return nil
}

func sendBallot(ctx context.Context, t *testing.T, test *Test, pkh *protocol.PublicKeyHash, vote string) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100006, txbuilder.P2PKHScriptForPKH(pkh.Bytes())))
	test.cache.AddTX(ctx, fundingTx)

	ballotData := protocol.BallotCast{
		VoteTxId: test.voteTxId,
		Vote:     vote,
	}

	// Build transaction
	ballotTx := wire.NewMsgTx(2)

	var ballotInputHash chainhash.Hash
	ballotInputHash = fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&ballotInputHash, 0), make([]byte, 130)))

	// To contract
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = ballotItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote ballot itx")
	}

	test.cache.AddTX(ctx, ballotTx)

	err = test.api.Trigger(ctx, protomux.SEE, ballotItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept ballot")
	}

	t.Logf("Ballot accepted")

	if err := checkResponse(ctx, t, test, "G4"); err != nil {
		return errors.Wrap(err, "Failed to check ballot counted result")
	}

	return nil
}

func processVoteResult(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		test.voteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, test, "G5"); err != nil {
		return errors.Wrap(err, "Failed to check vote result")
	}

	return nil
}

func proposalAmendment(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.proposalAmendment")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100007, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	amendmentData := protocol.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          test.voteResultTxId,
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
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = amendmentItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote amendment itx")
	}

	test.cache.AddTX(ctx, amendmentTx)

	err = test.api.Trigger(ctx, protomux.SEE, amendmentItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept amendment")
	}

	t.Logf("Amendment accepted")

	if err := checkResponse(ctx, t, test, "C2"); err != nil {
		return errors.Wrap(err, "Failed to check amendment response")
	}

	return nil
}

func freezeOrder(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.freezeOrder")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100008, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        test.assetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress()),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = orderItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	test.cache.AddTX(ctx, orderTx)

	err = test.api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Freeze order accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		test.freezeTxId = *protocol.TxIdFromBytes(hash[:])
	}

	if err := checkResponse(ctx, t, test, "E2"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	if asset.CheckBalanceFrozen(ctx, as, userPKH, 100, v.Now) {
		return errors.New("User unfrozen balance too high")
	}

	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 50, v.Now) {
		return errors.New("User unfrozen balance not high enough")
	}

	return nil
}

func thawOrder(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.freezeOrder")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionThaw,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        test.assetCode,
		FreezeTxId:       test.freezeTxId,
		Message:          "Court order released",
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = orderItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	test.cache.AddTX(ctx, orderTx)

	err = test.api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Thaw order accepted")

	if err := checkResponse(ctx, t, test, "E3"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 250, v.Now) {
		return errors.New("User balance not unfrozen")
	}

	return nil
}

func confiscateOrder(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.confiscateOrder")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionConfiscation,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        test.assetCode,
		DepositAddress:   *issuerPKH,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = orderItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	test.cache.AddTX(ctx, orderTx)

	err = test.api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Confiscate order accepted")

	if err := checkResponse(ctx, t, test, "E4"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != test.tokenQty-200 {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, test.tokenQty-200)
	}

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 200 {
		return fmt.Errorf("User token balance incorrect : %d != %d", userBalance, 200)
	}

	return nil
}

func reconcileOrder(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.reconcileOrder")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionReconciliation,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        test.assetCode,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(test.userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	orderData.BitcoinDispersions = append(orderData.BitcoinDispersions, protocol.QuantityIndex{Index: 0, Quantity: 75000})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(751500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = orderItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote order itx")
	}

	test.cache.AddTX(ctx, orderTx)

	err = test.api.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept order")
	}

	t.Logf("Reconcile order accepted")

	if len(responses) < 1 {
		return errors.New("No response for reconcile")
	}

	// Check for bitcoin dispersion to user
	found := false
	for _, output := range responses[0].TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}
		if bytes.Equal(pkh, userPKH.Bytes()) && output.Value == 75000 {
			t.Logf("Found reconcile bitcoin dispersion")
			found = true
		}
	}

	if !found {
		return errors.New("Failed to find bitcoin dispersion")
	}

	if err := checkResponse(ctx, t, test, "E5"); err != nil {
		return errors.Wrap(err, "Failed to check order response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != test.tokenQty-200 {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, test.tokenQty-200)
	}
	t.Logf("Verified issuer balance : %d", issuerBalance)

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 150 {
		return fmt.Errorf("User token balance incorrect : %d != %d", userBalance, 150)
	}
	t.Logf("Verified user balance : %d", userBalance)

	return nil
}

func assetAmendment(ctx context.Context, t *testing.T, test *Test) error {
	ctx, span := trace.StartSpan(ctx, "Test.assetAmendment")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100007, txbuilder.P2PKHScriptForPKH(test.issuerKey.Address.ScriptAddress())))
	test.cache.AddTX(ctx, fundingTx)

	amendmentData := protocol.AssetModification{
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     test.assetCode,
		AssetRevision: 0,
	}

	// Serialize new token quantity
	newQuantity := uint64(1200)
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, &newQuantity)
	if err != nil {
		return errors.Wrap(err, "Failed to serialize new quantity")
	}

	amendmentData.Amendments = append(amendmentData.Amendments, protocol.Amendment{
		FieldIndex: 11, // Token quantity
		Data:       buf.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	var amendmentInputHash chainhash.Hash
	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

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

	err = amendmentItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote amendment itx")
	}

	test.cache.AddTX(ctx, amendmentTx)

	err = test.api.Trigger(ctx, protomux.SEE, amendmentItx)
	if err != nil {
		return errors.Wrap(err, "Failed to accept amendment")
	}

	t.Logf("Amendment accepted")

	if err := checkResponse(ctx, t, test, "A2"); err != nil {
		return errors.Wrap(err, "Failed to check amendment response")
	}

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.DB, contractPKH, &test.assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	if as.TokenQty != 1200 {
		return fmt.Errorf("Asset token quantity incorrect : %d != %d", as.TokenQty, 1200)
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(test.issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != 1000 {
		return fmt.Errorf("Issuer token balance incorrect : %d != %d", issuerBalance, 1000)
	}
	t.Logf("Verified issuer balance : %d", issuerBalance)

	return nil
}

func checkResponse(ctx context.Context, t *testing.T, test *Test, responseCode string) error {
	ctx, span := trace.StartSpan(ctx, "Test.CheckResponse")
	defer span.End()

	ctx = test.Context(ctx, span.SpanContext().TraceID.String())

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

	err = responseItx.Promote(ctx, &test.cache)
	if err != nil {
		return errors.Wrap(err, "Failed to promote response itx")
	}

	test.cache.AddTX(ctx, response)

	err = test.api.Trigger(ctx, protomux.SEE, responseItx)
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
