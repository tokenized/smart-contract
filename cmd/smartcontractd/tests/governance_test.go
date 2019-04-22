package tests

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestGovernance is the entry point for testing governance functions.
func TestGovernance(t *testing.T) {
	defer tests.Recover(t)

	t.Run("holderProposal", holderProposal)
	t.Run("sendBallots", sendBallots)
	t.Run("processVoteResult", processVoteResult)
}

func holderProposal(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100003, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

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
		t.Fatalf("\t%s\tFailed to serialize proposal : %v", tests.Failed, err)
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create proposal itx : %v", tests.Failed, err)
	}

	err = proposalItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote proposal itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, proposalTx)

	err = a.Trigger(ctx, "SEE", proposalItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept proposal : %v", tests.Failed, err)
	}

	t.Logf("Proposal accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G2")
}

// sendBallots performs multiple tests against the sendBallot function
func sendBallots(t *testing.T) {
	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	sendBallot(t, issuerPKH, "A")

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	sendBallot(t, userPKH, "B")
}

func sendBallot(t *testing.T, pkh *protocol.PublicKeyHash, vote string) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100006, txbuilder.P2PKHScriptForPKH(pkh.Bytes())))
	test.RPCNode.AddTX(ctx, fundingTx)

	ballotData := protocol.BallotCast{
		VoteTxId: testVoteTxId,
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
		t.Fatalf("\t%s\tFailed to serialize ballot : %v", tests.Failed, err)
	}
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(0, script))

	ballotItx, err := inspector.NewTransactionFromWire(ctx, ballotTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create ballot itx : %v", tests.Failed, err)
	}

	err = ballotItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote ballot itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, ballotTx)

	err = a.Trigger(ctx, "SEE", ballotItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept ballot : %v", tests.Failed, err)
	}

	t.Logf("Ballot accepted")

	// Check the response
	checkResponse(t, "G4")
}

func processVoteResult(t *testing.T) {
	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G5")
}

func randomTxId() *protocol.TxId {
	data := make([]byte, 32)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i, _ := range data {
		data[i] = byte(r.Intn(256))
	}
	return protocol.TxIdFromBytes(data)
}

func mockUpAssetAmendmentVote(ctx context.Context, initiator, system uint8, amendment *protocol.Amendment) error {
	var voteData = state.Vote{
		Initiator:         initiator,
		VoteSystem:        system,
		AssetSpecificVote: true,
		AssetType:         testAssetType,
		AssetCode:         testAssetCode,
		Specific:          true,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		VoteTxId: *randomTxId(),
	}

	testVoteTxId = voteData.VoteTxId

	voteData.ProposedAmendments = append(voteData.ProposedAmendments, *amendment)

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	return vote.Save(ctx, test.MasterDB, contractPKH, &voteData)
}

func mockUpVoteResultTx(ctx context.Context, result string) error {
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		return err
	}

	vt.CompletedAt = protocol.CurrentTimestamp()
	vt.Result = result

	// Set result Id
	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100002, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	// Build result transaction
	resultTx := wire.NewMsgTx(2)

	var resultInputHash chainhash.Hash
	resultInputHash = fundingTx.TxHash()

	// From issuer
	resultTx.TxIn = append(resultTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&resultInputHash, 0), make([]byte, 130)))

	// To contract
	resultTx.TxOut = append(resultTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	resultData := protocol.Result{
		AssetSpecificVote:  vt.AssetSpecificVote,
		AssetType:          vt.AssetType,
		AssetCode:          vt.AssetCode,
		Specific:           vt.Specific,
		ProposedAmendments: vt.ProposedAmendments,
		VoteTxId:           testVoteTxId,
		OptionTally:        []uint64{1000, 0},
		Result:             "A",
		Timestamp:          protocol.CurrentTimestamp(),
	}
	script, err := protocol.Serialize(&resultData)
	if err != nil {
		return err
	}
	resultTx.TxOut = append(resultTx.TxOut, wire.NewTxOut(0, script))

	resultItx, err := inspector.NewTransactionFromWire(ctx, resultTx)
	if err != nil {
		return err
	}

	err = resultItx.Promote(ctx, test.RPCNode)
	if err != nil {
		return err
	}

	testVoteResultTxId = *protocol.TxIdFromBytes(resultItx.Hash[:])

	if err := transactions.AddTx(ctx, test.MasterDB, resultItx); err != nil {
		return err
	}

	return vote.Save(ctx, test.MasterDB, contractPKH, vt)
}
