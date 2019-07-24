package tests

import (
	"os"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	spynodeHandlers "github.com/tokenized/smart-contract/pkg/spynode/handlers"
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
	// b.Run("null", nullBenchmark)
}

func nullBenchmark(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count++
	}
}

func simpleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
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
	hashes := make([]*chainhash.Hash, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(i), issuerKey.Address)

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
			protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
				Quantity: transferAmount})

		transferData.Assets = append(transferData.Assets, assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(2)

		transferInputHash := fundingTx.TxHash()

		// From issuer
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

		// To contract
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

		// Data output
		script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.SaveTX(ctx, transferTx)
		requests = append(requests, transferTx)
		hash := transferTx.TxHash()
		hashes = append(hashes, &hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	walletKeys := test.Wallet.ListAll()
	pubKeys := make([][]byte, 0, len(walletKeys))
	for _, walletKey := range walletKeys {
		pubKeys = append(pubKeys, walletKey.Key.PublicKey().Bytes())
	}
	txFilter := filters.NewTxFilter(&test.NodeConfig.ChainParams, pubKeys, tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter)

	server.SetAlternateResponder(respondTx)
	server.SetInSync()

	responses = make([]*wire.MsgTx, 0, b.N)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			b.Logf("Server failed : %s", err)
		}
	}()

	profFile, err := os.OpenFile("simple_transfer_cpu.prof", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		b.Fatalf("\t%s\tFailed to create prof file : %v", tests.Failed, err)
	}
	err = pprof.StartCPUProfile(profFile)
	if err != nil {
		b.Fatalf("\t%s\tFailed to start prof : %v", tests.Failed, err)
	}
	b.ResetTimer()

	for i, request := range requests {
		if _, err := server.HandleTx(ctx, request); err != nil {
			b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
		}

		if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *hashes[i]); err != nil {
			b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
		}
	}

	responsesProcessed := 0
	for responsesProcessed < b.N {
		response := getResponse()
		if response == nil {
			time.Sleep(time.Millisecond)
			continue
		}
		responsesProcessed++
		// rType := responseType(response)
		// if rType != "T2" {
		// 	b.Fatalf("Invalid response type : %s", rType)
		// }

		if _, err := server.HandleTx(ctx, response); err != nil {
			b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
		}

		if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, response.TxHash()); err != nil {
			b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
		}
	}

	pprof.StopCPUProfile()
	b.StopTimer()

	// Check balance
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		b.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if h.FinalizedBalance != 0 {
		b.Fatalf("\t%s\tBalance not zeroized : %d", tests.Failed, h.FinalizedBalance)
	}

	server.Stop(ctx)
	wg.Wait()
}

func oracleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
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
	hashes := make([]*chainhash.Hash, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(i), issuerKey.Address)

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

		contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
		blockHeight := test.Headers.LastHeight(ctx) - 5
		blockHash, err := test.Headers.Hash(ctx, blockHeight)
		if err != nil {
			b.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
		}
		oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
			protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())), transferAmount, blockHash)
		node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
		}
		oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
		}
		receiver := protocol.AssetReceiver{
			Address:               *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
			Quantity:              transferAmount,
			OracleSigAlgorithm:    1,
			OracleConfirmationSig: oracleSig,
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
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

		// Data output
		script, err := protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.SaveTX(ctx, transferTx)
		requests = append(requests, transferTx)
		hash := transferTx.TxHash()
		hashes = append(hashes, &hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	walletKeys := test.Wallet.ListAll()
	pubKeys := make([][]byte, 0, len(walletKeys))
	for _, walletKey := range walletKeys {
		pubKeys = append(pubKeys, walletKey.Key.PublicKey().Bytes())
	}
	txFilter := filters.NewTxFilter(&test.NodeConfig.ChainParams, pubKeys, tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter)

	server.SetAlternateResponder(respondTx)
	server.SetInSync()

	responses = make([]*wire.MsgTx, 0, b.N)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			b.Logf("Server failed : %s", err)
		}
	}()

	profFile, err := os.OpenFile("oracle_transfer_cpu.prof", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		b.Fatalf("\t%s\tFailed to create prof file : %v", tests.Failed, err)
	}
	err = pprof.StartCPUProfile(profFile)
	if err != nil {
		b.Fatalf("\t%s\tFailed to start prof : %v", tests.Failed, err)
	}
	b.ResetTimer()
	for i, request := range requests {
		if _, err := server.HandleTx(ctx, request); err != nil {
			b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
		}

		if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *hashes[i]); err != nil {
			b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
		}

		// Commented because validation isn't part of smartcontract benchmark.
		// Uncomment to ensure benchmark is still functioning properly.
		// checkResponse(b, "T2")
	}

	responsesProcessed := 0
	for responsesProcessed < b.N {
		response := getResponse()
		if response == nil {
			time.Sleep(time.Millisecond)
			continue
		}
		responsesProcessed++
		// rType := responseType(response)
		// if rType != "T2" {
		// 	b.Fatalf("Invalid response type : %s", rType)
		// }

		if _, err := server.HandleTx(ctx, response); err != nil {
			b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
		}

		if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, response.TxHash()); err != nil {
			b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
		}
	}

	pprof.StopCPUProfile()
	b.StopTimer()

	// Check balance
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		b.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if h.FinalizedBalance != 0 {
		b.Fatalf("\t%s\tBalance not zeroized : %d", tests.Failed, h.FinalizedBalance)
	}

	server.Stop(ctx)
	wg.Wait()
}

func sendTokens(t *testing.T) {
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

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)

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
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

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
	test.RPCNode.SaveTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	// Check the response
	response := checkResponse(t, "T2")

	var responseMsg protocol.OpReturnMessage
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tResponse doesn't contain tokenized op return", tests.Failed)
	}

	settlement, ok := responseMsg.(*protocol.Settlement)
	if !ok {
		t.Fatalf("\t%s\tResponse isn't a settlement", tests.Failed)
	}

	if settlement.Assets[0].Settlements[0].Quantity != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[0].Quantity, testTokenQty-transferAmount)
	}

	if settlement.Assets[0].Settlements[1].Quantity != transferAmount {
		t.Fatalf("\t%s\tUser token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[1].Quantity, transferAmount)
	}

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	issuerPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)

	// Send a second transfer
	fundingTx2 := tests.MockFundingTx(ctx, test.RPCNode, 100022, issuerKey.Address)

	// Build transfer transaction
	transferTx2 := wire.NewMsgTx(2)

	transferInputHash = fundingTx2.TxHash()

	// From issuer
	transferTx2.TxIn = append(transferTx2.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx2.TxOut = append(transferTx2.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

	// Data output
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx2.TxOut = append(transferTx2.TxOut, wire.NewTxOut(0, script))

	transferItx2, err := inspector.NewTransactionFromWire(ctx, transferTx2, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx2.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, transferTx2)

	err = a.Trigger(ctx, "SEE", transferItx2)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	// Check the response
	response = checkResponse(t, "T2")

	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tResponse doesn't contain tokenized op return", tests.Failed)
	}

	settlement, ok = responseMsg.(*protocol.Settlement)
	if !ok {
		t.Fatalf("\t%s\tResponse isn't a settlement", tests.Failed)
	}

	if settlement.Assets[0].Settlements[0].Quantity != testTokenQty-(transferAmount*2) {
		t.Fatalf("\t%s\tIssuer token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[0].Quantity, testTokenQty-(transferAmount*2))
	}

	if settlement.Assets[0].Settlements[1].Quantity != transferAmount*2 {
		t.Fatalf("\t%s\tUser token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[1].Quantity, transferAmount*2)
	}

	// Check issuer and user balance
	issuerHolding, err = holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-(transferAmount*2) {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-(transferAmount*2))
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err = holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != transferAmount*2 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)
}

func multiExchange(t *testing.T) {
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
	user1HoldingBalance := uint64(100)
	err = mockUpHolding(ctx, userKey.Address, user1HoldingBalance)
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
	user2HoldingBalance := uint64(200)
	err = mockUpHolding2(ctx, user2Key.Address, user2HoldingBalance)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding 2 : %v", tests.Failed, err)
	}

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address)

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
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(user2Key.Key.PublicKey().Bytes())),
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
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
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
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, test.ContractKey.Address.LockingScript()))
	// To contract2
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(1000, test.Contract2Key.Address.LockingScript()))
	// To contract1 boomerang
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(5000, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

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

	test.RPCNode.SaveTX(ctx, response)

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

	test.RPCNode.SaveTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tSignature request accepted", tests.Success)

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	user1PKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	user1Holding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, user1PKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user1Holding.FinalizedBalance != user1HoldingBalance-transfer1Amount {
		t.Fatalf("\t%s\tUser 1 token 1 balance incorrect : %d != %d", tests.Failed,
			user1Holding.FinalizedBalance, user1HoldingBalance-transfer1Amount)
	}

	t.Logf("\t%s\tUser 1 token 1 balance : %d", tests.Success, user1Holding.FinalizedBalance)

	user2PKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(user2Key.Key.PublicKey().Bytes()))
	user2Holding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, user2PKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user2Holding.FinalizedBalance != transfer1Amount {
		t.Fatalf("\t%s\tUser 2 token 1 balance incorrect : %d != %d", tests.Failed,
			user2Holding.FinalizedBalance, transfer1Amount)
	}

	t.Logf("\t%s\tUser 2 token 1 balance : %d", tests.Success, user2Holding.FinalizedBalance)

	contract2PKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.Contract2Key.Key.PublicKey().Bytes()))
	user1Holding, err = holdings.GetHolding(ctx, test.MasterDB, contract2PKH, &testAsset2Code, user1PKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}

	if user1Holding.FinalizedBalance != transfer2Amount {
		t.Fatalf("\t%s\tUser 1 token 2 balance incorrect : %d != %d", tests.Failed, user1Holding.FinalizedBalance, transfer2Amount)
	}

	t.Logf("\t%s\tUser 1 token 2 balance : %d", tests.Success, user1Holding.FinalizedBalance)

	user2Holding, err = holdings.GetHolding(ctx, test.MasterDB, contract2PKH, &testAsset2Code, user2PKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user2Holding.FinalizedBalance != user2HoldingBalance-transfer2Amount {
		t.Fatalf("\t%s\tUser 2 token 2 balance incorrect : %d != %d", tests.Failed,
			user2Holding.FinalizedBalance, user2HoldingBalance-transfer2Amount)
	}

	t.Logf("\t%s\tUser 2 token 2 balance : %d", tests.Success, user2Holding.FinalizedBalance)
}

func oracleTransfer(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
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

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	blockHeight := test.Headers.LastHeight(ctx) - 5
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
		protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())), transferAmount, blockHash)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := protocol.AssetReceiver{
		Address:               *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig,
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
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	issuerPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)
}

func oracleTransferBad(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
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

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address)
	bitcoinFundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100017, userKey.Address)

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

	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	blockHeight := test.Headers.LastHeight(ctx) - 4
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, contractPKH, &testAssetCode,
		protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())), transferAmount+1, blockHash)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := protocol.AssetReceiver{
		Address:               *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig,
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
		Address:  *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes())),
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
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(52000, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

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
		address, err := bitcoin.AddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}
		if address.Equal(userKey.Address) && output.Value >= int64(bitcoinTransferAmount) {
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

	if err := resetTest(ctx); err != nil {
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

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)

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
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tTransfer accepted", tests.Success)

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(test.ContractKey.Key.PublicKey().Bytes()))
	issuerPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(issuerKey.Key.PublicKey().Bytes()))
	v := ctx.Value(node.KeyValues).(*node.Values)
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, issuerPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes()))
	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, contractPKH, &testAssetCode, userPKH, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)
}

func permittedBad(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
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
	err = mockUpHolding(ctx, user2Key.Address, user2Holding)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up holding : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 256, user2Key.Address)

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
		protocol.AssetReceiver{Address: *protocol.PublicKeyHashFromBytes(bitcoin.Hash160(userKey.Key.PublicKey().Bytes())),
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
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, test.ContractKey.Address.LockingScript()))

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

	test.RPCNode.SaveTX(ctx, transferTx)

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
