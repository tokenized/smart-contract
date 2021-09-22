package tests

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
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
	mockUpContract(t, ctx, "Test Contract", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	mockUpHolding(t, ctx, userKey.Address, 300)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction: actions.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCodes[0].Bytes(),
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, &actions.TargetAddressField{
		Address:  userKey.Address.Bytes(),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Freeze Order : %s", orderItx.Hash.String())
	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tFreeze order accepted", tests.Success)

	// Check the response
	checkResponse(t, "E2")

	// Check balance status
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	balance := holdings.UnfrozenBalance(h, v.Now)
	if balance != 100 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 100)
	}
}

func freezeAuthorityOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	mockUpHolding(t, ctx, userKey.Address, 300)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction:   actions.ComplianceActionFreeze,
		AssetType:          testAssetType,
		AssetCode:          testAssetCodes[0].Bytes(),
		Message:            "Court order",
		AuthorityName:      "District Court #345",
		AuthorityPublicKey: authorityKey.Key.PublicKey().Bytes(),
		SignatureAlgorithm: 1,
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, &actions.TargetAddressField{
		Address:  userKey.Address.Bytes(),
		Quantity: 200,
	})

	sigHash, err := protocol.OrderAuthoritySigHash(ctx, test.ContractKey.Address, &orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed generate authority signature hash : %v", tests.Failed, err)
	}

	sig, err := authorityKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to sign authority sig hash : %v", tests.Failed, err)
	}
	orderData.OrderSignature = sig.Bytes()

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Freeze Order : %s", orderItx.Hash.String())
	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tFreeze order with authority accepted", tests.Success)

	// Check the response
	checkResponse(t, "E2")

	// Check balance status
	v := ctx.Value(node.KeyValues).(*node.Values)

	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	balance := holdings.UnfrozenBalance(h, v.Now)
	if balance != 100 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 100)
	}
}

func thawOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	mockUpHolding(t, ctx, userKey.Address, 300)

	freezeTxId, err := mockUpFreeze(ctx, t, userKey.Address, 200)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up freeze : %v", tests.Failed, err)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100006, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction: actions.ComplianceActionThaw,
		AssetType:        testAssetType,
		AssetCode:        testAssetCodes[0].Bytes(),
		FreezeTxId:       freezeTxId.Bytes(),
		Message:          "Court order lifted",
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Thaw Order : %s", orderItx.Hash.String())
	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tThaw order accepted", tests.Success)

	// Check the response
	checkResponse(t, "E3")

	// Check balance status
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}

	balance := holdings.UnfrozenBalance(h, v.Now)
	if balance != 300 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 300)
	}
}

func confiscateOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	mockUpHolding(t, ctx, userKey.Address, 250)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100007, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction: actions.ComplianceActionConfiscation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCodes[0].Bytes(),
		DepositAddress:   issuerKey.Address.Bytes(),
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses,
		&actions.TargetAddressField{Address: userKey.Address.Bytes(), Quantity: 50})

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(3200, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Confiscate Order : %s", orderItx.Hash.String())
	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept order : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tConfiscate order accepted", tests.Success)

	// Check the response
	checkResponse(t, "E4")

	// Check balance status
	v := ctx.Value(node.KeyValues).(*node.Values)

	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty+50 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty+50)
	}
	t.Logf("\t%s\tIssuer token balance verified : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	if !userKey.Address.Equal(userHolding.Address) {
		t.Fatalf("\t%s\tFailed to get correct user holding : %x != %x", tests.Failed,
			userKey.Address.Bytes(), userHolding.Address.Bytes())
	}
	if userHolding.FinalizedBalance != 200 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d/%d != %d : %x", tests.Failed,
			userHolding.FinalizedBalance, userHolding.PendingBalance, 200, userHolding.Address.Bytes())
	}
	t.Logf("\t%s\tUser token balance verified : %d", tests.Success, userHolding.FinalizedBalance)
}

func reconcileOrder(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)
	mockUpHolding(t, ctx, userKey.Address, 150)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100008, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction: actions.ComplianceActionReconciliation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCodes[0].Bytes(),
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses,
		&actions.TargetAddressField{Address: userKey.Address.Bytes(), Quantity: 75})

	orderData.BitcoinDispersions = append(orderData.BitcoinDispersions,
		&actions.QuantityIndexField{Index: 0, Quantity: 75000})

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(752000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Reconcile Order : %s", orderItx.Hash.String())
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
		address, err := bitcoin.RawAddressFromLockingScript(output.LockingScript)
		if err != nil {
			continue
		}
		if address.Equal(userKey.Address) && output.Value == 75000 {
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
	v := ctx.Value(node.KeyValues).(*node.Values)

	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty)
	}
	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != 75 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, 75)
	}
	t.Logf("\t%s\tVerified user balance : %d", tests.Success, userHolding.FinalizedBalance)
}

func mockUpFreeze(ctx context.Context, t *testing.T, address bitcoin.RawAddress, quantity uint64) (*bitcoin.Hash32, error) {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000013, issuerKey.Address)

	orderData := actions.Order{
		ComplianceAction: actions.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCodes[0].Bytes(),
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, &actions.TargetAddressField{
		Address:  address.Bytes(),
		Quantity: quantity,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(1)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(orderInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&orderData, test.NodeConfig.IsTest)
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

	t.Logf("Mocked Freeze tx : %s", orderItx.Hash.String())
	err = a.Trigger(ctx, "SEE", orderItx)
	if err != nil {
		return nil, err
	}

	var freezeTxId *bitcoin.Hash32
	if len(responses) > 0 {
		hash := responses[0].TxHash()
		freezeTxId = hash

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

		err = a.Trigger(ctx, "SEE", freezeItx)
		if err != nil {
			return nil, err
		}
	}

	return freezeTxId, nil

	// contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	// pubkeyhash := protocol.PublicKeyHashFromBytes(pkh)
	// v := ctx.Value(node.KeyValues).(*node.Values)
	// h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCodes[0], pubkeyhash, v.Now)
	// if err != nil {
	// 	t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	// }
	//
	// ts := protocol.CurrentTimestamp()
	// err = holdings.AddFreeze(h, freezeTxId, quantity, protocol.CurrentTimestamp(),
	// 	protocol.NewTimestamp(ts.Nano()+100000000000))
	// holdings.FinalizeTx(h, freezeTxId, v.Now)
	// return freezeTxId, holdings.Save(ctx, test.MasterDB, contractPKH, &testAssetCodes[0], h)
}
