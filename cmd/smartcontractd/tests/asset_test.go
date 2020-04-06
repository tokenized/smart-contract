package tests

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestAssets is the entry point for testing asset based functions.
func TestAssets(t *testing.T) {
	defer tests.Recover(t)

	t.Run("create", createAsset)
	t.Run("adminMemberAsset", adminMemberAsset)
	t.Run("index", assetIndex)
	t.Run("amendment", assetAmendment)
	t.Run("proposalAmendment", assetProposalAmendment)
}

func createAsset(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testAssetType = assets.CodeShareCommon
	testAssetCodes = []protocol.AssetCode{*protocol.AssetCodeFromContract(test.ContractKey.Address, 0)}

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  testAssetType,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		TokenQty:                   1000,
	}

	assetPayloadData := assets.ShareCommon{
		Ticker:      "TST  ",
		Description: "Test common shares",
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	assetData.AssetPermissions, err = permissions.Bytes()
	t.Logf("Asset Permissions : 0x%x", assetData.AssetPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	// Build asset definition transaction
	assetTx := wire.NewMsgTx(1)

	assetInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(assetInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx : %v", tests.Failed, err)
	}

	err = assetItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, assetTx)

	err = a.Trigger(ctx, "SEE", assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "A2")

	// Check issuer balance
	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != assetData.TokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			assetData.TokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.AssetType != assetData.AssetType {
		t.Fatalf("\t%s\tAsset type incorrect : %s != %s", tests.Failed, as.AssetType,
			assetData.AssetType)
	}

	t.Logf("\t%s\tVerified asset type : %s", tests.Success, as.AssetType)
}

func adminMemberAsset(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testAssetType = assets.CodeShareCommon
	testAssetCodes[0] = *protocol.AssetCodeFromContract(test.ContractKey.Address, 0)

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  "MEM",
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		TokenQty:                   5,
	}

	assetPayloadData := assets.Membership{
		MembershipClass: "Administrator",
		MembershipType:  "Board Member",
		Description:     "Administrative Matter Voting Token",
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	assetData.AssetPermissions, err = permissions.Bytes()
	t.Logf("Asset Permissions : 0x%x", assetData.AssetPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	// Build asset definition transaction
	assetTx := wire.NewMsgTx(1)

	assetInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(assetInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx : %v", tests.Failed, err)
	}

	err = assetItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, assetTx)

	err = a.Trigger(ctx, "SEE", assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "A2")

	// Check issuer balance
	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if h.PendingBalance != assetData.TokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			assetData.TokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.AssetType != assetData.AssetType {
		t.Fatalf("\t%s\tAsset type incorrect : %s != %s", tests.Failed, as.AssetType,
			assetData.AssetType)
	}

	t.Logf("\t%s\tVerified asset type : %s", tests.Success, as.AssetType)

	/********************************* Attempt Second Token ***************************************/
	assetPayloadData.MembershipClass = "Owner"

	// Build asset definition transaction
	asset2Tx := wire.NewMsgTx(1)

	asset2InputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	asset2Tx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(asset2InputHash, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	asset2Tx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	asset2Tx.TxOut = append(asset2Tx.TxOut, wire.NewTxOut(0, script))

	asset2Itx, err := inspector.NewTransactionFromWire(ctx, asset2Tx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx : %v", tests.Failed, err)
	}

	err = asset2Itx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, asset2Tx)

	err = a.Trigger(ctx, "SEE", asset2Itx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject asset definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tDuplicate Administrative asset definition rejected", tests.Success)

	// Check the response
	checkResponse(t, "M2")
}

func assetIndex(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testAssetType = assets.CodeShareCommon
	testAssetCodes[0] = *protocol.AssetCodeFromContract(test.ContractKey.Address, 0)

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  testAssetType,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		TokenQty:                   1000,
	}

	assetPayloadData := assets.ShareCommon{
		Ticker:      "TST  ",
		Description: "Test common shares",
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              true,  // Issuer can update field without proposal
			AdministrationProposal: false, // Issuer can update field with a proposal
			HolderProposal:         false, // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
		},
	}
	permissions[0].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
	permissions[0].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	// Build asset definition transaction
	assetTx := wire.NewMsgTx(1)

	assetInputHash := fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(assetInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx : %v", tests.Failed, err)
	}

	err = assetItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, assetTx)

	err = a.Trigger(ctx, "SEE", assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition accepted", tests.Success)

	// Check the response
	checkResponse(t, "A2")

	// Check issuer balance
	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	if as.AssetIndex != 0 {
		t.Fatalf("\t%s\tAsset index incorrect : %d != %d", tests.Failed, as.AssetIndex, 0)
	}

	t.Logf("\t%s\tVerified asset index : %d", tests.Success, as.AssetIndex)

	// Create another asset --------------------------------------------------
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100021, issuerKey.Address)

	testAssetCodes = append(testAssetCodes, *protocol.AssetCodeFromContract(test.ContractKey.Address, 1))

	// Build asset definition transaction
	assetTx = wire.NewMsgTx(1)

	assetInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(assetInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err = inspector.NewTransactionFromWire(ctx, assetTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx 2 : %v", tests.Failed, err)
	}

	err = assetItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx 2 : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, assetTx)

	err = a.Trigger(ctx, "SEE", assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition 2 : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition 2 accepted", tests.Success)

	// Check the response
	checkResponse(t, "A2")

	// Check issuer balance
	as, err = asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[1])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset 2 : %v", tests.Failed, err)
	}

	if as.AssetIndex != 1 {
		t.Fatalf("\t%s\tAsset 2 index incorrect : %d != %d", tests.Failed, as.AssetIndex, 1)
	}

	t.Logf("\t%s\tVerified asset index 2 : %d", tests.Success, as.AssetIndex)
}

func assetAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100002, issuerKey.Address)

	amendmentData := actions.AssetModification{
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
		AssetRevision: 0,
	}

	// Serialize new token quantity
	newQuantity := uint64(1200)
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, &newQuantity)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize new quantity : %v", tests.Failed, err)
	}

	fip := actions.FieldIndexPath{actions.AssetFieldTokenQty}
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
	checkResponse(t, "A2")

	// Check balance status
	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	if as.TokenQty != newQuantity {
		t.Fatalf("\t%s\tAsset token quantity incorrect : %d != %d", tests.Failed, as.TokenQty, 1200)
	}

	t.Logf("\t%s\tVerified token quantity : %d", tests.Success, as.TokenQty)

	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
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

func assetProposalAmendment(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, 0, &sampleAssetPayload, false, true, true)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}

	fip := actions.FieldIndexPath{actions.AssetFieldAssetPayload, assets.ShareCommonFieldDescription}
	fipBytes, _ := fip.Bytes()
	assetAmendment := actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test new common shares"),
	}
	err = mockUpAssetAmendmentVote(ctx, 1, 0, &assetAmendment)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	err = mockUpVoteResultTx(ctx, "A")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote result : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100003, issuerKey.Address)

	amendmentData := actions.AssetModification{
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
		AssetRevision: 0,
		RefTxID:       testVoteResultTxId.Bytes(),
	}

	amendmentData.Amendments = append(amendmentData.Amendments, &assetAmendment)

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(1)

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

	err = a.Trigger(ctx, "SEE", amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAmendment accepted", tests.Success)

	// Check the response
	checkResponse(t, "A2")

	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	payload, err := assets.Deserialize([]byte(as.AssetType), as.AssetPayload)
	if err != nil {
		t.Fatalf("\t%s\tFailed to deserialize asset payload : %v", tests.Failed, err)
	}

	sharePayload, ok := payload.(*assets.ShareCommon)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert new payload", tests.Failed)
	}

	if sharePayload.Description != "Test new common shares" {
		t.Fatalf("\t%s\tFailed to verify new payload description : \"%s\" != \"%s\"", tests.Failed, sharePayload.Description, "Test new common shares")
	}

	t.Logf("\t%s\tVerified new payload description : %s", tests.Success, sharePayload.Description)
}

var currentTimestamp = protocol.CurrentTimestamp()

var sampleAssetPayload = assets.ShareCommon{
	Ticker:      "TST  ",
	Description: "Test common shares",
}

var sampleAssetPayload2 = assets.ShareCommon{
	Ticker:      "TS2  ",
	Description: "Test common shares 2",
}

var sampleAdminAssetPayload = assets.Membership{
	MembershipClass: "Administrator",
	Description:     "Test admin token",
}

func mockUpAsset(ctx context.Context, transfers, enforcement, voting bool, quantity uint64,
	index uint64, payload assets.Asset, permitted, issuer, holder bool) error {

	var assetData = state.Asset{
		Code:                        protocol.AssetCodeFromContract(test.ContractKey.Address, index),
		AssetType:                   payload.Code(),
		TransfersPermitted:          transfers,
		EnforcementOrdersPermitted:  enforcement,
		VotingRights:                voting,
		TokenQty:                    quantity,
		AssetModificationGovernance: 1,
		CreatedAt:                   protocol.CurrentTimestamp(),
		UpdatedAt:                   protocol.CurrentTimestamp(),
	}

	testAssetType = payload.Code()
	for uint64(len(testAssetCodes)) <= index {
		testAssetCodes = append(testAssetCodes, protocol.AssetCode{})
	}
	testAssetCodes[index] = *assetData.Code
	testTokenQty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		return err
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false},
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0], &issuerHolding)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, test.ContractKey.Address, &assetData)
	if err != nil {
		return err
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address)
	if err != nil {
		return err
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)

	if payload.Code() == "MEM" {
		membership, _ := payload.(*assets.Membership)
		if membership.MembershipClass == "Owner" || membership.MembershipClass == "Administrator" {
			ct.AdminMemberAsset = *assetData.Code
		}
	}

	return contract.Save(ctx, test.MasterDB, ct)
}

func mockUpAsset2(ctx context.Context, transfers, enforcement, voting bool, quantity uint64,
	payload assets.Asset, permitted, issuer, holder bool) error {

	var assetData = state.Asset{
		Code:                       protocol.AssetCodeFromContract(test.Contract2Key.Address, 0),
		AssetType:                  payload.Code(),
		TransfersPermitted:         transfers,
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		TokenQty:                   quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testAsset2Type = payload.Code()
	testAsset2Code = *assetData.Code
	testToken2Qty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		return err
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testAssetCodes[0], &issuerHolding)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, test.Contract2Key.Address, &assetData)
	if err != nil {
		return err
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.Contract2Key.Address)
	if err != nil {
		return err
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)
	return contract.Save(ctx, test.MasterDB, ct)
}

func mockUpOtherAsset(ctx context.Context, key *wallet.Key, transfers, enforcement, voting bool,
	quantity uint64, payload assets.Asset, permitted, issuer, holder bool) error {

	var assetData = state.Asset{
		Code:                       protocol.AssetCodeFromContract(key.Address, 0),
		AssetType:                  payload.Code(),
		TransfersPermitted:         transfers,
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		TokenQty:                   quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testAsset2Type = payload.Code()
	testAsset2Code = *assetData.Code
	testToken2Qty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		return err
	}

	// Define permissions for asset fields
	permissions := actions.Permissions{
		actions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		return err
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testAssetCodes[0], &issuerHolding)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, key.Address, &assetData)
	if err != nil {
		return err
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, key.Address)
	if err != nil {
		return err
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)
	return contract.Save(ctx, test.MasterDB, ct)
}

func mockUpHolding(ctx context.Context, address bitcoin.RawAddress, quantity uint64) error {
	return mockUpAssetHolding(ctx, address, testAssetCodes[0], quantity)
}

func mockUpAssetHolding(ctx context.Context, address bitcoin.RawAddress, assetCode protocol.AssetCode,
	quantity uint64) error {

	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address, &assetCode, &h)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)
	return nil
}

func mockUpHolding2(ctx context.Context, address bitcoin.RawAddress, quantity uint64) error {
	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testAsset2Code, &h)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)
	return nil
}

func mockUpOtherHolding(ctx context.Context, key *wallet.Key, address bitcoin.RawAddress,
	quantity uint64) error {

	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[protocol.TxId]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testAsset2Code, &h)
	if err != nil {
		return err
	}
	test.HoldingsChannel.Add(cacheItem)
	return nil
}
