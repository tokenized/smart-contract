package tests

import (
	"context"
	"testing"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestContracts is the entry point for testing contract based functions.
func TestContracts(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createContract)
	t.Run("amendment", contractAmendment)
	t.Run("proposalAmendment", contractProposalAmendment)
}

func createContract(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	// New Contract Offer
	offerData := protocol.ContractOffer{
		ContractName:        "Test Name",
		BodyOfAgreementType: 0,
		BodyOfAgreement:     []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: protocol.Entity{
			Type:           'I',
			Administration: []protocol.Administrator{protocol.Administrator{Type: 1, Name: "John Smith"}},
		},
		VotingSystems:  []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000}},
		HolderProposal: true,
	}

	// Define permissions for contract fields
	permissions := make([]protocol.Permission, 20)
	for i, _ := range permissions {
		permissions[i].Permitted = false              // Issuer can update field without proposal
		permissions[i].AdministrationProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false         // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	var err error
	offerData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize contract auth flags : %v", tests.Failed, err)
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address.ScriptAddress())

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&offerInputHash, 0), make([]byte, 130)))

	// To contract
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(750000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	offerItx, err := inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
	}

	err = offerItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
	}

	err = offerItx.Validate(ctx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, offerTx)

	err = a.Trigger(ctx, "SEE", offerItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted invalid contract offer", tests.Failed)
	}
	t.Logf("\t%s\tRejected invalid contract offer : %s", tests.Success, err)

	// ********************************************************************************************
	// Check reject response
	if len(responses) != 1 {
		t.Fatalf("\t%s\tHandle contract offer created no reject response", tests.Failed)
	}

	var responseMsg protocol.OpReturnMessage
	response := responses[0].Copy()
	responses = nil
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
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

	t.Logf("\t%s\tInvalid Contract offer rejection : (%d) %s", tests.Success, reject.RejectionCode,
		reject.Message)

	// ********************************************************************************************
	// Correct Contract Offer
	offerData.BodyOfAgreementType = 2

	// Reserialize and update tx
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut[1].PkScript = script

	offerItx, err = inspector.NewTransactionFromWire(ctx, offerTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create itx : %v", tests.Failed, err)
	}

	err = offerItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote itx : %v", tests.Failed, err)
	}

	err = offerItx.Validate(ctx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to validate itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, offerTx)

	// Resubmit to handler
	err = a.Trigger(ctx, "SEE", offerItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to handle contract offer : %v", tests.Failed, err)
	}

	t.Logf("Contract offer accepted")

	// Check the response
	checkResponse(t, "C2")

	// Verify data
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, test.MasterDB, contractPKH)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractName != offerData.ContractName {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractName, offerData.ContractName)
	}

	t.Logf("\t%s\tVerified contract name : %s", tests.Success, ct.ContractName)

	if string(ct.BodyOfAgreement) != string(offerData.BodyOfAgreement) {
		t.Fatalf("\t%s\tContract body incorrect : \"%s\" != \"%s\"", tests.Failed, string(ct.BodyOfAgreement), string(offerData.BodyOfAgreement))
	}

	t.Logf("\t%s\tVerified body name : %s", tests.Success, ct.BodyOfAgreement)

	if ct.Issuer.Administration[0].Name != offerData.Issuer.Administration[0].Name {
		t.Fatalf("\t%s\tContract issuer name incorrect : \"%s\" != \"%s\"", tests.Failed, ct.Issuer.Administration[0].Name, "John Smith")
	}

	t.Logf("\t%s\tVerified issuer name : %s", tests.Success, ct.Issuer.Administration[0].Name)

	if !ct.HolderProposal {
		t.Fatalf("\t%s\tContract holder proposal incorrect : %t", tests.Failed, ct.HolderProposal)
	}

	t.Logf("\t%s\tVerified holder proposal", tests.Success)

	if ct.AdministrationProposal {
		t.Fatalf("\t%s\tContract issuer proposal incorrect : %t", tests.Failed, ct.AdministrationProposal)
	}

	t.Logf("\t%s\tVerified issuer proposal", tests.Success)
}

func contractAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100015, issuerKey.Address.ScriptAddress())

	amendmentData := protocol.ContractAmendment{
		ContractRevision: 0,
	}

	amendmentData.Amendments = append(amendmentData.Amendments, protocol.Amendment{
		FieldIndex: 0, // ContractName
		Data:       []byte("Test Contract 2"),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize contract amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create contract amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote contract amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, amendmentTx)

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept contract amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tContract Amendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "C2")

	// Check contract name
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, test.MasterDB, contractPKH)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractName != "Test Contract 2" {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractName, "Test Contract 2")
	}

	t.Logf("\t%s\tVerified contract name : %s", tests.Success, ct.ContractName)
}

func contractProposalAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, true, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	assetAmendment := protocol.Amendment{
		FieldIndex: 3, // ContractType
		Data:       []byte("New Type"),
	}
	err = mockUpContractAmendmentVote(ctx, 0, 0, &assetAmendment)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpVoteResultTx(ctx, "A")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote result : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000015, issuerKey.Address.ScriptAddress())

	amendmentData := protocol.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          testVoteResultTxId,
	}

	amendmentData.Amendments = append(amendmentData.Amendments, assetAmendment)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize contract amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create contract amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote contract amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, amendmentTx)

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept contract amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tContract Amendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "C2")

	// Check contract type
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, test.MasterDB, contractPKH)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractType != "New Type" {
		t.Fatalf("\t%s\tContract type incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractType, "New Type")
	}

	t.Logf("\t%s\tVerified contract type : %s", tests.Success, ct.ContractType)
}

func mockUpContract(ctx context.Context, name, agreement string, issuerType byte, issuerRole uint8, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) error {

	var contractData = state.Contract{
		ID:                  *protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress()),
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: protocol.Entity{
			Type:           issuerType,
			Administration: []protocol.Administrator{protocol.Administrator{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000},
			protocol.VotingSystem{Name: "Absolute 75", VoteType: 'A', ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		CreatedAt:         protocol.CurrentTimestamp(),
		UpdatedAt:         protocol.CurrentTimestamp(),
		AdministrationPKH: *protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress()),
		MasterPKH:         *protocol.PublicKeyHashFromBytes(test.MasterKey.Address.ScriptAddress()),
	}

	// Define permissions for asset fields
	permissions := make([]protocol.Permission, 20)
	for i, _ := range permissions {
		permissions[i].Permitted = permitted           // Issuer can update field without proposal
		permissions[i].AdministrationProposal = issuer // Issuer can update field with a proposal
		permissions[i].HolderProposal = holder         // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	var err error
	contractData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContract2(ctx context.Context, name, agreement string, issuerType byte, issuerRole uint8, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) error {

	var contractData = state.Contract{
		ID:                  *protocol.PublicKeyHashFromBytes(test.Contract2Key.Address.ScriptAddress()),
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: protocol.Entity{
			Type:           issuerType,
			Administration: []protocol.Administrator{protocol.Administrator{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000},
			protocol.VotingSystem{Name: "Absolute 75", VoteType: 'A', ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		CreatedAt:         protocol.CurrentTimestamp(),
		UpdatedAt:         protocol.CurrentTimestamp(),
		AdministrationPKH: *protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress()),
		MasterPKH:         *protocol.PublicKeyHashFromBytes(test.Master2Key.Address.ScriptAddress()),
	}

	// Define permissions for asset fields
	permissions := make([]protocol.Permission, 20)
	for i, _ := range permissions {
		permissions[i].Permitted = permitted           // Issuer can update field without proposal
		permissions[i].AdministrationProposal = issuer // Issuer can update field with a proposal
		permissions[i].HolderProposal = holder         // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	var err error
	contractData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContractWithOracle(ctx context.Context, name, agreement string, issuerType byte, issuerRole uint8, issuerName string) error {
	var contractData = state.Contract{
		ID:                  *protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress()),
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: protocol.Entity{
			Type:           issuerType,
			Administration: []protocol.Administrator{protocol.Administrator{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []protocol.VotingSystem{protocol.VotingSystem{Name: "Relative 50", VoteType: 'R', ThresholdPercentage: 50, HolderProposalFee: 50000},
			protocol.VotingSystem{Name: "Absolute 75", VoteType: 'A', ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		CreatedAt:         protocol.CurrentTimestamp(),
		UpdatedAt:         protocol.CurrentTimestamp(),
		AdministrationPKH: *protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress()),
		MasterPKH:         *protocol.PublicKeyHashFromBytes(test.MasterKey.Address.ScriptAddress()),
		Oracles:           []protocol.Oracle{protocol.Oracle{Name: "KYC, Inc.", URL: "bsv.kyc.com", PublicKey: oracleKey.PublicKey.SerializeCompressed()}},
	}

	// Define permissions for asset fields
	permissions := make([]protocol.Permission, 20)
	for i, _ := range permissions {
		permissions[i].Permitted = false              // Issuer can update field without proposal
		permissions[i].AdministrationProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false         // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	var err error
	contractData.ContractAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		return err
	}

	if err := contract.ExpandOracles(ctx, &contractData); err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}
