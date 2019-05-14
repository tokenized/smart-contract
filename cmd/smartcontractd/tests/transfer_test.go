package tests

import (
	"bytes"
	"testing"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/node"
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
	t.Run("oracle", oracleTransfer)
	t.Run("oracleBad", oracleTransferBad)
	t.Run("permitted", permitted)
	t.Run("permittedBad", permittedBad)
}

func BenchmarkTransfers(b *testing.B) {
	defer tests.Recover(b)

	b.Run("simple", simpleTransfersBenchmark)
	b.Run("oracle", oracleTransfersBenchmark)
}

func simpleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, uint64(b.N), &sampleAssetPayload, true, false, false)
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}

	requests := make([]*wire.MsgTx, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(b.N), issuerKey.Address.ScriptAddress())

		// Create Transfer message
		transferAmount := uint64(1)
		transferData := protocol.Transfer{}

		assetTransferData := protocol.AssetTransfer{
			ContractIndex: 0, // first output
			AssetType:     testAssetType,
			AssetCode:     testAssetCode,
		}

		assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
			protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
		assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
			protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
				Quantity: transferAmount})

		transferData.Assets = append(transferData.Assets, assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(2)

		transferInputHash := fundingTx.TxHash()

		// From issuer
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

		// To contract
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

		// Data output
		script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.AddTX(ctx, transferTx)
		requests = append(requests, transferTx)
	}

	b.ResetTimer()
	for _, request := range requests {
		transferItx, err := inspector.NewTransactionFromWire(ctx, request, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
		}

		err = transferItx.Promote(ctx, test.RPCNode)
		if err != nil {
			b.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
		}

		if err := a.Trigger(ctx, "SEE", transferItx); err != nil {
			b.Fatalf("\t%s\tTransfer failed : %v", tests.Failed, err)
		}

		// Commented because validation isn't part of smartcontract benchmark.
		// Uncomment to ensure benchmark is still functioning properly.
		// checkResponse(b, "T2")
	}
}

func oracleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContractWithOracle(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin")
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up contract with oracle : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, uint64(b.N), &sampleAssetPayload, true, false, false)
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	requests := make([]*wire.MsgTx, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(b.N), issuerKey.Address.ScriptAddress())

		// Create Transfer message
		transferAmount := uint64(1)
		transferData := protocol.Transfer{}

		assetTransferData := protocol.AssetTransfer{
			ContractIndex: 0, // first output
			AssetType:     testAssetType,
			AssetCode:     testAssetCode,
		}

		assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
			protocol.QuantityIndex{Index: 0, Quantity: transferAmount})

		contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
		blockHeight := test.Headers.LastHeight(ctx) - 5
		blockHash, err := test.Headers.Hash(ctx, blockHeight)
		if err != nil {
			b.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
		}
		oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
			protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()), transferAmount, blockHash)
		node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
		}
		oracleSig, err := oracleKey.PrivateKey.Sign(oracleSigHash)
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
		}
		receiver := protocol.AssetReceiver{
			Address:               *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
			Quantity:              transferAmount,
			OracleSigAlgorithm:    1,
			OracleConfirmationSig: oracleSig.Serialize(),
			OracleSigBlockHeight:  uint32(blockHeight),
		}
		assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, receiver)

		transferData.Assets = append(transferData.Assets, assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(2)

		transferInputHash := fundingTx.TxHash()

		// From issuer
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

		// To contract
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

		// Data output
		script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.AddTX(ctx, transferTx)
		requests = append(requests, transferTx)
	}

	b.ResetTimer()
	for _, request := range requests {
		transferItx, err := inspector.NewTransactionFromWire(ctx, request, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
		}

		err = transferItx.Promote(ctx, test.RPCNode)
		if err != nil {
			b.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
		}

		if err := a.Trigger(ctx, "SEE", transferItx); err != nil {
			b.Fatalf("\t%s\tTransfer failed : %v", tests.Failed, err)
		}

		// Commented because validation isn't part of smartcontract benchmark.
		// Uncomment to ensure benchmark is still functioning properly.
		// checkResponse(b, "T2")
	}
}

func sendTokens(t *testing.T) {
	ctx := test.Context

	if err := resetTest(); err != nil {
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

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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
		t.Fatalf("\t%s\tAccepted transfer with insufficient funds", tests.Failed)
	}
	if err != node.ErrNoResponse {
		t.Fatalf("\t%s\tFailed to reject transfer with insufficient funds : %v", tests.Failed, err)
	}

	if len(responses) != 0 {
		t.Fatalf("\t%s\tHandle asset transfer created reject response without sufficient funds", tests.Failed)
	}

	t.Logf("\t%s\tUnderfunded asset transfer rejected with no response", tests.Success)

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 2000

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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

	if err := resetTest(); err != nil {
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

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders,
		protocol.QuantityIndex{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers,
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(user2Key.Address.ScriptAddress()),
			Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(150)
	assetTransfer2Data := protocol.AssetTransfer{
		ContractIndex: 1, // first output
		AssetType:     testAsset2Type,
		AssetCode:     testAsset2Code,
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders,
		protocol.QuantityIndex{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers,
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
			Quantity: transfer2Amount})

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

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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

	responseItx, err := inspector.NewTransactionFromWire(ctx, response, test.NodeConfig.IsTest)
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

	responseItx, err = inspector.NewTransactionFromWire(ctx, response, test.NodeConfig.IsTest)
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

func oracleTransfer(t *testing.T) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContractWithOracle(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract with oracle : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	blockHeight := test.Headers.LastHeight(ctx) - 5
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
		protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()), transferAmount, blockHash)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.PrivateKey.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := protocol.AssetReceiver{
		Address:               *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig.Serialize(),
		OracleSigBlockHeight:  uint32(blockHeight),
	}
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, receiver)

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
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

func oracleTransferBad(t *testing.T) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContractWithOracle(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin")
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract with oracle : %v", tests.Failed, err)
	}
	err = mockUpAsset(ctx, true, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	err = test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address.ScriptAddress())
	bitcoinFundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100017, userKey.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		protocol.QuantityIndex{Index: 0, Quantity: transferAmount})

	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	blockHeight := test.Headers.LastHeight(ctx) - 4
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
		protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()), transferAmount+1, blockHash)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.PrivateKey.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := protocol.AssetReceiver{
		Address:               *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig.Serialize(),
		OracleSigBlockHeight:  uint32(blockHeight),
	}
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, receiver)

	transferData.Assets = append(transferData.Assets, assetTransferData)

	bitcoinTransferAmount := uint64(50000)
	bitcoinTransferData := protocol.AssetTransfer{
		ContractIndex: uint16(0xffff),
		AssetType:     protocol.CodeCurrency,
	}

	bitcoinTransferData.AssetSenders = append(bitcoinTransferData.AssetSenders, protocol.QuantityIndex{Index: 1, Quantity: bitcoinTransferAmount})

	bitcoinTransferData.AssetReceivers = append(bitcoinTransferData.AssetReceivers, protocol.AssetReceiver{
		Address:  *protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress()),
		Quantity: bitcoinTransferAmount,
	})

	transferData.Assets = append(transferData.Assets, bitcoinTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	// From issuer
	transferInputHash := fundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// From user
	bitcoinInputHash := bitcoinFundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&bitcoinInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(52000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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
		t.Fatalf("\t%s\tAccepted transfer with invalid oracle sig", tests.Failed)
	}
	if err != node.ErrRejected {
		t.Fatalf("\t%s\tFailed to reject transfer with invalid oracle sig : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer rejected", tests.Success)

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed create reject response", tests.Failed)
	}
	response := responses[0]

	// Check the response
	checkResponse(t, "M2")

	// Find refund output
	found := false
	for _, output := range response.TxOut {
		hash, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}
		if bytes.Equal(hash, userKey.Address.ScriptAddress()) && output.Value >= int64(bitcoinTransferAmount) {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("\t%s\tRefund to user not found", tests.Failed)
	}

	t.Logf("\t%s\tVerified refund to user", tests.Success)
}

func permitted(t *testing.T) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	// TransfersPermitted = false
	err = mockUpAsset(ctx, false, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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

func permittedBad(t *testing.T) {
	ctx := test.Context

	if err := resetTest(); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	err := mockUpContract(ctx, "Test Contract", "This is a mock contract and means nothing.", 'I', 1, "John Bitcoin", true, true, false, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up contract : %v", tests.Failed, err)
	}
	// TransfersPermitted = false
	err = mockUpAsset(ctx, false, true, true, 1000, &sampleAssetPayload, true, false, false)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up asset : %v", tests.Failed, err)
	}
	user2Holding := uint64(100)
	err = mockUpHolding(ctx, user2Key.Address.ScriptAddress(), user2Holding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address.ScriptAddress())
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 256, user2Key.Address.ScriptAddress())

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		protocol.QuantityIndex{Index: 1, Quantity: user2Holding})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress()),
			Quantity: transferAmount + user2Holding})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	// From issuer
	transferInputHash := fundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))
	// From user2
	transfer2InputHash := funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transfer2InputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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
		t.Fatalf("\t%s\tAccepted non-permitted transfer", tests.Failed)
	}
	if err != node.ErrRejected {
		t.Fatalf("\t%s\tWrong error on non-permitted transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer rejected", tests.Success)

	response := responses[0]

	// Check the response
	checkResponse(t, "M2")

	rejectItx, err := inspector.NewTransactionFromWire(ctx, response, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create reject itx : %v", tests.Failed, err)
	}

	err = rejectItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote reject itx : %v", tests.Failed, err)
	}

	reject, ok := rejectItx.MsgProto.(*protocol.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert reject data", tests.Failed)
	}

	if reject.RejectionCode != protocol.RejectAssetNotPermitted {
		t.Fatalf("\t%s\tRejection code incorrect : %d", tests.Failed, reject.RejectionCode)
	}

	t.Logf("\t%s\tVerified rejection code", tests.Success)
}
