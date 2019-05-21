package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// TestGovernance is the entry point for testing governance functions.
func TestGovernance(t *testing.T) {
	defer tests.Recover(t)

	t.Run("proposal", holderProposal)
	t.Run("ballot", sendBallot)
	t.Run("result", voteResult)
	t.Run("relativeResult", voteResultRelative)
	t.Run("absoluteResult", voteResultAbsolute)
}

func holderProposal(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 150)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address.ScriptAddress())

	v := ctx.Value(node.KeyValues).(*node.Values)

	proposalData := protocol.Proposal{
		Initiator:           1,
		AssetSpecificVote:   false,
		VoteSystem:          0,
		Specific:            true,
		VoteOptions:         "AB",
		VoteMax:             1,
		ProposalDescription: "Change contract name",
		VoteCutOffTimestamp: protocol.NewTimestamp(v.Now.Nano() + 10000000000),
	}

	proposalData.ProposedAmendments = append(proposalData.ProposedAmendments, protocol.Amendment{
		FieldIndex: 0,
		Data:       []byte("Test Name 2"),
	})

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize proposal : %v", tests.Failed, err)
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create proposal itx : %v", tests.Failed, err)
	}

	err = proposalItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote proposal itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, proposalTx)

	err = a.Trigger(ctx, "SEE", proposalItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept proposal : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tProposal accepted", tests.Success)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G2")

	// Verify vote
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if vt.Initiator != proposalData.Initiator {
		t.Fatalf("\t%s\tInitiator incorrect : %d != %d", tests.Failed, vt.Initiator, proposalData.Initiator)
	}

	t.Logf("\t%s\tVerified initiator : %d", tests.Success, vt.Initiator)

	if vt.VoteSystem != proposalData.VoteSystem {
		t.Fatalf("\t%s\tVote system incorrect : %d != %d", tests.Failed, vt.VoteSystem, proposalData.VoteSystem)
	}

	t.Logf("\t%s\tVerified vote system : %d", tests.Success, vt.VoteSystem)

	if vt.Expires != proposalData.VoteCutOffTimestamp {
		t.Fatalf("\t%s\tCut-off incorrect : %d != %d", tests.Failed, vt.Expires, proposalData.VoteCutOffTimestamp)
	}

	t.Logf("\t%s\tVerified cut-off : %s", tests.Success, vt.Expires.String())
}

// sendBallot sends a ballot tx to the contract
func sendBallot(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpProposal(ctx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up proposal : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100010, userKey.Address.ScriptAddress())

	ballotData := protocol.BallotCast{
		VoteTxId: testVoteTxId,
		Vote:     "A",
	}

	// Build transaction
	ballotTx := wire.NewMsgTx(2)

	ballotInputHash := fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&ballotInputHash, 0), make([]byte, 130)))

	// To contract
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&ballotData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize ballot : %v", tests.Failed, err)
	}
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(0, script))

	ballotItx, err := inspector.NewTransactionFromWire(ctx, ballotTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create ballot itx : %v", tests.Failed, err)
	}

	err = ballotItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote ballot itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, ballotTx)

	err = a.Trigger(ctx, "SEE", ballotItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept ballot : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tBallot accepted", tests.Success)

	// Check the response
	checkResponse(t, "G4")

	// Verify ballot counted
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if !bytes.Equal(vt.Ballots[0].PKH.Bytes(), userKey.Address.ScriptAddress()) {
		t.Fatalf("\t%s\tFailed to verify ballot pkh : %x != %x", tests.Failed, vt.Ballots[0].PKH.Bytes(), userKey.Address.ScriptAddress())
	}
	t.Logf("\t%s\tVerified ballot pkh : %x", tests.Success, userKey.Address.ScriptAddress())

	if vt.Ballots[0].Quantity != 250 {
		t.Fatalf("\t%s\tFailed to verify ballot quantity : %d != %d", tests.Failed, vt.Ballots[0].Quantity, 250)
	}
	t.Logf("\t%s\tVerified ballot quantity : %d", tests.Success, vt.Ballots[0].Quantity)
}

func voteResult(t *testing.T) {
	ctx := test.Context

	// Mock up vote with expiration in half a second
	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G5")

	// Verify result
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if vt.CompletedAt.Nano() == 0 {
		t.Fatalf("\t%s\tVote not completed", tests.Failed)
	}

	t.Logf("\t%s\tVerified completed : %s", tests.Success, vt.CompletedAt.String())

	if vt.OptionTally[0] != uint64(0) {
		t.Fatalf("\t%s\tVote option tally 0 incorrect : %d != 0", tests.Failed, vt.OptionTally[0])
	}

	t.Logf("\t%s\tVerified option tally 0 : %d", tests.Success, vt.OptionTally[0])

	if vt.OptionTally[1] != uint64(0) {
		t.Fatalf("\t%s\tVote option tally 1 incorrect : %d != 0", tests.Failed, vt.OptionTally[1])
	}

	t.Logf("\t%s\tVerified option tally 1 : %d", tests.Success, vt.OptionTally[1])

	if len(vt.Result) > 0 {
		t.Fatalf("\t%s\tVote result incorrect : \"%s\" != \"\"", tests.Failed, vt.Result)
	}

	t.Logf("\t%s\tVerified result : \"%s\"", tests.Success, vt.Result)
}

func voteResultRelative(t *testing.T) {
	ctx := test.Context

	// Mock up vote with expiration in half a second
	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpBallot(ctx, userKey.Address.ScriptAddress(), 250, "A")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up ballot : %v", tests.Failed, err)
	}

	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G5")

	// Verify result
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if vt.CompletedAt.Nano() == 0 {
		t.Fatalf("\t%s\tVote not completed", tests.Failed)
	}

	t.Logf("\t%s\tVerified completed : %s", tests.Success, vt.CompletedAt.String())

	if vt.OptionTally[0] != uint64(250) {
		t.Fatalf("\t%s\tVote option tally 0 incorrect : %d != 0", tests.Failed, vt.OptionTally[0])
	}

	t.Logf("\t%s\tVerified option tally 0 : %d", tests.Success, vt.OptionTally[0])

	if vt.OptionTally[1] != uint64(0) {
		t.Fatalf("\t%s\tVote option tally 1 incorrect : %d != 0", tests.Failed, vt.OptionTally[1])
	}

	t.Logf("\t%s\tVerified option tally 1 : %d", tests.Success, vt.OptionTally[1])

	if vt.Result != "A" {
		t.Fatalf("\t%s\tVote result incorrect : \"%s\" != \"A\"", tests.Failed, vt.Result)
	}

	t.Logf("\t%s\tVerified result : \"%s\"", tests.Success, vt.Result)
}

func voteResultAbsolute(t *testing.T) {
	ctx := test.Context

	// Mock up vote with expiration in half a second
	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpBallot(ctx, userKey.Address.ScriptAddress(), 250, "A")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up ballot : %v", tests.Failed, err)
	}

	// Wait for vote expiration
	time.Sleep(time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G5")

	// Verify result
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if vt.CompletedAt.Nano() == 0 {
		t.Fatalf("\t%s\tVote not completed", tests.Failed)
	}

	t.Logf("\t%s\tVerified completed : %s", tests.Success, vt.CompletedAt.String())

	if vt.OptionTally[0] != uint64(250) {
		t.Fatalf("\t%s\tVote option tally 0 incorrect : %d != 0", tests.Failed, vt.OptionTally[0])
	}

	t.Logf("\t%s\tVerified option tally 0 : %d", tests.Success, vt.OptionTally[0])

	if vt.OptionTally[1] != uint64(0) {
		t.Fatalf("\t%s\tVote option tally 1 incorrect : %d != 0", tests.Failed, vt.OptionTally[1])
	}

	t.Logf("\t%s\tVerified option tally 1 : %d", tests.Success, vt.OptionTally[1])

	if len(vt.Result) > 0 {
		t.Fatalf("\t%s\tVote result incorrect : \"%s\" != \"\"", tests.Failed, vt.Result)
	}

	t.Logf("\t%s\tVerified result : \"%s\"", tests.Success, vt.Result)
}

func mockUpBallot(ctx context.Context, pkh []byte, quantity uint64, v string) error {
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	vt, err := vote.Fetch(ctx, test.MasterDB, contractPKH, &testVoteTxId)
	if err != nil {
		return err
	}

	vt.Ballots = append(vt.Ballots, &state.Ballot{
		PKH:       *protocol.PublicKeyHashFromBytes(pkh),
		Vote:      v,
		Quantity:  quantity,
		Timestamp: protocol.CurrentTimestamp(),
	})

	return vote.Save(ctx, test.MasterDB, contractPKH, vt)
}

func mockUpVote(ctx context.Context, voteSystem uint8) error {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address.ScriptAddress())

	v := ctx.Value(node.KeyValues).(*node.Values)

	proposalData := protocol.Proposal{
		Initiator:           1,
		AssetSpecificVote:   false,
		VoteSystem:          voteSystem,
		Specific:            true,
		VoteOptions:         "AB",
		VoteMax:             1,
		VoteCutOffTimestamp: protocol.NewTimestamp(v.Now.Nano() + 500000000),
	}

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(3000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}

	err = proposalItx.Promote(ctx, test.RPCNode)
	if err != nil {
		return err
	}

	test.RPCNode.SaveTX(ctx, proposalTx)
	transactions.AddTx(ctx, test.MasterDB, proposalItx)

	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 1000014, test.ContractKey.Address.ScriptAddress())

	voteActionData := protocol.Vote{
		Timestamp: protocol.CurrentTimestamp(),
	}

	// Build proposal transaction
	voteTx := wire.NewMsgTx(2)

	voteInputHash := proposalTx.TxHash()

	// From user
	voteTx.TxIn = append(voteTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&voteInputHash, 1), make([]byte, 130)))

	// To contract
	voteTx.TxOut = append(voteTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err = protocol.Serialize(&voteActionData, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}
	voteTx.TxOut = append(voteTx.TxOut, wire.NewTxOut(0, script))

	voteItx, err := inspector.NewTransactionFromWire(ctx, voteTx, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}

	err = voteItx.Promote(ctx, test.RPCNode)
	if err != nil {
		return err
	}

	testVoteTxId = *protocol.TxIdFromBytes(voteItx.Hash[:])

	test.RPCNode.SaveTX(ctx, voteTx)

	err = a.Trigger(ctx, "SEE", voteItx)
	if err != nil {
		return err
	}

	return nil
}

func mockUpProposal(ctx context.Context) error {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address.ScriptAddress())

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

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(0, script))

	proposalItx, err := inspector.NewTransactionFromWire(ctx, proposalTx, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}

	err = proposalItx.Promote(ctx, test.RPCNode)
	if err != nil {
		return err
	}

	test.RPCNode.SaveTX(ctx, proposalTx)
	transactions.AddTx(ctx, test.MasterDB, proposalItx)

	now := protocol.CurrentTimestamp()
	testVoteTxId = *tests.RandomTxId()

	var voteData = state.Vote{
		Initiator:         1,
		VoteSystem:        0,
		AssetSpecificVote: false,
		Specific:          false,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		ProposalTxId: *protocol.TxIdFromBytes(proposalItx.Hash[:]),
		VoteTxId:     testVoteTxId,
		Expires:      protocol.NewTimestamp(now.Nano() + 500000000),
	}

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	return vote.Save(ctx, test.MasterDB, contractPKH, &voteData)
}

func mockUpAssetAmendmentVote(ctx context.Context, initiator, system uint8, amendment *protocol.Amendment) error {
	now := protocol.CurrentTimestamp()
	var voteData = state.Vote{
		Initiator:         initiator,
		VoteSystem:        system,
		AssetSpecificVote: true,
		AssetType:         testAssetType,
		AssetCode:         testAssetCode,
		Specific:          true,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		VoteTxId: *tests.RandomTxId(),
		Expires:  protocol.NewTimestamp(now.Nano() + 5000000000),
	}

	testVoteTxId = voteData.VoteTxId

	voteData.ProposedAmendments = append(voteData.ProposedAmendments, *amendment)

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	return vote.Save(ctx, test.MasterDB, contractPKH, &voteData)
}

func mockUpContractAmendmentVote(ctx context.Context, initiator, system uint8, amendment *protocol.Amendment) error {
	now := protocol.CurrentTimestamp()
	var voteData = state.Vote{
		Initiator:         initiator,
		VoteSystem:        system,
		AssetSpecificVote: false,
		Specific:          true,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		VoteTxId: *tests.RandomTxId(),
		Expires:  protocol.NewTimestamp(now.Nano() + 5000000000),
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
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100011, issuerKey.Address.ScriptAddress())

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
	script, err := protocol.Serialize(&resultData, test.NodeConfig.IsTest)
	if err != nil {
		return err
	}
	resultTx.TxOut = append(resultTx.TxOut, wire.NewTxOut(0, script))

	resultItx, err := inspector.NewTransactionFromWire(ctx, resultTx, test.NodeConfig.IsTest)
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
