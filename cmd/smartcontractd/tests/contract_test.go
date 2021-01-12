package tests

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
	"github.com/tokenized/spynode/pkg/client"
)

// TestContracts is the entry point for testing contract based functions.
func TestContracts(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createContract)
	t.Run("masterAddress", masterAddress)
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
	offerTx := wire.NewMsgTx(1)

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
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractType != offerData.ContractType {
		t.Fatalf("\t%s\tContract type incorrect : %d != %d", tests.Failed, ct.ContractType,
			offerData.ContractType)
	}

	t.Logf("\t%s\tVerified contract type", tests.Success)

	if !ct.HolderProposal {
		t.Fatalf("\t%s\tContract holder proposal incorrect : %t", tests.Failed, ct.HolderProposal)
	}

	t.Logf("\t%s\tVerified holder proposal", tests.Success)

	if ct.AdministrationProposal {
		t.Fatalf("\t%s\tContract issuer proposal incorrect : %t", tests.Failed, ct.AdministrationProposal)
	}

	t.Logf("\t%s\tVerified issuer proposal", tests.Success)
}

func masterAddress(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	addressText := "19ec6LTfLjbm9Ea1oJavAey4u6g1eNCzeu"
	address, err := bitcoin.DecodeAddress(addressText)
	if err != nil {
		t.Fatalf("\t%s\tFailed to parse address : %v", tests.Failed, err)
	}
	ra := bitcoin.NewRawAddressFromAddress(address)

	// New Contract Offer
	offerData := actions.ContractOffer{
		ContractName:        "Test Name",
		BodyOfAgreementType: 2,
		BodyOfAgreement:     []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: &actions.EntityField{
			Type:           "I",
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: 1, Name: "John Smith"}},
		},
		VotingSystems:  []*actions.VotingSystemField{&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000}},
		HolderProposal: true,
		MasterAddress:  []byte(addressText),
	}

	// Create funding tx
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000014, issuerKey.Address)

	// Build offer transaction
	offerTx := wire.NewMsgTx(1)

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
	offerData.MasterAddress = ra.Bytes()

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
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.ContractType != offerData.ContractType {
		t.Fatalf("\t%s\tContract type incorrect : %d != %d", tests.Failed, ct.ContractType,
			offerData.ContractType)
	}

	t.Logf("\t%s\tVerified contract name", tests.Success)

	cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed contract formation fetch %s", tests.Failed, err)
	}

	if cf.Issuer.Administration[0].Name != offerData.Issuer.Administration[0].Name {
		t.Fatalf("\t%s\tContract issuer name incorrect : \"%s\" != \"%s\"", tests.Failed,
			cf.Issuer.Administration[0].Name, "John Smith")
	}

	t.Logf("\t%s\tVerified issuer name : %s", tests.Success, cf.Issuer.Administration[0].Name)

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

	oracleKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle key : %v", tests.Failed, err)
	}

	mockIdentityContract(t, ctx, test.Contract2Key.Key, oracleKey.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

	// New Contract Offer
	offerData := actions.ContractOffer{
		ContractName:        "Test Name",
		BodyOfAgreementType: 2,
		BodyOfAgreement:     []byte("This is a test contract and not to be used for any official purpose."),
		Issuer: &actions.EntityField{
			Type: "I",
			Administration: []*actions.AdministratorField{
				&actions.AdministratorField{
					Type: 1,
					Name: "John Smith",
				},
			},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{
				Name:                "Relative 50",
				VoteType:            "R",
				ThresholdPercentage: 50,
				HolderProposalFee:   50000,
			},
		},
		HolderProposal: true,
		AdminIdentityCertificates: []*actions.AdminIdentityCertificateField{
			&actions.AdminIdentityCertificateField{
				EntityContract: test.Contract2Key.Address.Bytes(),
				BlockHeight:    uint32(test.Headers.LastHeight(ctx) - 5),
			},
		},
	}

	blockHash, err := test.Headers.Hash(ctx, int(offerData.AdminIdentityCertificates[0].BlockHeight))
	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, issuerKey.Address,
		offerData.Issuer, *blockHash, 0, 0)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature hash : %v", tests.Failed, err)
	}
	sig, err := oracleKey.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	offerData.AdminIdentityCertificates[0].Signature = sig.Bytes()

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
	offerTx := wire.NewMsgTx(1)

	offerInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	offerTx.TxIn = append(offerTx.TxIn, wire.NewTxIn(wire.NewOutPoint(offerInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(75000, script))

	// Data output
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut = append(offerTx.TxOut, wire.NewTxOut(0, script))

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	holdingsChannel := &holdings.CacheChannel{}
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB, nil,
		test.Headers, test.Scheduler, tracer, test.UTXOs, holdingsChannel)

	if err := server.Load(ctx); err != nil {
		t.Fatalf("Failed to load server : %s", err)
	}

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

	t.Logf("Contract offer tx : %s", offerTx.TxHash())
	offerCtx := &client.Tx{
		Tx:      offerTx,
		Outputs: []*wire.TxOut{fundingTx.TxOut[0]},
		State: client.TxState{
			Safe: true,
		},
	}
	server.HandleTx(ctx, offerCtx)

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

	clearResponses()

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
	sigHash, err = protocol.ContractAdminIdentityOracleSigHash(ctx, issuerKey.Address,
		offerData.Issuer, *blockHash, 0, 1)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature hash : %v", tests.Failed, err)
	}
	sig, err = oracleKey.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	offerData.AdminIdentityCertificates[0].Signature = sig.Bytes()

	// Update Data output
	script, err = protocol.Serialize(&offerData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	offerTx.TxOut[len(offerTx.TxOut)-1] = wire.NewTxOut(0, script)

	t.Logf("Contract offer tx : %s", offerTx.TxHash())
	server.HandleTx(ctx, offerCtx)

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
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, true, false, false)

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
	amendmentTx := wire.NewMsgTx(1)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2050, script))

	// Data output
	var err error
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

	t.Logf("Amendment tx : %s", amendmentItx.Hash.String())

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept contract amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tContract Amendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "C2")

	// Check contract name
	cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if cf.ContractName != "Test Contract 2" {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, cf.ContractName,
			"Test Contract 2")
	}

	t.Logf("\t%s\tVerified contract name : %s", tests.Success, cf.ContractName)
}

func contractListAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	newKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("\t%s\tFailed to generate key : %v", tests.Failed, err)
	}

	newAddress, err := newKey.RawAddress()
	if err != nil {
		t.Fatalf("\t%s\tFailed to create address : %v", tests.Failed, err)
	}

	mockIdentityContract(t, ctx, newKey, oracleKey.Key.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

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
				actions.FieldIndexPath{actions.ContractFieldOracles, actions.OracleFieldEntityContract},
			},
		},
	}

	mockUpContractWithPermissions(t, ctx, "Test Contract",
		"This is a mock contract and means nothing.", "I", 1, "John Bitcoin", permissions)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100002, issuerKey.Address)

	amendmentData := actions.ContractAmendment{
		ContractRevision: 0,
	}

	fip := actions.FieldIndexPath{
		actions.ContractFieldOracles,
		1, // Oracles list index to second item
		actions.OracleFieldEntityContract,
	}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Operation:      0, // Modify element
		Data:           newAddress.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2200, script))

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

	// Check oracle contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %s", tests.Failed, err)
	}

	if !bytes.Equal(ct.Oracles[1].EntityContract, newAddress.Bytes()) {
		t.Fatalf("\t%s\tContract oracle 2 contract incorrect : \"%x\" != \"%x\"", tests.Failed,
			ct.Oracles[1].EntityContract, newAddress.Bytes())
	}

	t.Logf("\t%s\tVerified contract oracle 2 contract : %x", tests.Success, ct.Oracles[1].EntityContract)

	// Try to modify URL, which should not be allowed
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100004, issuerKey.Address)

	amendmentData = actions.ContractAmendment{
		ContractRevision: 0,
	}

	var buf bytes.Buffer
	if err := bitcoin.WriteBase128VarInt(&buf, uint64(actions.ServiceTypeAuthorityOracle)); err != nil {
		t.Fatalf("\t%s\tFailed to write oracle type : %s", tests.Failed, err)
	}

	fip = actions.FieldIndexPath{
		actions.ContractFieldOracles,
		1, // Oracles list index to second item
		actions.OracleFieldOracleTypes,
		0, // Oracles list index to first item
	}
	fipBytes, _ = fip.Bytes()
	amendmentData.Amendments = []*actions.AmendmentField{
		&actions.AmendmentField{
			FieldIndexPath: fipBytes,
			Operation:      0, // Modify element
			Data:           buf.Bytes(),
		},
	}

	// Build amendment transaction
	amendmentTx = wire.NewMsgTx(1)

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
	checkResponse(t, "M2") // Wrong revision

	// Check oracle type
	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.Oracles[1].OracleTypes[0] != actions.ServiceTypeIdentityOracle {
		t.Fatalf("\t%s\tContract oracle 2 type 1 incorrect : %d != %d", tests.Failed,
			ct.Oracles[1].OracleTypes[0], actions.ServiceTypeIdentityOracle)
	}

	t.Logf("\t%s\tVerified contract oracle 2 type 1 : %d", tests.Success, ct.Oracles[1].OracleTypes[0])

	amendmentData.ContractRevision = 1

	// Build amendment transaction
	amendmentTx = wire.NewMsgTx(1)

	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2100, script))

	// Data output
	script, err = protocol.Serialize(&amendmentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	action, err := protocol.Deserialize(script, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to deserialize amendment : %s", tests.Failed, err)
	}

	desAmend, ok := action.(*actions.ContractAmendment)
	if !ok {
		t.Fatalf("\t%s\tDeserialized amendment wrong type", tests.Failed)
	}

	if !bytes.Equal(fipBytes, desAmend.Amendments[0].FieldIndexPath) {
		t.Fatalf("\t%s\tContract amendment wrong FIP bytes : \"%x\" != \"%x\"", tests.Failed,
			fipBytes, desAmend.Amendments[0].FieldIndexPath)
	}

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
	if err != nil {
		t.Errorf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	} else {
		t.Logf("\t%s\tAmendment accepted", tests.Success)
	}

	// Check the response
	checkResponse(t, "C2")

	// Check oracle type
	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if ct.Oracles[1].OracleTypes[0] != actions.ServiceTypeAuthorityOracle {
		t.Fatalf("\t%s\tContract oracle 2 type 1 incorrect : %d != %d", tests.Failed,
			ct.Oracles[1].OracleTypes[0], actions.ServiceTypeAuthorityOracle)
	}

	t.Logf("\t%s\tVerified contract oracle 2 type 1 : %d", tests.Success, ct.Oracles[1].OracleTypes[0])
}

func contractOracleAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	if err := test.Headers.Populate(ctx, 50000, 12); err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	ct, cf := mockUpContractWithAdminOracle(t, ctx, "Test Contract",
		"This is a mock contract and means nothing.", "I", 1, "John Bitcoin")

	blockHeight := uint32(test.Headers.LastHeight(ctx) - 4)
	blockHash, err := test.Headers.Hash(ctx, int(blockHeight))
	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, issuer2Key.Address, cf.Issuer,
		*blockHash, 0, 1)
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

	fip := actions.FieldIndexPath{
		actions.ContractFieldAdminIdentityCertificates,
		0, // index to first certificate
		actions.AdminIdentityCertificateFieldSignature,
	}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           signature.Bytes(),
	})

	var blockHeightBuf bytes.Buffer
	if err := bitcoin.WriteBase128VarInt(&blockHeightBuf, uint64(blockHeight)); err != nil {
		t.Fatalf("\t%s\tFailed to serialize block height : %v", tests.Failed, err)
	}

	fip = actions.FieldIndexPath{
		actions.ContractFieldAdminIdentityCertificates,
		0, // index to first certificate
		actions.AdminIdentityCertificateFieldBlockHeight,
	}
	fipBytes, _ = fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           blockHeightBuf.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

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
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2200, script))

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
	t.Logf("Contract Oracle Amendment Tx : %s", amendmentTx.TxHash().String())

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept contract amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tContract Amendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "C2")

	// Check contract name
	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if !ct.AdminAddress.Equal(issuer2Key.Address) {
		t.Fatalf("\t%s\tContract admin incorrect : \"%x\" != \"%x\"", tests.Failed,
			ct.AdminAddress.Bytes(), issuer2Key.Address.Bytes())
	}
}

func contractProposalAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, true, false)

	fip := actions.FieldIndexPath{actions.ContractFieldContractName}
	fipBytes, _ := fip.Bytes()
	assetAmendment := actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("New Name"),
	}
	if err := mockUpContractAmendmentVote(ctx, 0, 0, &assetAmendment); err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	if err := mockUpVoteResultTx(ctx, "A"); err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote result : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000015, issuerKey.Address)

	amendmentData := actions.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          testVoteResultTxId.Bytes(),
	}

	amendmentData.Amendments = append(amendmentData.Amendments, &assetAmendment)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2200, script))

	// Data output
	var err error
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
	cf, err := contract.FetchContractFormation(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if cf.ContractName != "New Name" {
		t.Fatalf("\t%s\tContract name incorrect : \"%s\" != \"%s\"", tests.Failed, cf.ContractName,
			"New Name")
	}

	t.Logf("\t%s\tVerified contract name : %s", tests.Success, cf.ContractName)
}

func mockUpContract(t testing.TB, ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string, issuerProposal, holderProposal, permitted, issuer, holder bool) (*state.Contract, *actions.ContractFormation) {

	contractData := &state.Contract{
		Address:   test.ContractKey.Address,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		AdminAddress:  issuerKey.Address.Bytes(),
		MasterAddress: test.MasterKey.Address.Bytes(),
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(cf.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	cf.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize contract permissions : %s", err)
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, test.ContractKey.Address, cf,
		test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockUpContract2(t testing.TB, ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string, issuerProposal, holderProposal, permitted, issuer, holder bool) (*state.Contract, *actions.ContractFormation) {

	contractData := &state.Contract{
		Address:   test.Contract2Key.Address,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		AdminAddress:  issuer2Key.Address.Bytes(),
		MasterAddress: test.Master2Key.Address.Bytes(),
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(cf.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	cf.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize contract permissions : %s", err)
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, test.Contract2Key.Address, cf,
		test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockUpOtherContract(t testing.TB, ctx context.Context, ra bitcoin.RawAddress, name, agreement string,
	issuerType string, issuerRole uint32, issuerName string,
	issuerProposal, holderProposal, permitted, issuer, holder bool) (*state.Contract, *actions.ContractFormation) {

	contractData := &state.Contract{
		Address:   ra,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: issuerProposal,
		HolderProposal:         holderProposal,
		ContractFee:            1000,

		AdminAddress:  issuerKey.Address.Bytes(),
		MasterAddress: test.Master2Key.Address.Bytes(),
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(cf.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	var err error
	cf.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize contract permissions : %s", err)
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, ra, cf, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockUpContractWithOracle(t testing.TB, ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string) (*state.Contract, *actions.ContractFormation) {

	oracleContractKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate oracle contract key : %s", err)
	}

	oracleAddress, err := oracleContractKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create oracle contract address : %s", err)
	}

	mockIdentityContract(t, ctx, oracleContractKey, oracleKey.Key.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

	contractData := &state.Contract{
		Address:   test.ContractKey.Address,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		Oracles: []*actions.OracleField{
			&actions.OracleField{
				OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
				EntityContract: oracleAddress.Bytes(),
			},
		},

		AdminAddress:  issuerKey.Address.Bytes(),
		MasterAddress: test.MasterKey.Address.Bytes(),
	}

	// Define permissions for contract fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
		},
	}

	permissions[0].VotingSystemsAllowed = make([]bool, len(cf.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	cf.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize contract permissions : %s", err)
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, test.ContractKey.Address, cf,
		test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockUpContractWithAdminOracle(t testing.TB, ctx context.Context, name, agreement string,
	issuerType string, issuerRole uint32, issuerName string) (*state.Contract, *actions.ContractFormation) {

	oracleContractKey, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate oracle contract key : %s", err)
	}

	oracleAddress, err := oracleContractKey.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create oracle contract address : %s", err)
	}

	mockIdentityContract(t, ctx, oracleContractKey, oracleKey.Key.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

	sigBlockHeight := uint32(test.Headers.LastHeight(ctx) - 5)

	contractData := &state.Contract{
		Address:   test.ContractKey.Address,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		AdminAddress:  issuerKey.Address.Bytes(),
		MasterAddress: test.MasterKey.Address.Bytes(),

		Oracles: []*actions.OracleField{
			&actions.OracleField{
				OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
				EntityContract: oracleAddress.Bytes(),
			},
		},

		AdminIdentityCertificates: []*actions.AdminIdentityCertificateField{
			&actions.AdminIdentityCertificateField{
				EntityContract: oracleAddress.Bytes(),
				BlockHeight:    sigBlockHeight,
			},
		},
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	blockHash, err := test.Headers.Hash(ctx, int(sigBlockHeight))
	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, issuerKey.Address, cf.Issuer,
		*blockHash, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create sig hash : %s", err)
	}
	sig, err := oracleKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("Failed to create signature : %s", err)
	}
	cf.AdminIdentityCertificates[0].Signature = sig.Bytes()

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, test.ContractKey.Address, cf,
		test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockUpContractWithPermissions(t testing.TB, ctx context.Context, name, agreement string, issuerType string,
	issuerRole uint32, issuerName string, permissions actions.Permissions) (*state.Contract, *actions.ContractFormation) {

	oracleContract1Key, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate oracle contract key : %s", err)
	}

	oracle1Address, err := oracleContract1Key.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create oracle contract address : %s", err)
	}

	mockIdentityContract(t, ctx, oracleContract1Key, oracleKey.Key.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

	oracleContract2Key, err := bitcoin.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("Failed to generate oracle contract key : %s", err)
	}

	oracle2Address, err := oracleContract2Key.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create oracle contract address : %s", err)
	}

	mockIdentityContract(t, ctx, oracleContract2Key, oracle2Key.Key.PublicKey(),
		actions.EntitiesPublicCompany, actions.RolesCEO, "John Bitcoin")

	contractData := &state.Contract{
		Address:   test.ContractKey.Address,
		CreatedAt: protocol.CurrentTimestamp(),
		UpdatedAt: protocol.CurrentTimestamp(),
	}

	cf := &actions.ContractFormation{
		ContractName:        name,
		BodyOfAgreementType: 1,
		BodyOfAgreement:     []byte(agreement),
		Issuer: &actions.EntityField{
			Type:           issuerType,
			Administration: []*actions.AdministratorField{&actions.AdministratorField{Type: issuerRole, Name: issuerName}},
		},
		VotingSystems: []*actions.VotingSystemField{
			&actions.VotingSystemField{Name: "Relative 50", VoteType: "R", ThresholdPercentage: 50, HolderProposalFee: 50000},
			&actions.VotingSystemField{Name: "Absolute 75", VoteType: "A", ThresholdPercentage: 75, HolderProposalFee: 25000},
		},
		AdministrationProposal: false,
		HolderProposal:         false,
		ContractFee:            1000,

		Oracles: []*actions.OracleField{
			&actions.OracleField{
				OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
				EntityContract: oracle1Address.Bytes(),
			},
			&actions.OracleField{
				OracleTypes:    []uint32{actions.ServiceTypeIdentityOracle},
				EntityContract: oracle2Address.Bytes(),
			},
		},

		AdminAddress:  issuerKey.Address.Bytes(),
		MasterAddress: test.MasterKey.Address.Bytes(),
	}

	cf.ContractPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize contract permissions : %s", err)
	}

	if err := node.Convert(ctx, cf, contractData); err != nil {
		t.Fatalf("Failed to convert contract : %s", err)
	}

	if err := contract.ExpandOracles(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to expand oracles : %s", err)
	}

	if err := contract.Save(ctx, test.MasterDB, contractData, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, test.ContractKey.Address, cf,
		test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	return contractData, cf
}

func mockIdentityContract(t testing.TB, ctx context.Context, key bitcoin.Key,
	publicKey bitcoin.PublicKey, issuerType string, issuerRole uint32, issuerName string) {

	cf := &actions.ContractFormation{
		ContractType: actions.ContractTypeEntity,
		ContractName: "Test Identity Oracle",
		Issuer: &actions.EntityField{
			Type: issuerType,
			Administration: []*actions.AdministratorField{
				&actions.AdministratorField{
					Type: issuerRole,
					Name: issuerName,
				},
			},
		},
		ContractFee: 1000,
		Services: []*actions.ServiceField{
			&actions.ServiceField{
				Type:      actions.ServiceTypeIdentityOracle,
				URL:       "tokenized.com/identity",
				PublicKey: publicKey.Bytes(),
			},
		},
	}

	ra, err := key.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create identity contract address : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, ra, cf, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save identity contract address : %s", err)
	}
}

func mockOperatorContract(t testing.TB, ctx context.Context, key bitcoin.Key,
	publicKey bitcoin.PublicKey, issuerType string, issuerRole uint32, issuerName string) {

	cf := &actions.ContractFormation{
		ContractType: actions.ContractTypeEntity,
		ContractName: "Test ContractOperator",
		Issuer: &actions.EntityField{
			Type: issuerType,
			Administration: []*actions.AdministratorField{
				&actions.AdministratorField{
					Type: issuerRole,
					Name: issuerName,
				},
			},
		},
		ContractFee: 1000,
		Services: []*actions.ServiceField{
			&actions.ServiceField{
				Type:      actions.ServiceTypeContractOperator,
				URL:       "tokenized.com/contracts",
				PublicKey: publicKey.Bytes(),
			},
		},
	}

	ra, err := key.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create operator contract address : %s", err)
	}

	if err := contract.SaveContractFormation(ctx, test.MasterDB, ra, cf, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("Failed to save operator contract address : %s", err)
	}
}
