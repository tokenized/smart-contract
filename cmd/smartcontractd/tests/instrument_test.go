package tests

import (
	"bytes"
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/instrument"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestInstruments is the entry point for testing instrument based functions.
func TestInstruments(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createInstrument)
	t.Run("adminMemberInstrument", adminMemberInstrument)
	t.Run("index", instrumentIndex)
	t.Run("amendment", instrumentAmendment)
	t.Run("proposalAmendment", instrumentProposalAmendment)
	t.Run("duplicateInstrument", duplicateInstrument)
}

func createInstrument(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testInstrumentType = instruments.CodeShareCommon
	testInstrumentCodes = []bitcoin.Hash20{protocol.InstrumentCodeFromContract(test.ContractKey.Address, 0)}

	// Create InstrumentDefinition message
	instrumentData := actions.InstrumentDefinition{
		InstrumentType:             testInstrumentType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	instrumentPayloadData := instruments.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	instrumentData.InstrumentPayload, err = instrumentPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	t.Logf("Instrument Permissions : 0x%x", instrumentData.InstrumentPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	// Build instrument definition transaction
	instrumentTx := wire.NewMsgTx(1)

	instrumentInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument definition : %v", tests.Failed, err)
	}
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(0, script))

	instrumentItx, err := inspector.NewTransactionFromWire(ctx, instrumentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx : %v", tests.Failed, err)
	}

	err = instrumentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx)

	err = a.Trigger(ctx, "SEE", instrumentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	// Check issuer balance
	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != instrumentData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			instrumentData.AuthorizedTokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.InstrumentType != instrumentData.InstrumentType {
		t.Fatalf("\t%s\tInstrument type incorrect : %s != %s", tests.Failed, as.InstrumentType,
			instrumentData.InstrumentType)
	}

	if as.AuthorizedTokenQty != instrumentData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tInstrument token quantity incorrect : %d != %d", tests.Failed,
			as.AuthorizedTokenQty, instrumentData.AuthorizedTokenQty)
	}

	t.Logf("\t%s\tVerified instrument type : %s", tests.Success, as.InstrumentType)
}

func adminMemberInstrument(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testInstrumentType = instruments.CodeShareCommon
	testInstrumentCodes[0] = protocol.InstrumentCodeFromContract(test.ContractKey.Address, 0)

	// Create InstrumentDefinition message
	instrumentData := actions.InstrumentDefinition{
		InstrumentType:             instruments.CodeMembership,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         5,
	}

	instrumentPayloadData := instruments.Membership{
		MembershipClass:    "Administrator",
		MembershipType:     "Board Member",
		Description:        "Administrative Matter Voting Token",
		TransfersPermitted: true,
	}
	instrumentData.InstrumentPayload, err = instrumentPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	t.Logf("Instrument Permissions : 0x%x", instrumentData.InstrumentPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	// Build instrument definition transaction
	instrumentTx := wire.NewMsgTx(1)

	instrumentInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(0, script))

	instrumentItx, err := inspector.NewTransactionFromWire(ctx, instrumentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx : %v", tests.Failed, err)
	}

	err = instrumentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx)

	err = a.Trigger(ctx, "SEE", instrumentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	// Check issuer balance
	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != instrumentData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			instrumentData.AuthorizedTokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.InstrumentType != instrumentData.InstrumentType {
		t.Fatalf("\t%s\tInstrument type incorrect : %s != %s", tests.Failed, as.InstrumentType,
			instrumentData.InstrumentType)
	}

	t.Logf("\t%s\tVerified instrument type : %s", tests.Success, as.InstrumentType)

	/********************************* Attempt Second Token ***************************************/
	instrumentPayloadData.MembershipClass = "Owner"

	// Build instrument definition transaction
	instrument2Tx := wire.NewMsgTx(1)

	instrument2InputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrument2Tx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrument2InputHash, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	instrument2Tx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	instrument2Tx.TxOut = append(instrument2Tx.TxOut, wire.NewTxOut(0, script))

	instrument2Itx, err := inspector.NewTransactionFromWire(ctx, instrument2Tx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx : %v", tests.Failed, err)
	}

	err = instrument2Itx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrument2Tx)

	err = a.Trigger(ctx, "SEE", instrument2Itx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject instrument definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tDuplicate Administrative instrument definition rejected", tests.Success)

	// Check the response
	checkResponse(t, "M2")
}

func instrumentIndex(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testInstrumentType = instruments.CodeShareCommon
	testInstrumentCodes[0] = protocol.InstrumentCodeFromContract(test.ContractKey.Address, 0)

	// Create InstrumentDefinition message
	instrumentData := actions.InstrumentDefinition{
		InstrumentType:             testInstrumentType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	instrumentPayloadData := instruments.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	instrumentData.InstrumentPayload, err = instrumentPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	// Build instrument definition transaction
	instrumentTx := wire.NewMsgTx(1)

	instrumentInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(0, script))

	instrumentItx, err := inspector.NewTransactionFromWire(ctx, instrumentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx : %v", tests.Failed, err)
	}

	err = instrumentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx)

	err = a.Trigger(ctx, "SEE", instrumentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	// Check issuer balance
	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	if as.InstrumentIndex != 0 {
		t.Fatalf("\t%s\tInstrument index incorrect : %d != %d", tests.Failed, as.InstrumentIndex, 0)
	}

	t.Logf("\t%s\tVerified instrument index : %d", tests.Success, as.InstrumentIndex)

	// Create another instrument --------------------------------------------------
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100021, issuerKey.Address)

	testInstrumentCodes = append(testInstrumentCodes, protocol.InstrumentCodeFromContract(test.ContractKey.Address, 1))

	// Build instrument definition transaction
	instrumentTx = wire.NewMsgTx(1)

	instrumentInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(0, script))

	instrumentItx, err = inspector.NewTransactionFromWire(ctx, instrumentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx 2 : %v", tests.Failed, err)
	}

	err = instrumentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx 2 : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx)

	err = a.Trigger(ctx, "SEE", instrumentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition 2 : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition 2 accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	// Check issuer balance
	as, err = instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[1])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument 2 : %v", tests.Failed, err)
	}

	if as.InstrumentIndex != 1 {
		t.Fatalf("\t%s\tInstrument 2 index incorrect : %d != %d", tests.Failed, as.InstrumentIndex, 1)
	}

	t.Logf("\t%s\tVerified instrument index 2 : %d", tests.Success, as.InstrumentIndex)
}

func instrumentAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)
	mockUpInstrument(t, ctx, true, true, true, 1000, 0, &sampleInstrumentPayload, true, false, false)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100002, issuerKey.Address)

	amendmentData := actions.InstrumentModification{
		InstrumentType:     testInstrumentType,
		InstrumentCode:     testInstrumentCodes[0].Bytes(),
		InstrumentRevision: 0,
	}

	// Serialize new token quantity
	newQuantity := uint64(1200)
	var buf bytes.Buffer
	if err := bitcoin.WriteBase128VarInt(&buf, newQuantity); err != nil {
		t.Fatalf("\t%s\tFailed to serialize new quantity : %v", tests.Failed, err)
	}

	fip := permissions.FieldIndexPath{actions.InstrumentFieldAuthorizedTokenQty}
	fipBytes, _ := fip.Bytes()
	amendmentData.Amendments = append(amendmentData.Amendments, &actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           buf.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
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

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAmendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	// Check balance status
	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	if as.AuthorizedTokenQty != newQuantity {
		t.Fatalf("\t%s\tInstrument token quantity incorrect : %d != %d", tests.Failed,
			as.AuthorizedTokenQty, 1200)
	}

	t.Logf("\t%s\tVerified token quantity : %d", tests.Success, as.AuthorizedTokenQty)

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != 1200 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			1200)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)
}

func instrumentProposalAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	mockUpInstrument(t, ctx, true, true, true, 1000, 0, &sampleInstrumentPayload, false, true, true)

	fip := permissions.FieldIndexPath{actions.InstrumentFieldInstrumentPayload, instruments.ShareCommonFieldDescription}
	fipBytes, _ := fip.Bytes()
	instrumentAmendment := actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test new common shares"),
	}

	if err := mockUpInstrumentAmendmentVote(ctx, 1, 0, &instrumentAmendment); err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	if err := mockUpVoteResultTx(ctx, "A"); err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote result : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100003, issuerKey.Address)

	amendmentData := actions.InstrumentModification{
		InstrumentType:     testInstrumentType,
		InstrumentCode:     testInstrumentCodes[0].Bytes(),
		InstrumentRevision: 0,
		RefTxID:            testVoteResultTxId.Bytes(),
	}

	amendmentData.Amendments = append(amendmentData.Amendments, &instrumentAmendment)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

	amendmentInputHash := fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(amendmentInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
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

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAmendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "I2")

	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	payload, err := instruments.Deserialize([]byte(as.InstrumentType), as.InstrumentPayload)
	if err != nil {
		t.Fatalf("\t%s\tFailed to deserialize instrument payload : %v", tests.Failed, err)
	}

	sharePayload, ok := payload.(*instruments.ShareCommon)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert new payload", tests.Failed)
	}

	if sharePayload.Description != "Test new common shares" {
		t.Fatalf("\t%s\tFailed to verify new payload description : \"%s\" != \"%s\"", tests.Failed, sharePayload.Description, "Test new common shares")
	}

	t.Logf("\t%s\tVerified new payload description : %s", tests.Success, sharePayload.Description)
}

func duplicateInstrument(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I",
		1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 102001, issuerKey.Address)

	testInstrumentType = instruments.CodeShareCommon
	testInstrumentCodes = []bitcoin.Hash20{protocol.InstrumentCodeFromContract(test.ContractKey.Address, 0)}

	// Create InstrumentDefinition message
	instrumentData := actions.InstrumentDefinition{
		InstrumentType:             testInstrumentType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	instrumentPayloadData := instruments.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	instrumentData.InstrumentPayload, err = instrumentPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	t.Logf("Instrument Permissions : 0x%x", instrumentData.InstrumentPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	// Build instrument definition transaction
	instrumentTx := wire.NewMsgTx(1)

	instrumentInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx.TxIn = append(instrumentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument definition : %v", tests.Failed, err)
	}
	instrumentTx.TxOut = append(instrumentTx.TxOut, wire.NewTxOut(0, script))

	instrumentItx, err := inspector.NewTransactionFromWire(ctx, instrumentTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx : %v", tests.Failed, err)
	}

	err = instrumentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx)

	t.Logf("Instrument definition 1 tx : %s", instrumentItx.Hash.String())

	err = a.Trigger(ctx, "SEE", instrumentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition 1 accepted", tests.Success)

	// Second Instrument ********************************************************************************
	fundingTx2 := tests.MockFundingTx(ctx, test.RPCNode, 102002, issuerKey.Address)

	testInstrumentCodes = append(testInstrumentCodes,
		protocol.InstrumentCodeFromContract(test.ContractKey.Address, 1))

	// Create InstrumentDefinition message
	instrumentData2 := actions.InstrumentDefinition{
		InstrumentType:             testInstrumentType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         2000,
	}

	instrumentPayloadData2 := instruments.ShareCommon{
		Ticker:             "TST2 ",
		Description:        "Test common shares 2",
		TransfersPermitted: true,
	}
	instrumentData2.InstrumentPayload, err = instrumentPayloadData2.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	instrumentData2.InstrumentPermissions, err = permissions.Bytes()
	t.Logf("Instrument Permissions : 0x%x", instrumentData2.InstrumentPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	// Build instrument definition transaction
	instrumentTx2 := wire.NewMsgTx(1)

	instrumentInputHash2 := fundingTx2.TxHash()

	// From issuer (Note: empty sig script)
	instrumentTx2.TxIn = append(instrumentTx2.TxIn, wire.NewTxIn(wire.NewOutPoint(instrumentInputHash2, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	instrumentTx2.TxOut = append(instrumentTx2.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&instrumentData2, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument definition 2 : %v", tests.Failed, err)
	}
	instrumentTx2.TxOut = append(instrumentTx2.TxOut, wire.NewTxOut(0, script))

	instrumentItx2, err := inspector.NewTransactionFromWire(ctx, instrumentTx2, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create instrument itx 2 : %v", tests.Failed, err)
	}

	err = instrumentItx2.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote instrument itx 2 : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, instrumentTx2)

	t.Logf("Instrument definition 2 tx : %s", instrumentItx2.Hash.String())

	err = a.Trigger(ctx, "SEE", instrumentItx2)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept instrument definition 2 : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tInstrument definition 2 accepted", tests.Success)

	// Check the responses *************************************************************************
	checkResponse(t, "I2")
	checkResponse(t, "I2")

	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if len(ct.InstrumentCodes) != 2 {
		t.Fatalf("\t%s\tWrong instrument code count : %d", tests.Failed, len(ct.InstrumentCodes))
	}

	if !ct.InstrumentCodes[0].Equal(&testInstrumentCodes[0]) {
		t.Fatalf("\t%s\tWrong instrument code 1 : %s", tests.Failed, ct.InstrumentCodes[0].String())
	}

	if !ct.InstrumentCodes[1].Equal(&testInstrumentCodes[1]) {
		t.Fatalf("\t%s\tWrong instrument code 2 : %s", tests.Failed, ct.InstrumentCodes[1].String())
	}

	// Check issuer balance
	as, err := instrument.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve instrument : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testInstrumentCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != instrumentData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			instrumentData.AuthorizedTokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.InstrumentType != instrumentData.InstrumentType {
		t.Fatalf("\t%s\tInstrument type incorrect : %s != %s", tests.Failed, as.InstrumentType,
			instrumentData.InstrumentType)
	}

	t.Logf("\t%s\tVerified instrument type : %s", tests.Success, as.InstrumentType)
}

var currentTimestamp = protocol.CurrentTimestamp()

var sampleInstrumentPayload = instruments.ShareCommon{
	Ticker:             "TST  ",
	Description:        "Test common shares",
	TransfersPermitted: true,
}

var sampleInstrumentPayloadNotPermitted = instruments.ShareCommon{
	Ticker:             "TST  ",
	Description:        "Test common shares",
	TransfersPermitted: false,
}

var sampleInstrumentPayload2 = instruments.ShareCommon{
	Ticker:             "TS2  ",
	Description:        "Test common shares 2",
	TransfersPermitted: true,
}

var sampleAdminInstrumentPayload = instruments.Membership{
	MembershipClass:    "Administrator",
	Description:        "Test admin token",
	TransfersPermitted: true,
}

func mockUpInstrument(t testing.TB, ctx context.Context, transfers, enforcement, voting bool,
	quantity uint64, index uint64, payload instruments.Instrument, permitted, issuer, holder bool) {

	instrumentCode := protocol.InstrumentCodeFromContract(test.ContractKey.Address, index)
	var instrumentData = state.Instrument{
		Code:                             &instrumentCode,
		InstrumentType:                   payload.Code(),
		EnforcementOrdersPermitted:       enforcement,
		VotingRights:                     voting,
		AuthorizedTokenQty:               quantity,
		InstrumentModificationGovernance: 1,
		CreatedAt:                        protocol.CurrentTimestamp(),
		UpdatedAt:                        protocol.CurrentTimestamp(),
		AdministrationProposal:           issuer, // Issuer can update field with a proposal
		HolderProposal:                   holder, // Holder's can initiate proposals to update field
	}

	testInstrumentType = payload.Code()
	for uint64(len(testInstrumentCodes)) <= index {
		testInstrumentCodes = append(testInstrumentCodes, bitcoin.Hash20{})
	}
	testInstrumentCodes[index] = *instrumentData.Code
	testTokenQty = quantity

	var err error
	instrumentData.InstrumentPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false},
		},
	}

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        instrumentData.CreatedAt,
		UpdatedAt:        instrumentData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address,
		&testInstrumentCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = instrument.Save(ctx, test.MasterDB, test.ContractKey.Address, &instrumentData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save instrument : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.InstrumentCodes = append(ct.InstrumentCodes, instrumentData.Code)

	if payload.Code() == instruments.CodeMembership {
		membership, _ := payload.(*instruments.Membership)
		if membership.MembershipClass == "Owner" || membership.MembershipClass == "Administrator" {
			ct.AdminMemberInstrument = *instrumentData.Code
		}
	}

	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpInstrument2(t testing.TB, ctx context.Context, transfers, enforcement, voting bool,
	quantity uint64, payload instruments.Instrument, permitted, issuer, holder bool) {

	instrumentCode := protocol.InstrumentCodeFromContract(test.Contract2Key.Address, 0)
	var instrumentData = state.Instrument{
		Code:                       &instrumentCode,
		InstrumentType:             payload.Code(),
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		AuthorizedTokenQty:         quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testInstrument2Type = payload.Code()
	testInstrument2Code = *instrumentData.Code
	testToken2Qty = quantity

	var err error
	instrumentData.InstrumentPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        instrumentData.CreatedAt,
		UpdatedAt:        instrumentData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testInstrumentCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = instrument.Save(ctx, test.MasterDB, test.Contract2Key.Address, &instrumentData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save instrument : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.Contract2Key.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.InstrumentCodes = append(ct.InstrumentCodes, instrumentData.Code)
	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpOtherInstrument(t testing.TB, ctx context.Context, key *wallet.Key, transfers, enforcement,
	voting bool, quantity uint64, payload instruments.Instrument, permitted, issuer, holder bool) {

	instrumentCode := protocol.InstrumentCodeFromContract(key.Address, 0)
	var instrumentData = state.Instrument{
		Code:                       &instrumentCode,
		InstrumentType:             payload.Code(),
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		AuthorizedTokenQty:         quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testInstrument2Type = payload.Code()
	testInstrument2Code = *instrumentData.Code
	testToken2Qty = quantity

	var err error
	instrumentData.InstrumentPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument payload : %v", tests.Failed, err)
	}

	// Define permissions for instrument fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	instrumentData.InstrumentPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize instrument permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        instrumentData.CreatedAt,
		UpdatedAt:        instrumentData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testInstrumentCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = instrument.Save(ctx, test.MasterDB, key.Address, &instrumentData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save instrument : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, key.Address, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.InstrumentCodes = append(ct.InstrumentCodes, instrumentData.Code)
	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpHolding(t testing.TB, ctx context.Context, address bitcoin.RawAddress, quantity uint64) {
	mockUpInstrumentHolding(t, ctx, address, testInstrumentCodes[0], quantity)
}

func mockUpInstrumentHolding(t testing.TB, ctx context.Context, address bitcoin.RawAddress,
	instrumentCode bitcoin.Hash20, quantity uint64) {

	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address, &instrumentCode, &h)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)
}

func mockUpHolding2(t testing.TB, ctx context.Context, address bitcoin.RawAddress, quantity uint64) {
	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testInstrument2Code, &h)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)
}

func mockUpOtherHolding(t testing.TB, ctx context.Context, key *wallet.Key, address bitcoin.RawAddress,
	quantity uint64) {

	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testInstrument2Code, &h)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)
}
