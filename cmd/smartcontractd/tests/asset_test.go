package tests

import (
	"bytes"
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/permissions"
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
	t.Run("duplicateAsset", duplicateAsset)
}

func createAsset(t *testing.T) {
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

	testAssetType = assets.CodeShareCommon
	testAssetCodes = []bitcoin.Hash20{protocol.AssetCodeFromContract(test.ContractKey.Address, 0)}

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  testAssetType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	assetPayloadData := assets.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
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
		t.Fatalf("\t%s\tFailed to serialize asset definition : %v", tests.Failed, err)
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
	if h.PendingBalance != assetData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			assetData.AuthorizedTokenQty)
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
	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testAssetType = assets.CodeShareCommon
	testAssetCodes[0] = protocol.AssetCodeFromContract(test.ContractKey.Address, 0)

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  assets.CodeMembership,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         5,
	}

	assetPayloadData := assets.Membership{
		MembershipClass:    "Administrator",
		MembershipType:     "Board Member",
		Description:        "Administrative Matter Voting Token",
		TransfersPermitted: true,
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
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
	if h.PendingBalance != assetData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			assetData.AuthorizedTokenQty)
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

	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)

	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100001, issuerKey.Address)

	testAssetType = assets.CodeShareCommon
	testAssetCodes[0] = protocol.AssetCodeFromContract(test.ContractKey.Address, 0)

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  testAssetType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	assetPayloadData := assets.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
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

	testAssetCodes = append(testAssetCodes, protocol.AssetCodeFromContract(test.ContractKey.Address, 1))

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
	mockUpContract(t, ctx, "Test Contract", "I", 1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100002, issuerKey.Address)

	amendmentData := actions.AssetModification{
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
		AssetRevision: 0,
	}

	// Serialize new token quantity
	newQuantity := uint64(1200)
	var buf bytes.Buffer
	if err := bitcoin.WriteBase128VarInt(&buf, newQuantity); err != nil {
		t.Fatalf("\t%s\tFailed to serialize new quantity : %v", tests.Failed, err)
	}

	fip := permissions.FieldIndexPath{actions.AssetFieldAuthorizedTokenQty}
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
	checkResponse(t, "A2")

	// Check balance status
	as, err := asset.Retrieve(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0])
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	if as.AuthorizedTokenQty != newQuantity {
		t.Fatalf("\t%s\tAsset token quantity incorrect : %d != %d", tests.Failed,
			as.AuthorizedTokenQty, 1200)
	}

	t.Logf("\t%s\tVerified token quantity : %d", tests.Success, as.AuthorizedTokenQty)

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
	mockUpContract(t, ctx, "Test Contract", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, false, true, true)

	fip := permissions.FieldIndexPath{actions.AssetFieldAssetPayload, assets.ShareCommonFieldDescription}
	fipBytes, _ := fip.Bytes()
	assetAmendment := actions.AmendmentField{
		FieldIndexPath: fipBytes,
		Data:           []byte("Test new common shares"),
	}

	if err := mockUpAssetAmendmentVote(ctx, 1, 0, &assetAmendment); err != nil {
		t.Fatalf("\t%s\tFailed to mock up vote : %v", tests.Failed, err)
	}

	if err := mockUpVoteResultTx(ctx, "A"); err != nil {
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

func duplicateAsset(t *testing.T) {
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

	testAssetType = assets.CodeShareCommon
	testAssetCodes = []bitcoin.Hash20{protocol.AssetCodeFromContract(test.ContractKey.Address, 0)}

	// Create AssetDefinition message
	assetData := actions.AssetDefinition{
		AssetType:                  testAssetType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         1000,
	}

	assetPayloadData := assets.ShareCommon{
		Ticker:             "TST  ",
		Description:        "Test common shares",
		TransfersPermitted: true,
	}
	assetData.AssetPayload, err = assetPayloadData.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
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
		t.Fatalf("\t%s\tFailed to serialize asset definition : %v", tests.Failed, err)
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

	t.Logf("Asset definition 1 tx : %s", assetItx.Hash.String())

	err = a.Trigger(ctx, "SEE", assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition 1 accepted", tests.Success)

	// Second Asset ********************************************************************************
	fundingTx2 := tests.MockFundingTx(ctx, test.RPCNode, 102002, issuerKey.Address)

	testAssetCodes = append(testAssetCodes,
		protocol.AssetCodeFromContract(test.ContractKey.Address, 1))

	// Create AssetDefinition message
	assetData2 := actions.AssetDefinition{
		AssetType:                  testAssetType,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		AuthorizedTokenQty:         2000,
	}

	assetPayloadData2 := assets.ShareCommon{
		Ticker:             "TST2 ",
		Description:        "Test common shares 2",
		TransfersPermitted: true,
	}
	assetData2.AssetPayload, err = assetPayloadData2.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	assetData2.AssetPermissions, err = permissions.Bytes()
	t.Logf("Asset Permissions : 0x%x", assetData2.AssetPermissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	// Build asset definition transaction
	assetTx2 := wire.NewMsgTx(1)

	assetInputHash2 := fundingTx2.TxHash()

	// From issuer (Note: empty sig script)
	assetTx2.TxIn = append(assetTx2.TxIn, wire.NewTxIn(wire.NewOutPoint(assetInputHash2, 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	assetTx2.TxOut = append(assetTx2.TxOut, wire.NewTxOut(100000, script))

	// Data output
	script, err = protocol.Serialize(&assetData2, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset definition 2 : %v", tests.Failed, err)
	}
	assetTx2.TxOut = append(assetTx2.TxOut, wire.NewTxOut(0, script))

	assetItx2, err := inspector.NewTransactionFromWire(ctx, assetTx2, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx 2 : %v", tests.Failed, err)
	}

	err = assetItx2.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx 2 : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, assetTx2)

	t.Logf("Asset definition 2 tx : %s", assetItx2.Hash.String())

	err = a.Trigger(ctx, "SEE", assetItx2)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition 2 : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tAsset definition 2 accepted", tests.Success)

	// Check the responses *************************************************************************
	checkResponse(t, "A2")
	checkResponse(t, "A2")

	ct, err = contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	if len(ct.AssetCodes) != 2 {
		t.Fatalf("\t%s\tWrong asset code count : %d", tests.Failed, len(ct.AssetCodes))
	}

	if !ct.AssetCodes[0].Equal(&testAssetCodes[0]) {
		t.Fatalf("\t%s\tWrong asset code 1 : %s", tests.Failed, ct.AssetCodes[0].String())
	}

	if !ct.AssetCodes[1].Equal(&testAssetCodes[1]) {
		t.Fatalf("\t%s\tWrong asset code 2 : %s", tests.Failed, ct.AssetCodes[1].String())
	}

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
	if h.PendingBalance != assetData.AuthorizedTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, h.PendingBalance,
			assetData.AuthorizedTokenQty)
	}

	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, h.PendingBalance)

	if as.AssetType != assetData.AssetType {
		t.Fatalf("\t%s\tAsset type incorrect : %s != %s", tests.Failed, as.AssetType,
			assetData.AssetType)
	}

	t.Logf("\t%s\tVerified asset type : %s", tests.Success, as.AssetType)
}

var currentTimestamp = protocol.CurrentTimestamp()

var sampleAssetPayload = assets.ShareCommon{
	Ticker:             "TST  ",
	Description:        "Test common shares",
	TransfersPermitted: true,
}

var sampleAssetPayloadNotPermitted = assets.ShareCommon{
	Ticker:             "TST  ",
	Description:        "Test common shares",
	TransfersPermitted: false,
}

var sampleAssetPayload2 = assets.ShareCommon{
	Ticker:             "TS2  ",
	Description:        "Test common shares 2",
	TransfersPermitted: true,
}

var sampleAdminAssetPayload = assets.Membership{
	MembershipClass:    "Administrator",
	Description:        "Test admin token",
	TransfersPermitted: true,
}

func mockUpAsset(t testing.TB, ctx context.Context, transfers, enforcement, voting bool,
	quantity uint64, index uint64, payload assets.Asset, permitted, issuer, holder bool) {

	assetCode := protocol.AssetCodeFromContract(test.ContractKey.Address, index)
	var assetData = state.Asset{
		Code:                        &assetCode,
		AssetType:                   payload.Code(),
		EnforcementOrdersPermitted:  enforcement,
		VotingRights:                voting,
		AuthorizedTokenQty:          quantity,
		AssetModificationGovernance: 1,
		CreatedAt:                   protocol.CurrentTimestamp(),
		UpdatedAt:                   protocol.CurrentTimestamp(),
	}

	testAssetType = payload.Code()
	for uint64(len(testAssetCodes)) <= index {
		testAssetCodes = append(testAssetCodes, bitcoin.Hash20{})
	}
	testAssetCodes[index] = *assetData.Code
	testTokenQty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false},
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, test.ContractKey.Address, &assetData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save asset : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.ContractKey.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)

	if payload.Code() == assets.CodeMembership {
		membership, _ := payload.(*assets.Membership)
		if membership.MembershipClass == "Owner" || membership.MembershipClass == "Administrator" {
			ct.AdminMemberAsset = *assetData.Code
		}
	}

	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpAsset2(t testing.TB, ctx context.Context, transfers, enforcement, voting bool,
	quantity uint64, payload assets.Asset, permitted, issuer, holder bool) {

	assetCode := protocol.AssetCodeFromContract(test.Contract2Key.Address, 0)
	var assetData = state.Asset{
		Code:                       &assetCode,
		AssetType:                  payload.Code(),
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		AuthorizedTokenQty:         quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testAsset2Type = payload.Code()
	testAsset2Code = *assetData.Code
	testToken2Qty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testAssetCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, test.Contract2Key.Address, &assetData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save asset : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, test.Contract2Key.Address,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)
	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpOtherAsset(t testing.TB, ctx context.Context, key *wallet.Key, transfers, enforcement,
	voting bool, quantity uint64, payload assets.Asset, permitted, issuer, holder bool) {

	assetCode := protocol.AssetCodeFromContract(key.Address, 0)
	var assetData = state.Asset{
		Code:                       &assetCode,
		AssetType:                  payload.Code(),
		EnforcementOrdersPermitted: enforcement,
		VotingRights:               voting,
		AuthorizedTokenQty:         quantity,
		CreatedAt:                  protocol.CurrentTimestamp(),
		UpdatedAt:                  protocol.CurrentTimestamp(),
	}

	testAsset2Type = payload.Code()
	testAsset2Code = *assetData.Code
	testToken2Qty = quantity

	var err error
	assetData.AssetPayload, err = payload.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := permissions.Permissions{
		permissions.Permission{
			Permitted:              permitted, // Issuer can update field without proposal
			AdministrationProposal: issuer,    // Issuer can update field with a proposal
			HolderProposal:         holder,    // Holder's can initiate proposals to update field
			AdministrativeMatter:   false,
			VotingSystemsAllowed:   []bool{true, false}, // Enable this voting system for proposals on this field.
		},
	}

	assetData.AssetPermissions, err = permissions.Bytes()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset permissions : %v", tests.Failed, err)
	}

	issuerHolding := state.Holding{
		Address:          issuerKey.Address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        assetData.CreatedAt,
		UpdatedAt:        assetData.UpdatedAt,
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testAssetCodes[0], &issuerHolding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)

	err = asset.Save(ctx, test.MasterDB, key.Address, &assetData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save asset : %v", tests.Failed, err)
	}

	// Add to contract
	ct, err := contract.Retrieve(ctx, test.MasterDB, key.Address, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve contract : %v", tests.Failed, err)
	}

	ct.AssetCodes = append(ct.AssetCodes, assetData.Code)
	if err := contract.Save(ctx, test.MasterDB, ct, test.NodeConfig.IsTest); err != nil {
		t.Fatalf("\t%s\tFailed to save contract : %v", tests.Failed, err)
	}
}

func mockUpHolding(t testing.TB, ctx context.Context, address bitcoin.RawAddress, quantity uint64) {
	mockUpAssetHolding(t, ctx, address, testAssetCodes[0], quantity)
}

func mockUpAssetHolding(t testing.TB, ctx context.Context, address bitcoin.RawAddress,
	assetCode bitcoin.Hash20, quantity uint64) {

	h := state.Holding{
		Address:          address,
		PendingBalance:   quantity,
		FinalizedBalance: quantity,
		CreatedAt:        protocol.CurrentTimestamp(),
		UpdatedAt:        protocol.CurrentTimestamp(),
		HoldingStatuses:  make(map[bitcoin.Hash32]*state.HoldingStatus),
	}
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.ContractKey.Address, &assetCode, &h)
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
	cacheItem, err := holdings.Save(ctx, test.MasterDB, test.Contract2Key.Address, &testAsset2Code, &h)
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
	cacheItem, err := holdings.Save(ctx, test.MasterDB, key.Address, &testAsset2Code, &h)
	if err != nil {
		t.Fatalf("\t%s\tFailed to save holdings : %v", tests.Failed, err)
	}
	test.HoldingsChannel.Add(cacheItem)
}
