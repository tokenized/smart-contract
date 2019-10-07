package tests

import (
	"context"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestGovernance is the entry point for testing governance functions.
func TestGovernance(t *testing.T) {
	defer tests.Recover(t)

	t.Run("proposal", holderProposal)
	t.Run("ballot", sendBallot)
	t.Run("adminBallot", adminBallot)
	t.Run("result", voteResult)
	t.Run("relativeResult", voteResultRelative)
	t.Run("absoluteResult", voteResultAbsolute)
}

func holderProposal(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address, 150)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address)

	v := ctx.Value(node.KeyValues).(*node.Values)

	proposalData := actions.Proposal{
		Type:                1,
		VoteSystem:          0,
		VoteOptions:         "AB",
		VoteMax:             1,
		ProposalDescription: "Change contract name",
		VoteCutOffTimestamp: v.Now.Nano() + 10000000000,
	}

	fip := actions.FieldIndexPath{actions.ContractFieldContractName}
	fipBytes, _ := fip.Bytes()
	proposalData.ProposedAmendments = append(proposalData.ProposedAmendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test Name 2"),
	})

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	script, _ := test.ContractKey.Address.LockingScript()
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, script))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
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
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if vt.Type != proposalData.Type {
		t.Fatalf("\t%s\tType incorrect : %d != %d", tests.Failed, vt.Type, proposalData.Type)
	}

	t.Logf("\t%s\tVerified initiator : %d", tests.Success, vt.Type)

	if vt.VoteSystem != proposalData.VoteSystem {
		t.Fatalf("\t%s\tVote system incorrect : %d != %d", tests.Failed, vt.VoteSystem, proposalData.VoteSystem)
	}

	t.Logf("\t%s\tVerified vote system : %d", tests.Success, vt.VoteSystem)

	if vt.Expires.Nano() != proposalData.VoteCutOffTimestamp {
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
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address, 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpProposal(ctx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up proposal : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100010, userKey.Address)

	ballotData := actions.BallotCast{
		VoteTxId: testVoteTxId.Bytes(),
		Vote:     "A",
	}

	// Build transaction
	ballotTx := wire.NewMsgTx(2)

	ballotInputHash := fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(ballotInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&ballotData, test.NodeConfig.IsTest)
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
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if !vt.Ballots[0].Address.Equal(userKey.Address) {
		t.Fatalf("\t%s\tFailed to verify ballot pkh : %x != %x", tests.Failed,
			vt.Ballots[0].Address.Bytes(), userKey.Address.Bytes())
	}
	t.Logf("\t%s\tVerified ballot address : %x", tests.Success, userKey.Address.Bytes())

	if vt.Ballots[0].Quantity != 250 {
		t.Fatalf("\t%s\tFailed to verify ballot quantity : %d != %d", tests.Failed, vt.Ballots[0].Quantity, 250)
	}
	t.Logf("\t%s\tVerified ballot quantity : %d", tests.Success, vt.Ballots[0].Quantity)
}

// adminBallot tests ballots in an administrativ vote
func adminBallot(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 10, 0, &sampleAdminAssetPayload, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 1, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpAssetHolding(ctx, userKey.Address, testAssetCodes[0], 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpAssetHolding(ctx, user2Key.Address, testAssetCodes[1], 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpProposalType(ctx, 2, &testAssetCodes[0]) // Administrative
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up proposal : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100010, userKey.Address)

	ballotData := actions.BallotCast{
		VoteTxId: testVoteTxId.Bytes(),
		Vote:     "A",
	}

	// Build transaction
	ballotTx := wire.NewMsgTx(2)

	ballotInputHash := fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(ballotInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&ballotData, test.NodeConfig.IsTest)
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

	t.Logf("Ballot tx %s", ballotItx.Hash.String())
	err = a.Trigger(ctx, "SEE", ballotItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept ballot : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tBallot accepted", tests.Success)

	// Check the response
	checkResponse(t, "G4")

	// Verify ballot counted
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve vote : %v", tests.Failed, err)
	}

	if !vt.Ballots[0].Address.Equal(userKey.Address) {
		t.Fatalf("\t%s\tFailed to verify ballot pkh : %x != %x", tests.Failed,
			vt.Ballots[0].Address.Bytes(), userKey.Address.Bytes())
	}
	t.Logf("\t%s\tVerified ballot address : %x", tests.Success, userKey.Address.Bytes())

	if vt.Ballots[0].Quantity != 1 {
		t.Fatalf("\t%s\tFailed to verify ballot quantity : %d != %d", tests.Failed, vt.Ballots[0].Quantity, 1)
	}
	t.Logf("\t%s\tVerified ballot quantity : %d", tests.Success, vt.Ballots[0].Quantity)

	/*********************************** Vote from someone without admin asset ********************/
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100010, user2Key.Address)

	// Build transaction
	ballotTx = wire.NewMsgTx(2)

	ballotInputHash = fundingTx.TxHash()

	// From pkh
	ballotTx.TxIn = append(ballotTx.TxIn, wire.NewTxIn(wire.NewOutPoint(ballotInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&ballotData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize ballot : %v", tests.Failed, err)
	}
	ballotTx.TxOut = append(ballotTx.TxOut, wire.NewTxOut(0, script))

	ballotItx, err = inspector.NewTransactionFromWire(ctx, ballotTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create ballot itx : %v", tests.Failed, err)
	}

	err = ballotItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote ballot itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, ballotTx)

	t.Logf("Invalid ballot tx %s", ballotItx.Hash.String())
	err = a.Trigger(ctx, "SEE", ballotItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject invalid ballot", tests.Failed)
	}

	t.Logf("\t%s\tInvalid ballot rejected", tests.Success)

	// Check the response
	checkResponse(t, "M2")
}

func voteResult(t *testing.T) {
	ctx := test.Context

	// Mock up vote with expiration in half a second
	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address, 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	// Wait for vote expiration
	time.Sleep(2 * time.Second)

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testVoteResultTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "G5")

	// Verify result
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
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
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address, 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpBallot(ctx, userKey.Address, 250, "A")
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
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
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
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address, 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	err = mockUpVote(ctx, 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpBallot(ctx, userKey.Address, 250, "A")
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
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
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

func mockUpBallot(ctx context.Context, address bitcoin.RawAddress, quantity uint64, v string) error {
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
	if err != nil {
		return err
	}

	vt.Ballots = append(vt.Ballots, &state.Ballot{
		Address:   address,
		Vote:      v,
		Quantity:  quantity,
		Timestamp: protocol.CurrentTimestamp(),
	})

	return vote.Save(ctx, test.MasterDB, test.ContractKey.Address, vt)
}

func mockUpVote(ctx context.Context, voteSystem uint32) error {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address)

	v := ctx.Value(node.KeyValues).(*node.Values)

	proposalData := actions.Proposal{
		Type:                1,
		VoteSystem:          voteSystem,
		VoteOptions:         "AB",
		VoteMax:             1,
		VoteCutOffTimestamp: v.Now.Nano() + 500000000,
	}

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	script, _ := test.ContractKey.Address.LockingScript()
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, script))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
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

	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 1000014, test.ContractKey.Address)

	ts := protocol.CurrentTimestamp()
	voteActionData := actions.Vote{
		Timestamp: ts.Nano(),
	}

	// Build proposal transaction
	voteTx := wire.NewMsgTx(2)

	voteInputHash := proposalTx.TxHash()

	// From user
	voteTx.TxIn = append(voteTx.TxIn, wire.NewTxIn(wire.NewOutPoint(voteInputHash, 1), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	voteTx.TxOut = append(voteTx.TxOut, wire.NewTxOut(2000, script))

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
	return mockUpProposalType(ctx, 1, nil) // Administrator
}

func mockUpProposalType(ctx context.Context, proposalType uint32, assetCode *protocol.AssetCode) error {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100009, userKey.Address)

	v := ctx.Value(node.KeyValues).(*node.Values)

	proposalData := actions.Proposal{
		Type:                proposalType,
		VoteSystem:          0,
		VoteOptions:         "AB",
		VoteMax:             1,
		ProposalDescription: "Change contract name",
		VoteCutOffTimestamp: v.Now.Nano() + 500000000,
	}

	if assetCode != nil {
		proposalData.AssetCode = assetCode.Bytes()
	}

	fip := actions.FieldIndexPath{actions.ContractFieldContractName}
	fipBytes, _ := fip.Bytes()
	proposalData.ProposedAmendments = append(proposalData.ProposedAmendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test Name 2"),
	})

	// Build proposal transaction
	proposalTx := wire.NewMsgTx(2)

	proposalInputHash := fundingTx.TxHash()

	// From user
	proposalTx.TxIn = append(proposalTx.TxIn, wire.NewTxIn(wire.NewOutPoint(proposalInputHash, 0), make([]byte, 130)))

	// To contract (for vote response)
	script, _ := test.ContractKey.Address.LockingScript()
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(52000, script))

	// To contract (second output to fund result)
	proposalTx.TxOut = append(proposalTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&proposalData, test.NodeConfig.IsTest)
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
		Type:       proposalType,
		VoteSystem: 0,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		ProposalTxId: protocol.TxIdFromBytes(proposalItx.Hash[:]),
		VoteTxId:     &testVoteTxId,
		Expires:      protocol.NewTimestamp(now.Nano() + 500000000),
	}

	return vote.Save(ctx, test.MasterDB, test.ContractKey.Address, &voteData)
}

func mockUpAssetAmendmentVote(ctx context.Context, voteType, system uint32, amendment *actions.AmendmentField) error {
	now := protocol.CurrentTimestamp()
	var voteData = state.Vote{
		Type:       voteType,
		VoteSystem: system,
		AssetType:  testAssetType,
		AssetCode:  &testAssetCodes[0],

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		VoteTxId: tests.RandomTxId(),
		Expires:  protocol.NewTimestamp(now.Nano() + 5000000000),
	}

	testVoteTxId = *voteData.VoteTxId

	voteData.ProposedAmendments = append(voteData.ProposedAmendments, amendment)

	return vote.Save(ctx, test.MasterDB, test.ContractKey.Address, &voteData)
}

func mockUpContractAmendmentVote(ctx context.Context, voteType, system uint32,
	amendment *actions.AmendmentField) error {
	now := protocol.CurrentTimestamp()
	var voteData = state.Vote{
		Type:       voteType,
		VoteSystem: system,

		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),

		VoteTxId: tests.RandomTxId(),
		Expires:  protocol.NewTimestamp(now.Nano() + 5000000000),
	}

	testVoteTxId = *voteData.VoteTxId

	voteData.ProposedAmendments = append(voteData.ProposedAmendments, amendment)

	return vote.Save(ctx, test.MasterDB, test.ContractKey.Address, &voteData)
}

func mockUpVoteResultTx(ctx context.Context, result string) error {
	vt, err := vote.Fetch(ctx, test.MasterDB, test.ContractKey.Address, &testVoteTxId)
	if err != nil {
		return err
	}

	vt.CompletedAt = protocol.CurrentTimestamp()
	vt.Result = result

	// Set result Id
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100011, issuerKey.Address)

	// Build result transaction
	resultTx := wire.NewMsgTx(2)

	resultInputHash := fundingTx.TxHash()

	// From issuer
	resultTx.TxIn = append(resultTx.TxIn, wire.NewTxIn(wire.NewOutPoint(resultInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	resultTx.TxOut = append(resultTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	ts := protocol.CurrentTimestamp()
	resultData := actions.Result{
		AssetType:          vt.AssetType,
		AssetCode:          vt.AssetCode.Bytes(),
		ProposedAmendments: vt.ProposedAmendments,
		VoteTxId:           testVoteTxId.Bytes(),
		OptionTally:        []uint64{1000, 0},
		Result:             "A",
		Timestamp:          ts.Nano(),
	}
	script, err = protocol.Serialize(&resultData, test.NodeConfig.IsTest)
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

	return vote.Save(ctx, test.MasterDB, test.ContractKey.Address, vt)
}
