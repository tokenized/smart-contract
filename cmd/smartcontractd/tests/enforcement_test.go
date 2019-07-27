package tests

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
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
	err = mockUpHolding(ctx, userKey.Address, 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
		Quantity: 200,
	})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	balance := holdings.UnfrozenBalance(&h, v.Now)
	if balance != 100 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 100)
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
	err = mockUpHolding(ctx, userKey.Address, 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100005, issuerKey.Address)

	orderData := protocol.Order{
		ComplianceAction:   protocol.ComplianceActionFreeze,
		AssetType:          testAssetType,
		AssetCode:          testAssetCode,
		Message:            "Court order",
		AuthorityIncluded:  true,
		AuthorityName:      "District Court #345",
		AuthorityPublicKey: authorityKey.Key.PublicKey().Bytes(),
		SignatureAlgorithm: 1,
	}

	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{
		Address:  *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
		Quantity: 200,
	})

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	sigHash, err := protocol.OrderAuthoritySigHash(ctx, contractPKH, &orderData)
	if err != nil {
		t.Fatalf("\t%s\tFailed generate authority signature hash : %v", tests.Failed, err)
	}

	orderData.OrderSignature, err = authorityKey.Key.Sign(sigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to sign authority sig hash : %v", tests.Failed, err)
	}

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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
	v := ctx.Value(node.KeyValues).(*node.Values)

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	balance := holdings.UnfrozenBalance(&h, v.Now)
	if balance != 100 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 100)
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
	err = mockUpHolding(ctx, userKey.Address, 300)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}
	freezeTxId, err := mockUpFreeze(ctx, t, userKey.Address, 200)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up freeze : %v", tests.Failed, err)
	}

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100006, issuerKey.Address)

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
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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
	h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}

	balance := holdings.UnfrozenBalance(&h, v.Now)
	if balance != 300 {
		t.Fatalf("\t%s\tUser unfrozen balance incorrect : %d != %d", tests.Failed, balance, 300)
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
	err = mockUpHolding(ctx, userKey.Address, 250)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100007, issuerKey.Address)

	issuerPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionConfiscation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		DepositAddress:   *issuerPKH,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 50})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2500, test.ContractKey.Address.LockingScript()))

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
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)

	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty+50 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty+50)
	}
	t.Logf("\t%s\tIssuer token balance verified : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get user holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != 200 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userHolding.FinalizedBalance, 200)
	}
	t.Logf("\t%s\tUser token balance verified : %d", tests.Success, userHolding.FinalizedBalance)
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
	err = mockUpHolding(ctx, userKey.Address, 150)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100008, issuerKey.Address)

	issuerPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionReconciliation,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	orderData.TargetAddresses = append(orderData.TargetAddresses, protocol.TargetAddress{Address: *userPKH, Quantity: 75})

	orderData.BitcoinDispersions = append(orderData.BitcoinDispersions, protocol.QuantityIndex{Index: 0, Quantity: 75000})

	// Build order transaction
	orderTx := wire.NewMsgTx(2)

	orderInputHash := fundingTx.TxHash()

	// From issuer
	orderTx.TxIn = append(orderTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&orderInputHash, 0), make([]byte, 130)))

	// To contract
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(752000, test.ContractKey.Address.LockingScript()))

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
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}
		var pkh []byte
		switch a := address.(type) {
		case *bitcoin.RawAddressPKH:
			pkh = a.PKH()
		case *bitcoin.AddressPKH:
			pkh = a.PKH()
		default:
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
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)

	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerHolding.FinalizedBalance, testTokenQty)
	}
	t.Logf("\t%s\tVerified issuer balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get issuer holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != 75 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userHolding.FinalizedBalance, 75)
	}
	t.Logf("\t%s\tVerified user balance : %d", tests.Success, userHolding.FinalizedBalance)
}

func mockUpFreeze(ctx context.Context, t *testing.T, address bitcoin.RawAddress, quantity uint64) (*protocol.TxId, error) {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 1000013, issuerKey.Address)

	orderData := protocol.Order{
		ComplianceAction: protocol.ComplianceActionFreeze,
		AssetType:        testAssetType,
		AssetCode:        testAssetCode,
		Message:          "Court order",
	}

	var pkh []byte
	switch a := address.(type) {
	case *bitcoin.RawAddressPKH:
		pkh = a.PKH()
	case *bitcoin.AddressPKH:
		pkh = a.PKH()
	default:
		return nil, errors.New("Address not PKH")
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
	orderTx.TxOut = append(orderTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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

		err = a.Trigger(ctx, "SEE", freezeItx)
		if err != nil {
			return nil, err
		}
	}

	return freezeTxId, nil

	// contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	// pubkeyhash := protocol.PublicKeyHashFromBytes(pkh)
	// v := ctx.Value(node.KeyValues).(*node.Values)
	// h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, pubkeyhash, v.Now)
	// if err != nil {
	// 	t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	// }
	//
	// ts := protocol.CurrentTimestamp()
	// err = holdings.AddFreeze(&h, freezeTxId, quantity, protocol.CurrentTimestamp(),
	// 	protocol.NewTimestamp(ts.Nano()+100000000000))
	// holdings.FinalizeTx(&h, freezeTxId, v.Now)
	// return freezeTxId, holdings.Save(ctx, test.MasterDB, contractPKH, &testAssetCode, &h)
}
