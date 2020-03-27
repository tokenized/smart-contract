package tests

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestContracts is the entry point for testing contract based functions.
func TestContracts(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createContract)
	t.Run("oracle", oracleContract)
	t.Run("amendment", contractAmendment)
	t.Run("listAmendment", contractListAmendment)
	t.Run("oracleAmendment", contractOracleAmendment)
	t.Run("proposalAmendment", contractProposalAmendment)
}

func createContract(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	// New Contract Offer
	offerData := actions.ContractOffer{
		ContractName:        "Test Name",
		BodyOfAgreementType: 3,
		BodyOfAgreement:     []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: &actions.EntityField{
			Type:           "I",
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: 1, Name: "John Smith"}},
		},
		VotingSystems:  []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
		HolderProposal: true,
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              false, // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	offerData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize contract permissions : %v", tests.Failed, err)
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(1000, script))

	// Data output
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
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

	var responseMsg actions.Action
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
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tContract offer response not a reject : %s", tests.Failed, responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsMsgMalformed {
		t.Fatalf("\t%s\tWrong reject code for contract offer reject : %d", tests.Failed,
			reject.RejectionCode)
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
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractName != offerData.ContractName {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractName,
			offerData.ContractName)
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

func oracleContract(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	if err := test.Headers.Populate(ctx, 50000, 12); err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	// New Contract Offer
	offerData := actions.ContractOffer{
		ContractName:        "Test Name",
		BodyOfAgreementType: 2,
		BodyOfAgreement:     []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: &actions.EntityField{
			Type:           "I",
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: 1, Name: "John Smith"}},
		},
		AdminOracle: &actions.OracleField{
			Entity: &actions.EntityField{
				Name: "KYC, Inc.",
			},
			URL:       "bsv.kyc.com",
			PublicKey: oracleKey.Key.PublicKey().Bytes()},
		AdminOracleSigBlockHeight: uint32(test.Headers.LastHeight(ctx) - 5),
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{
				Name:                "Relative 50",
				VoteType:            "R",
				ThresholdPercentage: 50,
				HolderProposalFee:   50000,
			},
		},
		HolderProposal: true,
	}

	blockHash, err := test.Headers.Hash(ctx, int(offerData.AdminOracleSigBlockHeight))
	sigHash, err := protocol.ContractOracleSigHash(ctx, []bitcoin.RawAddress{issuerKey.Address},
		[]*actions.EntityField{offerData.Issuer}, blockHash, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature hash : %v", tests.Failed, err)
	}
	sig, err := oracleKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	offerData.AdminOracleSignature = sig.Bytes()

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              false, // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(offerData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	offerData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize contract permissions : %v", tests.Failed, err)
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(2)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(750000, script))

	// Data output
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	holdingsChannel := &holdings.CacheChannel{}
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		holdingsChannel)

	if err := server.SyncWallet(ctx); err != nil {
		t.Fatalf("Failed to load wallet : %s", err)
	}

	server.SetAlternateResponder(respondTx)
	server.SetInSync()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			t.Logf("Server failed : %s", err)
		}
	}()

	time.Sleep(time.Second)

	t.Logf("Contract offer tx : %s", offerTx.TxHash().String())
	if _, err := server.HandleTx(ctx, offerTx); err != nil {
		t.Fatalf("\t%s\tContract handle failed : %v", tests.Failed, err)
	}

	if err := server.HandleTxState(ctx, handlers.ListenerMsgTxStateSafe, *offerTx.TxHash()); err != nil {
		t.Fatalf("\t%s\tContract offer handle state failed : %v", tests.Failed, err)
	}

	var firstResponse *wire.MsgTx // Request tx is re-broadcast now
	var response *wire.MsgTx
	for {
		if firstResponse == nil {
			firstResponse = getResponse()
			time.Sleep(time.Millisecond)
			continue
		}
		response = getResponse()
		if response != nil {
			break
		}

		time.Sleep(time.Millisecond)
	}

	var responseMsg actions.Action
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tContract offer response doesn't contain tokenized op return", tests.Failed)
	}
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tContract offer response not a reject : %s", tests.Failed, responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsInvalidSignature {
		t.Fatalf("\t%s\tWrong reject code for contract offer reject", tests.Failed)
	}

	t.Logf("\t%s\tContract offer with invalid signature rejection : (%d) %s", tests.Success,
		reject.RejectionCode, reject.Message)

	// Fix signature and retry
	sigHash, err = protocol.ContractOracleSigHash(ctx, []bitcoin.RawAddress{issuerKey.Address},
		[]*actions.EntityField{offerData.Issuer}, blockHash, 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature hash : %v", tests.Failed, err)
	}
	sig, err = oracleKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	offerData.AdminOracleSignature = sig.Bytes()

	// Update Data output
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut[len(offerTx.TxOut)-1] = wire.NewTxOut(0, script)

	t.Logf("Contract offer tx : %s", offerTx.TxHash().String())
	if _, err := server.HandleTx(ctx, offerTx); err != nil {
		t.Fatalf("\t%s\tContract handle failed : %v", tests.Failed, err)
	}

	if err := server.HandleTxState(ctx, handlers.ListenerMsgTxStateSafe, *offerTx.TxHash()); err != nil {
		t.Fatalf("\t%s\tContract offer handle state failed : %v", tests.Failed, err)
	}

	firstResponse = nil // Request is re-broadcast
	for {
		if firstResponse == nil {
			firstResponse = getResponse()
			time.Sleep(time.Millisecond)
			continue
		}
		response = getResponse()
		if response != nil {
			break
		}

		time.Sleep(time.Millisecond)
	}

	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tContract offer response doesn't contain tokenized op return", tests.Failed)
	}
	if responseMsg.Code() != "C2" {
		t.Fatalf("\t%s\tContract offer response not a formation : %s", tests.Failed,
			responseMsg.Code())
	}
	_, ok = responseMsg.(*actions.ContractFormation)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to formation", tests.Failed)
	}

	t.Logf("\t%s\tContract offer with valid signature accepted", tests.Success)

	server.Stop(ctx)
	wg.Wait()
}

func contractAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100015, issuerKey.Address)

	amendmentData := actions.ContractAmendment{
		ContractRevision: 0,
	}

	fip := actions.FieldIndexPath{actions.ContractFieldContractName}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test Contract 2"),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
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
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractName != "Test Contract 2" {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractName, "Test Contract 2")
	}

	t.Logf("\t%s\tVerified contract name : %s", tests.Success, ct.ContractName)
}

func contractListAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              false, // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			Fields: []actions.FieldIndexPath{
				actions.FieldIndexPath{actions.ContractFieldOracles, actions.OracleFieldEntity},
			},
		},
	}

	err := mockUpContractWithPermissions(ctx, "Test Contract",
		"This is a mock contract and means nothing.", "I", 1, "John Bitcoin", permissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100002, issuerKey.Address)

	amendmentData := actions.ContractAmendment{
		ContractRevision: 0,
	}

	fip := actions.FieldIndexPath{
		actions.ContractFieldOracles,
		1, // Oracles list index to second item
		actions.OracleFieldEntity,
		actions.EntityFieldName,
	}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Operation:      0, // Modify element
		Data:           []byte("KYC 2 Updated"),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, amendmentTx)
	t.Logf("Contract Oracle Amendment Tx : %s", amendmentTx.TxHash().String())

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAmendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "C2")

	// Check oracle name
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.Oracles[1].Entity.Name != "KYC 2 Updated" {
		t.Fatalf("\t%s\tContract oracle 2 name incorrect : \"%s\" != \"%s\"", tests.Failed,
			ct.Oracles[1].Entity.Name, "KYC 2 Updated")
	}

	t.Logf("\t%s\tVerified contract oracle 2 name : %s", tests.Success, ct.Oracles[1].Entity.Name)

	// Try to modify URL, which should not be allowed
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	amendmentData = actions.ContractAmendment{
		ContractRevision: 0,
	}

	fip = actions.FieldIndexPath{
		actions.ContractFieldOracles,
		1, // Oracles list index to second item
		actions.OracleFieldURL,
	}
	fipBytes, _ = fip.Bytes()
	amendmentData.Amendments = []*actions.AmendmentField{
		&actions.AmendmentField{
			FieldIndexPath: fipBytes,
			Operation:      0, // Modify element
			Data:           []byte("bsv2.updated.kyc.com"),
		},
	}

	// Build amendment transaction
	amendmentTx = wire.NewMsgTx(2)

	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err = inspector.NewTransactionFromWire(ctx, amendmentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, amendmentTx)
	t.Logf("Contract Oracle Amendment Tx : %s", amendmentTx.TxHash().String())

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAmendment rejected", tests.Success)

	// Check the response
	checkResponse(t, "M2")

	// Check oracle name
	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.Oracles[1].URL != "bsv2.kyc.com" {
		t.Fatalf("\t%s\tContract oracle 2 URL incorrect : \"%s\" != \"%s\"", tests.Failed,
			ct.Oracles[1].URL, "bsv2.kyc.com")
	}

	t.Logf("\t%s\tVerified contract oracle 2 URL : %s", tests.Success, ct.Oracles[1].URL)
}

func contractOracleAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	if err := test.Headers.Populate(ctx, 50000, 12); err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	ct, err := mockUpContractWithAdminOracle(ctx, "Test Contract",
		"This is a mock contract and means nothing.", "I", 1, "John Bitcoin")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	blockHeight := uint32(test.Headers.LastHeight(ctx) - 4)
	blockHash, err := test.Headers.Hash(ctx, int(blockHeight))
	sigHash, err := protocol.ContractOracleSigHash(ctx, []bitcoin.RawAddress{issuer2Key.Address},
		[]*actions.EntityField{ct.Issuer}, blockHash, 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature hash : %v", tests.Failed, err)
	}
	signature, err := oracleKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}

	amendmentData := actions.ContractAmendment{
		ContractRevision:            0,
		ChangeAdministrationAddress: true,
	}

	fip := actions.FieldIndexPath{actions.ContractFieldAdminOracleSignature}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           signature.Bytes(),
	})

	var blockHeightBuf bytes.Buffer
	binary.Write(&blockHeightBuf, binary.LittleEndian, &blockHeight)
	fip = actions.FieldIndexPath{actions.ContractFieldAdminOracleSigBlockHeight}
	fipBytes, _ = fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           blockHeightBuf.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	// From issuer
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100015, issuerKey.Address)
	amendmentInputHash := fundingTx.TxHash()
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// From issuer 2
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100016, issuer2Key.Address)
	amendmentInputHash = fundingTx.TxHash()
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
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
	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if !ct.AdministrationAddress.Equal(issuer2Key.Address) {
		t.Fatalf("\t%s\tContract admin incorrect : \"%x\" != \"%x\"", tests.Failed,
			ct.AdministrationAddress.Bytes(), issuer2Key.Address.Bytes())
	}
}

func contractProposalAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, true, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	fip := actions.FieldIndexPath{actions.ContractFieldContractType}
	fipBytes, _ := fip.Bytes()
	assetAmendment := actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("New Type"),
	}
	err = mockUpContractAmendmentVote(ctx, 0, 0, &assetAmendment)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpVoteResultTx(ctx, "A")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote result : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000015, issuerKey.Address)

	amendmentData := actions.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          testVoteResultTxId.Bytes(),
	}

	amendmentData.Amendments = append(amendmentData.Amendments, &assetAmendment)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
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
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractType != "New Type" {
		t.Fatalf("\t%s\tContract type incorrect : \"%s\" != \"%s\"", tests.Failed, ct.ContractType, "New Type")
	}

	t.Logf("\t%s\tVerified contract type : %s", tests.Success, ct.ContractType)
}

func mockUpContract(ctx context.Context, name, agreement string, issuerType string, issuerRole uint32, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) error {

	var contractData = state.Contract{
		Address:             test.ContractKey.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,
		CreatedAt:              protocol.CurrentTimestamp(),
		UpdatedAt:              protocol.CurrentTimestamp(),
		AdministrationAddress:  issuerKey.Address,
		MasterAddress:          test.MasterKey.Address,
	}

	// Define permissions for contact fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContract2(ctx context.Context, name, agreement string, issuerType string, issuerRole uint32, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) error {

	var contractData = state.Contract{
		Address:             test.Contract2Key.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		CreatedAt:             protocol.CurrentTimestamp(),
		UpdatedAt:             protocol.CurrentTimestamp(),
		AdministrationAddress: issuerKey.Address,
		MasterAddress:         test.Master2Key.Address,
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpOtherContract(ctx context.Context, key *wallet.Key, name, agreement string,
	issuerType string, issuerRole uint32, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) error {

	var contractData = state.Contract{
		Address:             key.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		CreatedAt:             protocol.CurrentTimestamp(),
		UpdatedAt:             protocol.CurrentTimestamp(),
		AdministrationAddress: issuerKey.Address,
		MasterAddress:         test.Master2Key.Address,
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContractWithOracle(ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string) error {
	var contractData = state.Contract{
		Address:             test.ContractKey.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		CreatedAt:             protocol.CurrentTimestamp(),
		UpdatedAt:             protocol.CurrentTimestamp(),
		AdministrationAddress: issuerKey.Address,
		MasterAddress:         test.MasterKey.Address,
		Oracles: []*actions.OracleField{
			&actions.OracleField{
				Entity: &actions.EntityField{
					Name: "KYC, Inc.",
				},
				URL:       "bsv.kyc.com",
				PublicKey: oracleKey.Key.PublicKey().Bytes(),
			},
		},
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              false, // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	if err := contract.ExpandOracles(ctx, &contractData); err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContractWithAdminOracle(ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string) (*state.Contract, error) {

	var contractData = state.Contract{
		Address:             test.ContractKey.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		AdminOracle: &actions.OracleField{
			Entity: &actions.EntityField{
				Name: "KYC, Inc.",
			},
			URL:       "bsv.kyc.com",
			PublicKey: oracleKey.Key.PublicKey().Bytes()},
		AdminOracleSigBlockHeight: uint32(test.Headers.LastHeight(ctx) - 5),
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		CreatedAt:             protocol.CurrentTimestamp(),
		UpdatedAt:             protocol.CurrentTimestamp(),
		AdministrationAddress: issuerKey.Address,
		MasterAddress:         test.MasterKey.Address,
		Oracles: []*actions.OracleField{
			&actions.OracleField{
				Entity: &actions.EntityField{
					Name: "KYC, Inc.",
				},
				URL:       "bsv.kyc.com",
				PublicKey: oracleKey.Key.PublicKey().Bytes(),
			},
		},
	}

	blockHash, err := test.Headers.Hash(ctx, int(contractData.AdminOracleSigBlockHeight))
	sigHash, err := protocol.ContractOracleSigHash(ctx, []bitcoin.RawAddress{issuerKey.Address},
		[]*actions.EntityField{contractData.Issuer}, blockHash, 0)
	if err != nil {
		return nil, err
	}
	sig, err := oracleKey.Key.Sign(sigHash)
	if err != nil {
		return nil, err
	}
	contractData.AdminOracleSignature = sig.Bytes()

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(contractData.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return nil, err
	}

	if err := contract.ExpandOracles(ctx, &contractData); err != nil {
		return nil, err
	}

	return &contractData, contract.Save(ctx, test.MasterDB, &contractData)
}

func mockUpContractWithPermissions(ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string, permissions actions.Permissions) error {

	var contractData = state.Contract{
		Address:             test.ContractKey.Address,
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000}},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		CreatedAt:             protocol.CurrentTimestamp(),
		UpdatedAt:             protocol.CurrentTimestamp(),
		AdministrationAddress: issuerKey.Address,
		MasterAddress:         test.MasterKey.Address,
		Oracles: []*actions.OracleField{
			&actions.OracleField{
				Entity: &actions.EntityField{
					Name: "KYC 1, Inc.",
				},
				URL:       "bsv1.kyc.com",
				PublicKey: oracleKey.Key.PublicKey().Bytes(),
			},
			&actions.OracleField{
				Entity: &actions.EntityField{
					Name: "KYC 2, Inc.",
				},
				URL:       "bsv2.kyc.com",
				PublicKey: oracleKey.Key.PublicKey().Bytes(),
			},
		},
	}

	var err error
	contractData.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	if err := contract.ExpandOracles(ctx, &contractData); err != nil {
		return err
	}

	return contract.Save(ctx, test.MasterDB, &contractData)
}
