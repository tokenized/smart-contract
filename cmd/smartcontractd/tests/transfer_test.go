package tests

import (
	"testing"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestTransfers is the entry point for testing transfer functions.
func TestTransfers(t *testing.T) {
	defer tests.Recover(t)

	t.Run("sendTokens", sendTokens)
	t.Run("multiExchange", multiExchange)
}

func sendTokens(t *testing.T) {
	ctx := test.Context

	resetTest()
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, protocol.TokenReceiver{Index: 1, Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To user
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted transfer with insufficient value", tests.Failed)
	}

	if len(responses) != 0 {
		t.Fatalf("\t%s\tHandle asset transfer created reject response", tests.Failed)
	}

	t.Logf("\t%s\tUnderfunded asset transfer rejected with no response", tests.Success)

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 2000

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	// Resubmit
	test.RPCNode.AddTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerBalance)

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userBalance)
}

func multiExchange(t *testing.T) {
	ctx := test.Context

	resetTest()
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	user1Holding := uint64(100)
	err = mockUpHolding(ctx, userKey.Address.ScriptAddress(), user1Holding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	err = mockUpContract2(ctx, "Test Contract 2", "This is a mock contract and means nothing.", 'I', 1, "Karl Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset2(ctx, true, true, true, 1500, &sampleAssetPayload2, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset 2 : %v", tests.Failed, err)
	}
	user2Holding := uint64(200)
	err = mockUpHolding2(ctx, user2Key.Address.ScriptAddress(), user2Holding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding 2 : %v", tests.Failed, err)
	}

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address.ScriptAddress())
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address.ScriptAddress())

	// Create Transfer message
	transferData := protocol.Transfer{}

	// Transfer asset 1 from user1 to user2
	transfer1Amount := uint64(50)
	assetTransfer1Data := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers, protocol.TokenReceiver{Index: 4, Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(150)
	assetTransfer2Data := protocol.AssetTransfer{
		ContractIndex: 1, // first output
		AssetType:     testAsset2Type,
		AssetCode:     testAsset2Code,
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders, protocol.QuantityIndex{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers, protocol.TokenReceiver{Index: 3, Quantity: transfer2Amount})

	transferData.Assets = append(transferData.Assets, assetTransfer2Data)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	// From user1
	transferInputHash := funding1Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// From user2
	transferInputHash = funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract1
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))
	// To contract2
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(1000, txbuilder.P2PKHScriptForPKH(test.Contract2Key.Address.ScriptAddress())))
	// To contract1 boomerang
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(5000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To user
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))
	// To user2
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(user2Key.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create transfer response", tests.Failed)
	}

	response := responses[0]
	responses = nil

	responseItx, err := inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create response itx : %v", tests.Failed, err)
	}

	err = responseItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote response itx : %v", tests.Failed, err)
	}

	if responseItx.MsgProto.Type() != "M1" {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	settlementRequestMessage, ok := responseItx.MsgProto.(*protocol.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if settlementRequestMessage.MessageType != protocol.CodeSettlementRequest {
		t.Fatalf("\t%s\tResponse itx is not Settlement Request : %d", tests.Failed, settlementRequestMessage.MessageType)
	}

	test.RPCNode.AddTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tSettlement request accepted", tests.Success)

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create settlement request response", tests.Failed)
	}

	response = responses[0]
	responses = nil

	responseItx, err = inspector.NewTransactionFromWire(ctx, response)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create response itx : %v", tests.Failed, err)
	}

	err = responseItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote response itx : %v", tests.Failed, err)
	}

	if responseItx.MsgProto.Type() != "M1" {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	signatureRequestMessage, ok := responseItx.MsgProto.(*protocol.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if signatureRequestMessage.MessageType != protocol.CodeSignatureRequest {
		t.Fatalf("\t%s\tResponse itx is not Signature Request : %d", tests.Failed, signatureRequestMessage.MessageType)
	}

	test.RPCNode.AddTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tSignature request accepted", tests.Success)

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as1, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset 1 : %v", tests.Failed, err)
	}

	user1PKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	user1Balance := asset.GetBalance(ctx, as1, user1PKH)
	if user1Balance != user1Holding-transfer1Amount {
		t.Fatalf("\t%s\tUser 1 token 1 balance incorrect : %d != %d", tests.Failed, user1Balance, user1Holding-transfer1Amount)
	}

	t.Logf("\t%s\tUser 1 token 1 balance : %d", tests.Success, user1Balance)

	user2PKH := protocol.PublicKeyHashFromBytes(user2Key.Address.ScriptAddress())
	user2Balance := asset.GetBalance(ctx, as1, user2PKH)
	if user2Balance != transfer1Amount {
		t.Fatalf("\t%s\tUser 2 token 1 balance incorrect : %d != %d", tests.Failed, user2Balance, transfer1Amount)
	}

	t.Logf("\t%s\tUser 2 token 1 balance : %d", tests.Success, user2Balance)

	contract2PKH := protocol.PublicKeyHashFromBytes(test.Contract2Key.Address.ScriptAddress())
	as2, err := asset.Retrieve(ctx, test.MasterDB, contract2PKH, &testAsset2Code)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset 2 : %v", tests.Failed, err)
	}

	user1Balance = asset.GetBalance(ctx, as2, user1PKH)
	if user1Balance != transfer2Amount {
		t.Fatalf("\t%s\tUser 1 token 2 balance incorrect : %d != %d", tests.Failed, user1Balance, transfer2Amount)
	}

	t.Logf("\t%s\tUser 1 token 2 balance : %d", tests.Success, user1Balance)

	user2Balance = asset.GetBalance(ctx, as2, user2PKH)
	if user2Balance != user2Holding-transfer2Amount {
		t.Fatalf("\t%s\tUser 2 token 2 balance incorrect : %d != %d", tests.Failed, user2Balance, user2Holding-transfer2Amount)
	}

	t.Logf("\t%s\tUser 2 token 2 balance : %d", tests.Success, user2Balance)
}
