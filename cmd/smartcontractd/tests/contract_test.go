package tests

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestContracts is the entry point for testing contract based functions.
func TestContracts(t *testing.T) {
	defer tests.Recover(t)

	t.Run("createContract", createContract)
}

func createContract(t *testing.T) {
	ctx := test.Context

	// New Contract Offer
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
		t.Fatalf("\t%s\tFailed to serialize contract auth flags : %v", tests.Failed, err)
	}

	// Create funding tx
	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

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
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
	}

	err = offerItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, offerTx)

	err = a.Trigger(ctx, "SEE", offerItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted invalid contract offer", tests.Failed)
	}
	t.Logf("Rejected invalid contract offer : %s", err)

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Fatalf("\t%s\tHandle contract offer created no reject response", tests.Failed)
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
		t.Fatalf("\t%s\tContract offer response doesn't contain tokenized op return", tests.Failed)
	}
	if responseMsg.Type() != "M2" {
		t.Fatalf("\t%s\tContract offer response not a reject : %s", tests.Failed, responseMsg.Type())
	}
	reject, ok := responseMsg.(*protocol.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != protocol.RejectMsgMalformed {
		t.Fatalf("\t%s\tWrong reject code for contract offer reject", tests.Failed)
	}

	t.Logf("Invalid Contract offer rejection : (%d) %s", reject.RejectionCode, reject.Message)

	// ********************************************************************************************
	// Correct Contract Offer
	offerData.BodyOfAgreementType = 2

	// Reserialize and update tx
	script, err = protocol.Serialize(&offerData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut[1].PkScript = script

	offerItx, err = inspector.NewTransactionFromWire(ctx, offerTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
	}

	err = offerItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, offerTx)

	// Resubmit to handler
	err = a.Trigger(ctx, "SEE", offerItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to handle contract offer : %v", tests.Failed, err)
	}

	t.Logf("Contract offer accepted")

	// Check the response
	checkResponse(t, "C2")
}
