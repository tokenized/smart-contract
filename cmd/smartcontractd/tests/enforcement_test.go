package tests

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestEnforcement is the entry point for testing enforcement functions.
func TestEnforcement(t *testing.T) {
	defer tests.Recover(t)

	t.Run("freeze", freezeOrder)
	t.Run("authority", freezeAuthorityOrder)
	t.Run("thaw", thawOrder)
	t.Run("confiscate", confiscateOrder)
	t.Run("reconcile", reconcileOrder)
}

func freezeOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address.ScriptAddress())

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, orderTx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tFreeze order accepted", tests.Success)

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
	if asset.CheckBalanceFrozen(ctx, as, userPKH, 101, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance too high", tests.Failed)
	}

	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 100, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance not high enough", tests.Failed)
	}
}

func freezeAuthorityOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address.ScriptAddress())

	orderData := protocol.Order{
		ComplianceAction:   protocol.ComplianceActionFreeze,
		AssetType:          testAssetType,
		AssetCode:          testAssetCode,
		Message:            "Court order",
		AuthorityIncluded:  true,
		AuthorityName:      "District Court #345",
		AuthorityPublicKey: authorityKey.PublicKey.SerializeCompressed(),
		SignatureAlgorithm: 1,
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity: 200,
	})

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	sigHash, err := protocol.OrderAuthoritySigHash(ctx, contractPKH, &orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed generate authority signature hash : %v", tests.Failed, err)
	}

	signature, err := authorityKey.PrivateKey.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to sign authority sig hash : %v", tests.Failed, err)
	}

	orderData.OrderSignature = signature.Serialize()

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, orderTx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tFreeze order with authority accepted", tests.Success)

	// Check the response
	checkResponse(t, "E2")

	// Check balance status
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	if asset.CheckBalanceFrozen(ctx, as, userPKH, 101, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance too high", tests.Failed)
	}

	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 100, v.Now) {
		t.Fatalf("\t%s\tUser unfrozen balance not high enough", tests.Failed)
	}
}

func thawOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	freezeTxId, err := mockUpFreeze(ctx, t, userKey.Address.ScriptAddress(), 200)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up freeze : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100006, issuerKey.Address.ScriptAddress())

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionThaw,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		FreezeTxId:       *freezeTxId,
		Message:          "Court order lifted",
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, orderTx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tThaw order accepted", tests.Success)

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
	if !asset.CheckBalanceFrozen(ctx, as, userPKH, 300, v.Now) {
		t.Fatalf("\t%s\tUser balance not unfrozen", tests.Failed)
	}
}

func confiscateOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100007, issuerKey.Address.ScriptAddress())

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionConfiscation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		DepositAddress:   *issuerPKH,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, orderTx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tConfiscate order accepted", tests.Success)

	// Check the response
	checkResponse(t, "E4")

	// Check balance status
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != testTokenQty+50 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty+50)
	}
	t.Logf("\t%s\tIssuer token balance verified : %d", tests.Success, issuerBalance)

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 200 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, 200)
	}
	t.Logf("\t%s\tUser token balance verified : %d", tests.Success, userBalance)
}

func reconcileOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), 150)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100008, issuerKey.Address.ScriptAddress())

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionReconciliation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 75})

	orderData.BitcoinDispersions = append(orderData.BitcoinDispersions, protocol.QuantityIndex{Index: 0, Quantity: 75000})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(752000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize order : %v", tests.Failed, err)
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create order itx : %v", tests.Failed, err)
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote order itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, orderTx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tReconcile order accepted", tests.Success)

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
			t.Logf("\t%s\tFound reconcile bitcoin dispersion", tests.Success)
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
	if issuerBalance != testTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty)
	}
	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, issuerBalance)

	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != 75 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, 75)
	}
	t.Logf("\t%s\tVerified user balance : %d", tests.Success, userBalance)
}

func mockUpFreeze(ctx context.Context, t *testing.T, pkh []byte, quantity uint64) (*protocol.TxId, error) {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000013, issuerKey.Address.ScriptAddress())

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(pkh),
		Quantity: quantity,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&orderData, test.NodeConfig.IsTest)
	if err != nil {
		return nil, err
	}
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(0, script))

	orderItx, err := inspector.NewTransactionFromWire(ctx, orderTx, test.NodeConfig.IsTest)
	if err != nil {
		return nil, err
	}

	err = orderItx.Promote(ctx, test.RPCNode)
	if err != nil {
		return nil, err
	}

	test.RPCNode.SaveTX(ctx, orderTx)
	transactions.AddTx(ctx, test.MasterDB, orderItx)

	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		return nil, err
	}

	var freezeTxId *protocol.TxId
	if len(responses) > 0 {
		hash := responses[0].TxHash()
		freezeTxId = protocol.TxIdFromBytes(hash[:])

		test.RPCNode.SaveTX(ctx, responses[0])

		freezeItx, err := inspector.NewTransactionFromWire(ctx, responses[0], test.NodeConfig.IsTest)
		if err != nil {
			return nil, err
		}

		err = freezeItx.Promote(ctx, test.RPCNode)
		if err != nil {
			return nil, err
		}

		transactions.AddTx(ctx, test.MasterDB, freezeItx)
		responses = nil
	}

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		return freezeTxId, err
	}

	for i, _ := range as.Holdings {
		if bytes.Equal(as.Holdings[i].PKH.Bytes(), pkh) {
			ts := protocol.CurrentTimestamp()
			as.Holdings[i].HoldingStatuses = append(as.Holdings[i].HoldingStatuses, state.HoldingStatus{
				Code:    'F',
				Expires: protocol.NewTimestamp(ts.Nano() + 100000000000),
				Balance: quantity,
				TxId:    *freezeTxId,
			})
			return freezeTxId, asset.Save(ctx, test.MasterDB, contractPKH, as)
		}
	}

	return freezeTxId, errors.New("Holding not found for mock freeze")
}
