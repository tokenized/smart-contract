package tests

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestEnforcement is the entry point for testing enforcement functions.
func TestEnforcement(t *testing.T) {
	defer tests.Recover(t)

	t.Run("freezeOrder", freezeOrder)
	t.Run("thawOrder", thawOrder)
	t.Run("confiscateOrder", confiscateOrder)
	t.Run("reconcileOrder", reconcileOrder)
}

func freezeOrder(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100008, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, orderTx)

	err = a.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("Freeze order accepted")

	if len(responses) > 0 {
		hash := responses[0].TxHash()
		testFreezeTxId = *protocol.TxIdFromBytes(hash[:])
	}

	// Check the response
	checkResponse(t, "E2")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if asset.CheckBalanceFrozen(ctx, as, userPKH, 100, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance too high", tests.Failed)
	}

	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 50, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance not high enough", tests.Failed)
	}
}

func thawOrder(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionThaw,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        testAssetCode,
		FreezeTxId:       testFreezeTxId,
		Message:          "Court order released",
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, orderTx)

	err = a.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("Thaw order accepted")

	// Check the response
	checkResponse(t, "E3")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 250, v.Now) {
		t.Fatalf("\t%s\tUser balance not unfrozen", tests.Failed)
	}
}

func confiscateOrder(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionConfiscation,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        testAssetCode,
		DepositAddress:   *issuerPKH,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(1500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, orderTx)

	err = a.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("Confiscate order accepted")

	// Check the response
	checkResponse(t, "E4")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != testTokenQty-200 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty-200)
	}

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 200 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, 200)
	}
}

func reconcileOrder(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100009, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionReconciliation,
		AssetType:        protocol.CodeShareCommon,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	orderData.BitcoinDispersions = append(orderData.BitcoinDispersions, protocol.QuantityIndex{Index: 0, Quantity: 75000})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	var orderInputHash chainhash.Hash
	orderInputHash = fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(751500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, orderTx)

	err = a.Trigger(ctx, protomux.SEE, orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("Reconcile order accepted")

	if len(responses) < 1 {
		t.Fatalf("\t%s\tNo response for reconcile", tests.Failed)
	}

	// Check for bitcoin dispersion to user
	found := false
	for _, output := range responses[0].TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}
		if bytes.Equal(pkh, userPKH.Bytes()) && output.Value == 75000 {
			t.Logf("Found reconcile bitcoin dispersion")
			found = true
		}
	}

	if !found {
		t.Fatalf("\t%s\tFailed to find bitcoin dispersion", tests.Failed)
	}

	// Check the response
	checkResponse(t, "E5")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != testTokenQty-200 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty-200)
	}
	t.Logf("Verified issuer balance : %d", issuerBalance)

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 150 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, 150)
	}
	t.Logf("Verified user balance : %d", userBalance)
}
