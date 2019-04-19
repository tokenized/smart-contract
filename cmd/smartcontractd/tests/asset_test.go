package tests

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestAssets is the entry point for testing asset based functions.
func TestAssets(t *testing.T) {
	defer tests.Recover(t)

	t.Run("createAsset", createAsset)
	t.Run("assetAmendment", assetAmendment)
	t.Run("proposalAmendment", proposalAmendment)
}

func createAsset(t *testing.T) {
	ctx := test.Context

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, test.MasterDB, contractPKH)

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100001, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	testAssetCode = *protocol.AssetCodeFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})

	// Create AssetDefinition message
	assetData := protocol.AssetDefinition{
		AssetType:                  protocol.CodeShareCommon,
		AssetCode:                  testAssetCode,
		TransfersPermitted:         true,
		EnforcementOrdersPermitted: true,
		VotingRights:               true,
		TokenQty:                   1000,
	}

	assetPayloadData := protocol.ShareCommon{
		Ticker:      "TST",
		Description: "Test common shares",
	}
	assetData.AssetPayload, err = assetPayloadData.Serialize()
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset payload : %v", tests.Failed, err)
	}

	// Define permissions for asset fields
	permissions := make([]protocol.Permission, 13)
	for i, _ := range permissions {
		permissions[i].Permitted = true       // Issuer can update field without proposal
		permissions[i].IssuerProposal = false // Issuer can update field with a proposal
		permissions[i].HolderProposal = false // Holder's can initiate proposals to update field

		permissions[i].VotingSystemsAllowed = make([]bool, len(ct.VotingSystems))
		permissions[i].VotingSystemsAllowed[0] = true // Enable this voting system for proposals on this field.
	}

	assetData.AssetAuthFlags, err = protocol.WriteAuthFlags(permissions)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize asset auth flags : %v", tests.Failed, err)
	}

	// Build asset definition transaction
	assetTx := wire.NewMsgTx(2)

	var assetInputHash chainhash.Hash
	assetInputHash = fundingTx.TxHash()

	// From issuer (Note: empty sig script)
	assetTx.TxIn = append(assetTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&assetInputHash, 0), make([]byte, 130)))

	// To contract
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(100000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&assetData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize offer : %v", tests.Failed, err)
	}
	assetTx.TxOut = append(assetTx.TxOut, wire.NewTxOut(0, script))

	assetItx, err := inspector.NewTransactionFromWire(ctx, assetTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create asset itx : %v", tests.Failed, err)
	}

	err = assetItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote asset itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, assetTx)

	err = a.Trigger(ctx, protomux.SEE, assetItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept asset definition : %v", tests.Failed, err)
	}

	t.Logf("Asset definition accepted")

	// Check the response
	checkResponse(t, "A2")

	// Check issuer balance
	contractPKH = protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &assetData.AssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != assetData.TokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, assetData.TokenQty)
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
}

func assetAmendment(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100007, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	amendmentData := protocol.AssetModification{
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     testAssetCode,
		AssetRevision: 0,
	}

	// Serialize new token quantity
	newQuantity := uint64(1200)
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, &newQuantity)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize new quantity : %v", tests.Failed, err)
	}

	amendmentData.Amendments = append(amendmentData.Amendments, protocol.Amendment{
		FieldIndex: 11, // Token quantity
		Data:       buf.Bytes(),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	var amendmentInputHash chainhash.Hash
	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&amendmentData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, amendmentTx)

	err = a.Trigger(ctx, protomux.SEE, amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("Amendment accepted")

	// Check the response
	checkResponse(t, "A2")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	if as.TokenQty != 1200 {
		t.Fatalf("\t%s\tAsset token quantity incorrect : %d != %d", tests.Failed, as.TokenQty, 1200)
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != 1000 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, 1000)
	}

	t.Logf("Verified issuer balance : %d", issuerBalance)
}

func proposalAmendment(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100007, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	amendmentData := protocol.ContractAmendment{
		ContractRevision: 0,
		RefTxID:          testVoteResultTxId,
	}

	amendmentData.Amendments = append(amendmentData.Amendments, protocol.Amendment{
		FieldIndex: 0,
		Data:       []byte("Test Name 2"),
	})

	// Build amendment transaction
	amendmentTx := wire.NewMsgTx(2)

	var amendmentInputHash chainhash.Hash
	amendmentInputHash = fundingTx.TxHash()

	// From issuer
	amendmentTx.TxIn = append(amendmentTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&amendmentInputHash, 0), make([]byte, 130)))

	// To contract
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&amendmentData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize amendment : %v", tests.Failed, err)
	}
	amendmentTx.TxOut = append(amendmentTx.TxOut, wire.NewTxOut(0, script))

	amendmentItx, err := inspector.NewTransactionFromWire(ctx, amendmentTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create amendment itx : %v", tests.Failed, err)
	}

	err = amendmentItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote amendment itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, amendmentTx)

	err = a.Trigger(ctx, protomux.SEE, amendmentItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept amendment : %v", tests.Failed, err)
	}

	t.Logf("Amendment accepted")

	// Check the response
	checkResponse(t, "C2")
}
