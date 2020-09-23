package tests

import (
	"context"
	"os"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/scheduler"
	spynodeHandlers "github.com/tokenized/pkg/spynode/handlers"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/filters"
	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/messages"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestTransfers is the entry point for testing transfer functions.
func TestTransfers(t *testing.T) {
	defer tests.Recover(t)

	t.Run("sendTokens", sendTokens)
	t.Run("multiExchange", multiExchange)
	t.Run("bitcoinExchange", bitcoinExchange)
	t.Run("multiExchangeLock", multiExchangeLock)
	t.Run("multiExchangeTimeout", multiExchangeTimeout)
	t.Run("oracle", oracleTransfer)
	t.Run("oracleBad", oracleTransferBad)
	t.Run("permitted", permitted)
	t.Run("permittedBad", permittedBad)
}

func BenchmarkTransfers(b *testing.B) {
	defer tests.Recover(b)

	b.Run("simple", simpleTransfersBenchmark)
	b.Run("separate", separateTransfersBenchmark)
	b.Run("oracle", oracleTransfersBenchmark)
	b.Run("tree", treeTransfersBenchmark)
	// b.Run("null", nullBenchmark)
}

func nullBenchmark(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		count++
	}
}

// simpleTransfersBenchmark simulates a transfer from the issuer to N addresses.
func simpleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(b, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(b, ctx, true, true, true, uint64(b.N), 0, &sampleAssetPayload, true, false, false)

	requests := make([]*wire.MsgTx, 0, b.N)
	hashes := make([]*bitcoin.Hash32, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(i), issuerKey.Address)

		// Create Transfer message
		transferAmount := uint64(1)
		transferData := actions.Transfer{}

		assetTransferData := actions.AssetTransferField{
			ContractIndex: 0, // first output
			AssetType:     testAssetType,
			AssetCode:     testAssetCodes[0].Bytes(),
		}

		assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
			&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})
		assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
			&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
				Quantity: transferAmount})

		transferData.Assets = append(transferData.Assets, &assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(1)

		transferInputHash := fundingTx.TxHash()

		// From issuer
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

		// To contract
		script, _ := test.ContractKey.Address.LockingScript()
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, script))

		// Data output
		var err error
		script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.SaveTX(ctx, transferTx)
		requests = append(requests, transferTx)
		hash := transferTx.TxHash()
		hashes = append(hashes, hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		test.HoldingsChannel)

	if err := server.Load(ctx); err != nil {
		b.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		b.Fatalf("Failed to load wallet : %s", err)
	}

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

	wgInternal := sync.WaitGroup{}
	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		for i, request := range requests {
			if _, err := server.HandleTx(ctx, request); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *hashes[i]); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		responsesProcessed := 0
		for responsesProcessed < b.N {
			response := getResponse()
			if response == nil {
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

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *response.TxHash()); err != nil {
				b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Wait()

	pprof.StopCPUProfile()
	b.StopTimer()

	server.Stop(ctx)
	wg.Wait()

	// Check balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		b.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if h.FinalizedBalance != 0 {
		b.Fatalf("\t%s\tBalance not zeroized : %d", tests.Failed, h.FinalizedBalance)
	}
}

// separateTransfersBenchmark simulates a transfer from the issuer to N addresses.
func separateTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(b, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(b, ctx, true, true, true, uint64(b.N), 0, &sampleAssetPayload, true, false, false)

	requests := make([]*wire.MsgTx, 0, b.N)
	hashes := make([]*bitcoin.Hash32, 0, b.N)
	senders := make([]*wallet.Key, 0, b.N)
	receivers := make([]*wallet.Key, 0, b.N)
	transferAmount := uint64(1)
	for i := 0; i < b.N; i++ {
		senderKey, err := tests.GenerateKey(test.NodeConfig.Net)
		if err != nil {
			b.Fatalf("\t%s\tFailed to generate key : %v", tests.Failed, err)
		}
		senders = append(senders, senderKey)

		mockUpHolding(b, ctx, senderKey.Address, transferAmount)

		receiverKey, err := tests.GenerateKey(test.NodeConfig.Net)
		if err != nil {
			b.Fatalf("\t%s\tFailed to generate key : %v", tests.Failed, err)
		}

		receivers = append(receivers, receiverKey)

		// Create Transfer message
		transferData := actions.Transfer{}

		assetTransferData := actions.AssetTransferField{
			ContractIndex: 0, // first output
			AssetType:     testAssetType,
			AssetCode:     testAssetCodes[0].Bytes(),
		}

		assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
			&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})
		assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
			&actions.AssetReceiverField{Address: receiverKey.Address.Bytes(),
				Quantity: transferAmount})

		transferData.Assets = append(transferData.Assets, &assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(1)

		// From sender
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(i), senderKey.Address)
		transferInputHash := fundingTx.TxHash()
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

		// To contract
		script, _ := test.ContractKey.Address.LockingScript()
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, script))

		// Data output
		script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.SaveTX(ctx, transferTx)
		requests = append(requests, transferTx)
		hash := transferTx.TxHash()
		hashes = append(hashes, hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		test.HoldingsChannel)

	if err := server.Load(ctx); err != nil {
		b.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		b.Fatalf("Failed to load wallet : %s", err)
	}

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

	profFile, err := os.OpenFile("separate_transfer_cpu.prof", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		b.Fatalf("\t%s\tFailed to create prof file : %v", tests.Failed, err)
	}
	err = pprof.StartCPUProfile(profFile)
	if err != nil {
		b.Fatalf("\t%s\tFailed to start prof : %v", tests.Failed, err)
	}
	b.ResetTimer()

	wgInternal := sync.WaitGroup{}
	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		for i, request := range requests {
			if _, err := server.HandleTx(ctx, request); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *hashes[i]); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		responsesProcessed := 0
		for responsesProcessed < b.N {
			response := getResponse()
			if response == nil {
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

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *response.TxHash()); err != nil {
				b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Wait()

	pprof.StopCPUProfile()
	b.StopTimer()

	server.Stop(ctx)
	wg.Wait()

	// Check balance
	for _, sender := range senders {
		v := ctx.Value(node.KeyValues).(*node.Values)
		h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
			sender.Address, v.Now)
		if err != nil {
			b.Fatalf("\t%s\tFailed to get sender holding : %s", tests.Failed, err)
		}
		if h.FinalizedBalance != 0 {
			b.Fatalf("\t%s\tSender balance incorrect : %d", tests.Failed, h.FinalizedBalance)
		}
	}

	for _, receiver := range receivers {
		v := ctx.Value(node.KeyValues).(*node.Values)
		h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
			receiver.Address, v.Now)
		if err != nil {
			b.Fatalf("\t%s\tFailed to get receiver holding : %s", tests.Failed, err)
		}
		if h.FinalizedBalance != transferAmount {
			b.Fatalf("\t%s\tReceiver balance incorrect : %d", tests.Failed, h.FinalizedBalance)
		}
	}
}

func oracleTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContractWithOracle(b, ctx, "Test Contract", "This is a mock contract and means nothing.",
		"I", 1, "John Bitcoin")
	mockUpAsset(b, ctx, true, true, true, uint64(b.N), 0, &sampleAssetPayload, true, false, false)

	err := test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		b.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())

	requests := make([]*wire.MsgTx, 0, b.N)
	hashes := make([]*bitcoin.Hash32, 0, b.N)
	for i := 0; i < b.N; i++ {
		fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100000+uint64(i), issuerKey.Address)

		// Create Transfer message
		transferAmount := uint64(1)
		transferData := actions.Transfer{}

		assetTransferData := actions.AssetTransferField{
			ContractIndex: 0, // first output
			AssetType:     testAssetType,
			AssetCode:     testAssetCodes[0].Bytes(),
		}

		assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
			&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})

		blockHeight := test.Headers.LastHeight(ctx) - 5
		blockHash, err := test.Headers.Hash(ctx, blockHeight)
		if err != nil {
			b.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
		}
		oracleSigHash, err := protocol.TransferOracleSigHash(ctx, test.ContractKey.Address,
			testAssetCodes[0].Bytes(), userKey.Address, *blockHash, expiry, 1)
		node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
		}
		oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
		if err != nil {
			b.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
		}
		receiver := actions.AssetReceiverField{
			Address:               userKey.Address.Bytes(),
			Quantity:              transferAmount,
			OracleSigAlgorithm:    1,
			OracleConfirmationSig: oracleSig.Bytes(),
			OracleSigBlockHeight:  uint32(blockHeight),
			OracleSigExpiry:       expiry,
		}
		assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, &receiver)

		transferData.Assets = append(transferData.Assets, &assetTransferData)

		// Build transfer transaction
		transferTx := wire.NewMsgTx(1)

		transferInputHash := fundingTx.TxHash()

		// From issuer
		transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

		// To contract
		script, _ := test.ContractKey.Address.LockingScript()
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, script))

		// Data output
		script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
		if err != nil {
			b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
		}
		transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

		test.RPCNode.SaveTX(ctx, transferTx)
		requests = append(requests, transferTx)
		hash := transferTx.TxHash()
		hashes = append(hashes, hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		test.HoldingsChannel)

	if err := server.Load(ctx); err != nil {
		b.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		b.Fatalf("Failed to load wallet : %s", err)
	}

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

	wgInternal := sync.WaitGroup{}
	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
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
	}()

	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		responsesProcessed := 0
		for responsesProcessed < b.N {
			response := getResponse()
			if response == nil {
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

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *response.TxHash()); err != nil {
				b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Wait()
	pprof.StopCPUProfile()
	b.StopTimer()

	// Check balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		b.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if h.FinalizedBalance != 0 {
		b.Fatalf("\t%s\tBalance not zeroized : %d", tests.Failed, h.FinalizedBalance)
	}

	server.Stop(ctx)
	wg.Wait()
}

// splitTransfer creates a transfer transaction that splits the senders balance and sends it to two
//   new users. It returns the transfer tx, and the new user's keys.
func splitTransfer(b *testing.B, ctx context.Context, sender *wallet.Key, balance uint64) (*wire.MsgTx, *wallet.Key, *wallet.Key) {
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 30000+balance, sender.Address)

	receiver1, err := tests.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		b.Fatalf("\t%s\tFailed to generate receiver : %v", tests.Failed, err)
	}

	receiver2, err := tests.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		b.Fatalf("\t%s\tFailed to generate receiver : %v", tests.Failed, err)
	}

	// Create Transfer message
	transferAmount := balance / 2
	transferData := actions.Transfer{}

	// Mock up holding because otherwise we must wait for settlement, which shouldn't count for benchmark
	mockUpHolding(b, ctx, receiver1.Address, transferAmount)
	mockUpHolding(b, ctx, receiver2.Address, transferAmount)

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount * 2})

	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		&actions.AssetReceiverField{
			Address:  receiver1.Address.Bytes(),
			Quantity: transferAmount})

	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		&actions.AssetReceiverField{
			Address:  receiver2.Address.Bytes(),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		b.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	test.RPCNode.SaveTX(ctx, transferTx)

	return transferTx, receiver1, receiver2
}

func splitTransferRecurse(b *testing.B, ctx context.Context, sender *wallet.Key, balance uint64,
	levels uint) ([]*wire.MsgTx, []*wallet.Key) {

	// split this transfer
	tx, receiver1, receiver2 := splitTransfer(b, ctx, sender, balance)

	if levels == 1 {
		return []*wire.MsgTx{tx}, []*wallet.Key{receiver1, receiver2}
	}

	// Create another level
	childCount := (1 << (levels + 1)) - 1
	txs := make([]*wire.MsgTx, 0, childCount+1)
	receivers := make([]*wallet.Key, 0, 2+(2*childCount))

	txs = append(txs, tx)
	receivers = append(receivers, receiver1)
	receivers = append(receivers, receiver2)

	newTxs1, newReceivers1 := splitTransferRecurse(b, ctx, receiver1, balance/2, levels-1)
	txs = append(txs, newTxs1...)
	receivers = append(receivers, newReceivers1...)

	newTxs2, newReceivers2 := splitTransferRecurse(b, ctx, receiver2, balance/2, levels-1)
	txs = append(txs, newTxs2...)
	receivers = append(receivers, newReceivers2...)

	return txs, receivers
}

// treeTransfersBenchmark creates an asset, then splits their balance in half until the number of
//   transfer txs is met.
func treeTransfersBenchmark(b *testing.B) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		b.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(b, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)

	levels := uint(1)
	nodes := (1 << levels) - 1
	for nodes < b.N {
		levels++
		nodes = (1 << levels) - 1
	}
	// fmt.Printf("Using %d tree levels for %d transfers\n", levels, nodes)

	mockUpAsset(b, ctx, true, true, true, uint64(nodes)*2, 0, &sampleAssetPayload, true, false, false)

	requests, _ := splitTransferRecurse(b, ctx, issuerKey, uint64(nodes)*2, levels)
	hashes := make([]*bitcoin.Hash32, 0, nodes)
	for _, request := range requests {
		hash := request.TxHash()
		hashes = append(hashes, hash)
	}

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		test.HoldingsChannel)

	if err := server.Load(ctx); err != nil {
		b.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		b.Fatalf("Failed to load wallet : %s", err)
	}

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

	profFile, err := os.OpenFile("tree_transfer_cpu.prof", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		b.Fatalf("\t%s\tFailed to create prof file : %v", tests.Failed, err)
	}
	err = pprof.StartCPUProfile(profFile)
	if err != nil {
		b.Fatalf("\t%s\tFailed to start prof : %v", tests.Failed, err)
	}
	b.ResetTimer()

	wgInternal := sync.WaitGroup{}
	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		for i, request := range requests {
			if _, err := server.HandleTx(ctx, request); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *hashes[i]); err != nil {
				b.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
			}

			if i >= b.N {
				break
			}
		}
	}()

	wgInternal.Add(1)
	go func() {
		defer wgInternal.Done()
		responsesProcessed := 0
		for responsesProcessed < b.N {
			response := getResponse()
			if response == nil {
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

			if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *response.TxHash()); err != nil {
				b.Fatalf("\t%s\tSettlement handle failed : %v", tests.Failed, err)
			}
		}
	}()

	wgInternal.Wait()
	pprof.StopCPUProfile()
	b.StopTimer()

	server.Stop(ctx)
	wg.Wait()

	// Check balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	h, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address, &testAssetCodes[0],
		issuerKey.Address, v.Now)
	if err != nil {
		b.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if h.FinalizedBalance != 0 {
		b.Fatalf("\t%s\tBalance not zeroized : %d", tests.Failed, h.FinalizedBalance)
	}
}

func sendTokens(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	test.HoldingsChannel.Open(10)
	go func() {
		if err := holdings.ProcessCacheItems(ctx, test.MasterDB, test.HoldingsChannel); err != nil {
			node.LogError(ctx, "Process holdings cache failed : %s", err)
		}
		node.LogVerbose(ctx, "Process holdings cache thread finished")
	}()

	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	// Let holdings cache update
	time.Sleep(500 * time.Millisecond)
	holdings.Reset(ctx) // Clear cache

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)

	// Create Transfer message
	transferAmount := uint64(750)
	transferData := actions.Transfer{}

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(600, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
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
	t.Logf("\tUnderfunded asset transfer : %s", transferTx.TxHash().String())

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

	// Let holdings cache update
	time.Sleep(500 * time.Millisecond)

	// Check issuer and user balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.PendingBalance != 1000 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.PendingBalance, 1000)
	}

	// Adjust amount to contract to be low, but enough for a reject
	transferTx.TxOut[0].Value = 1000
	t.Logf("\tLow funding asset transfer : %s", transferTx.TxHash().String())

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx, test.NodeConfig.IsTest)
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
	if err != node.ErrRejected {
		t.Fatalf("\t%s\tFailed to reject transfer with insufficient funds : %v", tests.Failed, err)
	}

	checkResponse(t, "M2")

	t.Logf("\t%s\tUnderfunded asset transfer rejected with response", tests.Success)

	// Let holdings cache update
	time.Sleep(500 * time.Millisecond)

	// Check issuer and user balance
	issuerHolding, err = holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.PendingBalance != 1000 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.PendingBalance, 1000)
	}

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 2500
	t.Logf("\tFunded asset transfer : %s", transferTx.TxHash().String())

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

	var responseMsg actions.Action
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tResponse doesn't contain tokenized op return", tests.Failed)
	}

	settlement, ok := responseMsg.(*actions.Settlement)
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
	issuerHolding, err = holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)

	// Let holdings cache update
	time.Sleep(500 * time.Millisecond)

	// Send a second transfer ----------------------------------------------------------------------
	fundingTx2 := tests.MockFundingTx(ctx, test.RPCNode, 100022, issuerKey.Address)

	// Build transfer transaction
	transferTx2 := wire.NewMsgTx(1)

	transferInputHash = fundingTx2.TxHash()

	// From issuer
	transferTx2.TxIn = append(transferTx2.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	transferTx2.TxOut = append(transferTx2.TxOut, wire.NewTxOut(2500, script))

	// Data output
	transferData.Assets[0].AssetSenders[0].Quantity = 250
	transferData.Assets[0].AssetReceivers[0].Quantity = 250
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
	t.Logf("\tSecond asset transfer : %s", transferTx2.TxHash().String())

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

	settlement, ok = responseMsg.(*actions.Settlement)
	if !ok {
		t.Fatalf("\t%s\tResponse isn't a settlement", tests.Failed)
	}

	if settlement.Assets[0].Settlements[0].Quantity != 0 {
		t.Fatalf("\t%s\tIssuer token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[0].Quantity, 0)
	}

	if settlement.Assets[0].Settlements[1].Quantity != 1000 {
		t.Fatalf("\t%s\tUser token settlement balance incorrect : %d != %d", tests.Failed,
			settlement.Assets[0].Settlements[1].Quantity, 1000)
	}

	// Check issuer and user balance
	issuerHolding, err = holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != 0 {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, 0)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err = holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.FinalizedBalance != 1000 {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.FinalizedBalance, 1000)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.FinalizedBalance)

	test.HoldingsChannel.Close()
}

func multiExchange(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	user1HoldingBalance := uint64(100)
	mockUpHolding(t, ctx, userKey.Address, user1HoldingBalance)

	mockUpContract2(t, ctx, "Test Contract 2", "This is a mock contract and means nothing.", "I",
		1, "Karl Bitcoin", true, true, false, false, false)
	mockUpAsset2(t, ctx, true, true, true, 1500, &sampleAssetPayload2, true, false, false)

	user2HoldingBalance := uint64(200)
	mockUpHolding2(t, ctx, user2Key.Address, user2HoldingBalance)

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address)

	// Create Transfer message
	transferData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transfer1Amount := uint64(50)
	assetTransfer1Data := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(150)
	assetTransfer2Data := actions.AssetTransferField{
		ContractIndex: 1, // first output
		AssetType:     testAsset2Type,
		AssetCode:     testAsset2Code.Bytes(),
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transfer2Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer2Data)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From user1
	transferInputHash := funding1Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// From user2
	transferInputHash = funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// To contract1
	script1, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, script1))
	// To contract2
	script2, _ := test.Contract2Key.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(1000, script2))
	// To contract1 boomerang
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(5000, script1))

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

	if tracer.Count() != 1 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 1)
	}

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

	if responseItx.MsgProto.Code() != "M1" {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	settlementRequestMessage, ok := responseItx.MsgProto.(*actions.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if settlementRequestMessage.MessageCode != messages.CodeSettlementRequest {
		t.Fatalf("\t%s\tResponse itx is not Settlement Request : %d", tests.Failed,
			settlementRequestMessage.MessageCode)
	}

	test.RPCNode.SaveTX(ctx, response)

	err = a.Trigger(ctx, "SEE", responseItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to process response : %v", tests.Failed, err)
	}

	t.Logf("\t%s\tSettlement request accepted", tests.Success)

	if tracer.Count() != 1 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 1)
	}

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

	if responseItx.MsgProto.Code() != "M1" {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	signatureRequestMessage, ok := responseItx.MsgProto.(*actions.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if signatureRequestMessage.MessageCode != messages.CodeSignatureRequest {
		t.Fatalf("\t%s\tResponse itx is not Signature Request : %d", tests.Failed,
			signatureRequestMessage.MessageCode)
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
	v := ctx.Value(node.KeyValues).(*node.Values)
	user1Holding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user1Holding.FinalizedBalance != user1HoldingBalance-transfer1Amount {
		t.Fatalf("\t%s\tUser 1 token 1 balance incorrect : %d != %d", tests.Failed,
			user1Holding.FinalizedBalance, user1HoldingBalance-transfer1Amount)
	}

	t.Logf("\t%s\tUser 1 token 1 balance : %d", tests.Success, user1Holding.FinalizedBalance)

	user2Holding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], user2Key.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user2Holding.FinalizedBalance != transfer1Amount {
		t.Fatalf("\t%s\tUser 2 token 1 balance incorrect : %d != %d", tests.Failed,
			user2Holding.FinalizedBalance, transfer1Amount)
	}

	t.Logf("\t%s\tUser 2 token 1 balance : %d", tests.Success, user2Holding.FinalizedBalance)

	user1Holding, err = holdings.GetHolding(ctx, test.MasterDB, test.Contract2Key.Address,
		&testAsset2Code, userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}

	if user1Holding.FinalizedBalance != transfer2Amount {
		t.Fatalf("\t%s\tUser 1 token 2 balance incorrect : %d != %d", tests.Failed,
			user1Holding.FinalizedBalance, transfer2Amount)
	}

	t.Logf("\t%s\tUser 1 token 2 balance : %d", tests.Success, user1Holding.FinalizedBalance)

	user2Holding, err = holdings.GetHolding(ctx, test.MasterDB, test.Contract2Key.Address,
		&testAsset2Code, user2Key.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user2Holding.FinalizedBalance != user2HoldingBalance-transfer2Amount {
		t.Fatalf("\t%s\tUser 2 token 2 balance incorrect : %d != %d", tests.Failed,
			user2Holding.FinalizedBalance, user2HoldingBalance-transfer2Amount)
	}

	t.Logf("\t%s\tUser 2 token 2 balance : %d", tests.Success, user2Holding.FinalizedBalance)

	if tracer.Count() != 0 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 0)
	}
}

func bitcoinExchange(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	user1HoldingBalance := uint64(100)
	mockUpHolding(t, ctx, userKey.Address, user1HoldingBalance)

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address)

	// Create Transfer message
	transferData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transfer1Amount := uint64(50)
	assetTransfer1Data := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(1050)
	assetTransfer2Data := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     "BSV",
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transfer2Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer2Data)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From user1
	transferInputHash := funding1Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// From user2
	transferInputHash = funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// To contract1
	script1, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3200, script1))

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
	response := checkResponse(t, "T2")

	// Check issuer and user balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	user1Holding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user1Holding.FinalizedBalance != user1HoldingBalance-transfer1Amount {
		t.Fatalf("\t%s\tUser 1 token 1 balance incorrect : %d != %d", tests.Failed,
			user1Holding.FinalizedBalance, user1HoldingBalance-transfer1Amount)
	}

	t.Logf("\t%s\tUser 1 token 1 balance : %d", tests.Success, user1Holding.FinalizedBalance)

	user2Holding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], user2Key.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if user2Holding.FinalizedBalance != transfer1Amount {
		t.Fatalf("\t%s\tUser 2 token 1 balance incorrect : %d != %d", tests.Failed,
			user2Holding.FinalizedBalance, transfer1Amount)
	}

	t.Logf("\t%s\tUser 2 token 1 balance : %d", tests.Success, user2Holding.FinalizedBalance)

	if len(response.TxOut) != 4 {
		t.Fatalf("Incorrect output count : want %d, got %d\n%s\n", 4, len(response.TxOut),
			response.StringWithAddresses(test.NodeConfig.Net))
	}

	t.Logf("Response : \n%s\n", response.StringWithAddresses(test.NodeConfig.Net))

	found := false
	for _, output := range response.TxOut {
		ad, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}

		if ad.Equal(userKey.Address) {
			found = true
			if output.Value != transfer2Amount {
				t.Fatalf("Incorrect bitcoin receive amount : want %d, got %d", transfer2Amount,
					output.Value)
			}

			t.Logf("\t%s\tUser 2 bitcoin balance : %d", tests.Success, output.Value)
			break
		}
	}

	if !found {
		t.Fatalf("Bitcoin receive output not found")
	}
}

func multiExchangeLock(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}

	if tracer.Count() != 0 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 0)
	}

	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I", 1,
		"John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	user1HoldingBalance := uint64(150)
	mockUpHolding(t, ctx, userKey.Address, user1HoldingBalance)

	otherContractKey, err := tests.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("\t%s\tFailed to generate other contract key : %v", tests.Failed, err)
	}
	mockUpOtherContract(t, ctx, otherContractKey.Address, "Test Contract 2",
		"This is a mock contract and means nothing.", "I", 1, "Karl Bitcoin", true, true, false,
		false, false)
	mockUpOtherAsset(t, ctx, otherContractKey, true, true, true, 1500, &sampleAssetPayload2, true,
		false, false)

	user2HoldingBalance := uint64(200)
	mockUpOtherHolding(t, ctx, otherContractKey, user2Key.Address, user2HoldingBalance)

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address)

	// Create Transfer message
	transferData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transfer1Amount := uint64(50)
	assetTransfer1Data := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(150)
	assetTransfer2Data := actions.AssetTransferField{
		ContractIndex: 1, // first output
		AssetType:     testAsset2Type,
		AssetCode:     testAsset2Code.Bytes(),
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transfer2Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer2Data)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From user1
	transferInputHash := funding1Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// From user2
	transferInputHash = funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// To contract1
	script1, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, script1))
	// To contract2
	script2, _ := otherContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(1000, script2))
	// To contract1 boomerang
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(5000, script1))

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

	t.Logf("\t%s\tMulti-contract transfer accepted", tests.Success)

	if tracer.Count() != 1 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 1)
	}

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

	if responseItx.MsgProto.Code() != actions.CodeMessage {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	settlementRequestMessage, ok := responseItx.MsgProto.(*actions.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if settlementRequestMessage.MessageCode != messages.CodeSettlementRequest {
		t.Fatalf("\t%s\tResponse itx is not Settlement Request : %d", tests.Failed,
			settlementRequestMessage.MessageCode)
	}

	t.Logf("\t%s\tMulti-contract transfer response is a settlement request", tests.Success)

	test.RPCNode.SaveTX(ctx, response)

	/****************************** Attempt single contract transfer ******************************/
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100014, userKey.Address)

	// Create Transfer message
	transferOtherData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transferOtherAmount := uint64(75)
	assetTransferOtherData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferOtherData.AssetSenders = append(assetTransferOtherData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferOtherAmount})
	assetTransferOtherData.AssetReceivers = append(assetTransferOtherData.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transferOtherAmount})

	transferOtherData.Assets = append(transferOtherData.Assets, &assetTransferOtherData)

	// Build transfer transaction
	transferOtherTx := wire.NewMsgTx(1)

	// From user
	transferOtherTx.TxIn = append(transferOtherTx.TxIn,
		wire.NewTxIn(wire.NewOutPoint(fundingTx.TxHash(), 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&transferOtherData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(0, script))

	transferOtherItx, err := inspector.NewTransactionFromWire(ctx, transferOtherTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferOtherItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, transferOtherTx)

	err = a.Trigger(ctx, "SEE", transferOtherItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject transfer of locked funds", tests.Failed)
	}

	t.Logf("\t%s\tIntermediate transfer rejected", tests.Success)

	if tracer.Count() != 1 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 1)
	}

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create transfer response", tests.Failed)
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

	if responseItx.MsgProto.Code() != actions.CodeRejection {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	rejectMessage, ok := responseItx.MsgProto.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if rejectMessage.RejectionCode != actions.RejectionsHoldingsLocked {
		t.Fatalf("\t%s\tReject code is not holdings locked : %d", tests.Failed,
			rejectMessage.RejectionCode)
	}

	t.Logf("\t%s\tIntermediate transfer rejected with locked code", tests.Success)

	/***************************** Send cancel from other contract ********************************/
	// Create reject message
	rejectOtherData := actions.Rejection{
		RejectionCode:  actions.RejectionsAssetNotPermitted,
		Timestamp:      uint64(time.Now().UnixNano()),
		AddressIndexes: []uint32{0},
	}

	// Build transfer transaction
	rejectOtherTx := wire.NewMsgTx(1)

	// From other contract (second output of multi-contract transfer request)
	rejectOtherTx.TxIn = append(rejectOtherTx.TxIn,
		wire.NewTxIn(wire.NewOutPoint(transferTx.TxHash(), 1), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	rejectOtherTx.TxOut = append(rejectOtherTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&rejectOtherData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize rejection : %v", tests.Failed, err)
	}
	rejectOtherTx.TxOut = append(rejectOtherTx.TxOut, wire.NewTxOut(0, script))

	rejectOtherItx, err := inspector.NewTransactionFromWire(ctx, rejectOtherTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = rejectOtherItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, rejectOtherTx)

	err = a.Trigger(ctx, "SEE", rejectOtherItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to handle reject of multi-contract transfer", tests.Failed)
	}

	t.Logf("\t%s\tMulti-contract transfer cancel processed", tests.Success)

	if tracer.Count() != 0 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 0)
	}

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create transfer reject", tests.Failed)
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

	if responseItx.MsgProto.Code() != actions.CodeRejection {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	rejectMessage, ok = responseItx.MsgProto.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if rejectMessage.RejectionCode != actions.RejectionsAssetNotPermitted {
		t.Fatalf("\t%s\tReject code is not holdings locked : %d", tests.Failed,
			rejectMessage.RejectionCode)
	}

	t.Logf("\t%s\tMulti-contract transfer cancel response processed", tests.Success)

	/******************* Check that a transfer can now process successfully ***********************/
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100018, userKey.Address)

	// Create Transfer message
	transferOtherData = actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transferOtherAmount = uint64(100)
	assetTransferOtherData = actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferOtherData.AssetSenders = append(assetTransferOtherData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferOtherAmount})
	assetTransferOtherData.AssetReceivers = append(assetTransferOtherData.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transferOtherAmount})

	transferOtherData.Assets = append(transferOtherData.Assets, &assetTransferOtherData)

	// Build transfer transaction
	transferOtherTx = wire.NewMsgTx(1)

	// From user
	transferOtherTx.TxIn = append(transferOtherTx.TxIn,
		wire.NewTxIn(wire.NewOutPoint(fundingTx.TxHash(), 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&transferOtherData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(0, script))

	transferOtherItx, err = inspector.NewTransactionFromWire(ctx, transferOtherTx,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferOtherItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, transferOtherTx)

	err = a.Trigger(ctx, "SEE", transferOtherItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer after funds should be unlocked : %s",
			tests.Failed, err)
	}

	// Check the response
	checkResponse(t, "T2")

	t.Logf("\t%s\tFollow up transfer accepted", tests.Success)

	if tracer.Count() != 0 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 0)
	}
}

func multiExchangeTimeout(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	user1HoldingBalance := uint64(150)
	mockUpHolding(t, ctx, userKey.Address, user1HoldingBalance)

	otherContractKey, err := tests.GenerateKey(test.NodeConfig.Net)
	if err != nil {
		t.Fatalf("\t%s\tFailed to generate other contract key : %v", tests.Failed, err)
	}
	mockUpOtherContract(t, ctx, otherContractKey.Address,
		"Test Contract 2", "This is a mock contract and means nothing.", "I",
		1, "Karl Bitcoin", true, true, false, false, false)
	mockUpOtherAsset(t, ctx, otherContractKey, true, true, true, 1500, &sampleAssetPayload2,
		true, false, false)

	user2HoldingBalance := uint64(200)
	mockUpOtherHolding(t, ctx, otherContractKey, user2Key.Address, user2HoldingBalance)

	funding1Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, userKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 100012, user2Key.Address)

	// Create Transfer message
	transferData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transfer1Amount := uint64(50)
	assetTransfer1Data := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransfer1Data.AssetSenders = append(assetTransfer1Data.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transfer1Amount})
	assetTransfer1Data.AssetReceivers = append(assetTransfer1Data.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transfer1Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer1Data)

	// Transfer asset 2 from user2 to user1
	transfer2Amount := uint64(150)
	assetTransfer2Data := actions.AssetTransferField{
		ContractIndex: 1, // first output
		AssetType:     testAsset2Type,
		AssetCode:     testAsset2Code.Bytes(),
	}

	assetTransfer2Data.AssetSenders = append(assetTransfer2Data.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: transfer2Amount})
	assetTransfer2Data.AssetReceivers = append(assetTransfer2Data.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transfer2Amount})

	transferData.Assets = append(transferData.Assets, &assetTransfer2Data)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From user1
	transferInputHash := funding1Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// From user2
	transferInputHash = funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// To contract1
	script1, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(3000, script1))
	// To contract2
	script2, _ := otherContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(1000, script2))
	// To contract1 boomerang
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(5000, script1))

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

	t.Logf("\t%s\tMulti-contract transfer accepted", tests.Success)

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

	if responseItx.MsgProto.Code() != actions.CodeMessage {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	settlementRequestMessage, ok := responseItx.MsgProto.(*actions.Message)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if settlementRequestMessage.MessageCode != messages.CodeSettlementRequest {
		t.Fatalf("\t%s\tResponse itx is not Settlement Request : %d", tests.Failed,
			settlementRequestMessage.MessageCode)
	}

	t.Logf("\t%s\tMulti-contract transfer response is a settlement request", tests.Success)

	test.RPCNode.SaveTX(ctx, response)

	/****************************** Attempt single contract transfer ******************************/
	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100014, userKey.Address)

	// Create Transfer message
	transferOtherData := actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transferOtherAmount := uint64(75)
	assetTransferOtherData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferOtherData.AssetSenders = append(assetTransferOtherData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferOtherAmount})
	assetTransferOtherData.AssetReceivers = append(assetTransferOtherData.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transferOtherAmount})

	transferOtherData.Assets = append(transferOtherData.Assets, &assetTransferOtherData)

	// Build transfer transaction
	transferOtherTx := wire.NewMsgTx(1)

	// From user
	transferOtherTx.TxIn = append(transferOtherTx.TxIn,
		wire.NewTxIn(wire.NewOutPoint(fundingTx.TxHash(), 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&transferOtherData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(0, script))

	transferOtherItx, err := inspector.NewTransactionFromWire(ctx, transferOtherTx, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferOtherItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, transferOtherTx)

	err = a.Trigger(ctx, "SEE", transferOtherItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to reject transfer of locked funds", tests.Failed)
	}

	t.Logf("\t%s\tIntermediate transfer rejected", tests.Success)

	if tracer.Count() != 1 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 1)
	}

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create transfer response", tests.Failed)
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

	if responseItx.MsgProto.Code() != actions.CodeRejection {
		t.Fatalf("\t%s\tResponse itx is not M1 : %v", tests.Failed, err)
	}

	rejectMessage, ok := responseItx.MsgProto.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if rejectMessage.RejectionCode != actions.RejectionsHoldingsLocked {
		t.Fatalf("\t%s\tReject code is not holdings locked : %d", tests.Failed,
			rejectMessage.RejectionCode)
	}

	t.Logf("\t%s\tIntermediate transfer rejected with locked code", tests.Success)

	/************************************* Time out transfer **************************************/
	err = a.Trigger(ctx, "END", transferItx)
	if err == nil {
		t.Fatalf("\t%s\tFailed to time out transfer", tests.Failed)
	}

	t.Logf("\t%s\tMulti-contract transfer timed out", tests.Success)

	if len(responses) == 0 {
		t.Fatalf("\t%s\tFailed to create transfer response", tests.Failed)
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

	if responseItx.MsgProto.Code() != actions.CodeRejection {
		t.Fatalf("\t%s\tResponse itx is not M2 : %v", tests.Failed, err)
	}

	timeoutMessage, ok := responseItx.MsgProto.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tResponse itx is not Message", tests.Failed)
	}

	if timeoutMessage.RejectionCode != actions.RejectionsTimeout {
		t.Fatalf("\t%s\tResponse reject code is not timeout : %d", tests.Failed,
			timeoutMessage.RejectionCode)
	}

	test.RPCNode.SaveTX(ctx, response)

	/******************* Check that a transfer can now process successfully ***********************/
	fundingTx = tests.MockFundingTx(ctx, test.RPCNode, 100018, userKey.Address)

	// Create Transfer message
	transferOtherData = actions.Transfer{}

	// Transfer asset 1 from user1 to user2
	transferOtherAmount = uint64(100)
	assetTransferOtherData = actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferOtherData.AssetSenders = append(assetTransferOtherData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferOtherAmount})
	assetTransferOtherData.AssetReceivers = append(assetTransferOtherData.AssetReceivers,
		&actions.AssetReceiverField{Address: user2Key.Address.Bytes(),
			Quantity: transferOtherAmount})

	transferOtherData.Assets = append(transferOtherData.Assets, &assetTransferOtherData)

	// Build transfer transaction
	transferOtherTx = wire.NewMsgTx(1)

	// From user
	transferOtherTx.TxIn = append(transferOtherTx.TxIn,
		wire.NewTxIn(wire.NewOutPoint(fundingTx.TxHash(), 0), make([]byte, 130)))

	// To contract
	script, _ = test.ContractKey.Address.LockingScript()
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(3000, script))

	// Data output
	script, err = protocol.Serialize(&transferOtherData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferOtherTx.TxOut = append(transferOtherTx.TxOut, wire.NewTxOut(0, script))

	transferOtherItx, err = inspector.NewTransactionFromWire(ctx, transferOtherTx,
		test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferOtherItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.SaveTX(ctx, transferOtherTx)

	err = a.Trigger(ctx, "SEE", transferOtherItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer after funds should be unlocked : %s",
			tests.Failed, err)
	}

	// Check the response
	checkResponse(t, "T2")

	t.Logf("\t%s\tFollow up transfer accepted", tests.Success)

	if tracer.Count() != 0 {
		t.Errorf("\t%s\tWrong tracer count : got %d, want %d", tests.Failed, tracer.Count(), 0)
	}
}

func oracleTransfer(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContractWithOracle(t, ctx, "Test Contract", "This is a mock contract and means nothing.",
		"I", 1, "John Bitcoin")
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	err := test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := actions.Transfer{}

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})

	blockHeight := test.Headers.LastHeight(ctx) - 5
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, test.ContractKey.Address,
		testAssetCodes[0].Bytes(), userKey.Address, *blockHash, expiry, 1)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := actions.AssetReceiverField{
		Address:               userKey.Address.Bytes(),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig.Bytes(),
		OracleSigBlockHeight:  uint32(blockHeight),
		OracleSigExpiry:       expiry,
	}
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, &receiver)

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	holdingsChannel := &holdings.CacheChannel{}
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		holdingsChannel)

	if err := server.Load(ctx); err != nil {
		t.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		t.Fatalf("Failed to load wallet : %s", err)
	}

	server.SetAlternateResponder(respondTx)
	server.SetInSync()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			t.Logf("Server failed : %s", err)
		}
	}()

	time.Sleep(time.Second)

	t.Logf("Transfer tx : %s", transferTx.TxHash().String())
	if _, err := server.HandleTx(ctx, transferTx); err != nil {
		t.Fatalf("\t%s\tTransfer handle failed : %v", tests.Failed, err)
	}

	if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *transferTx.TxHash()); err != nil {
		t.Fatalf("\t%s\tTransfer handle state failed : %v", tests.Failed, err)
	}

	var firstResponse *wire.MsgTx // Request tx is re-broadcast now
	var response *wire.MsgTx
	for {
		if firstResponse == nil {
			firstResponse = getResponse()
			time.Sleep(time.Millisecond)
			continue
		}
		response = getResponse()
		if response != nil {
			break
		}

		time.Sleep(time.Millisecond)
	}

	var responseMsg actions.Action
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tTransfer response doesn't contain tokenized op return", tests.Failed)
	}
	if responseMsg.Code() != "T2" {
		t.Fatalf("\t%s\tTransfer response not a settlement : %s", tests.Failed, responseMsg.Code())
	}

	// Check issuer and user balance
	v := ctx.Value(node.KeyValues).(*node.Values)
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.PendingBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.PendingBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.PendingBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if userHolding.PendingBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed,
			userHolding.PendingBalance, transferAmount)
	}

	t.Logf("\t%s\tUser asset balance : %d", tests.Success, userHolding.PendingBalance)
}

func oracleTransferBad(t *testing.T) {
	ctx := test.Context

	if err := resetTest(ctx); err != nil {
		t.Fatalf("\t%s\tFailed to reset test : %v", tests.Failed, err)
	}
	mockUpContractWithOracle(t, ctx, "Test Contract", "This is a mock contract and means nothing.",
		"I", 1, "John Bitcoin")
	mockUpAsset(t, ctx, true, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	err := test.Headers.Populate(ctx, 50000, 12)
	if err != nil {
		t.Fatalf("\t%s\tFailed to mock up headers : %v", tests.Failed, err)
	}

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100016, issuerKey.Address)
	bitcoinFundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100017, userKey.Address)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := actions.Transfer{}

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})

	blockHeight := test.Headers.LastHeight(ctx) - 4
	blockHash, err := test.Headers.Hash(ctx, blockHeight)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve header hash : %v", tests.Failed, err)
	}
	expiry := uint64(time.Now().Add(1 * time.Hour).UnixNano())
	oracleSigHash, err := protocol.TransferOracleSigHash(ctx, test.ContractKey.Address,
		testAssetCodes[0].Bytes(), userKey.Address, *blockHash, expiry, 0)
	node.LogVerbose(ctx, "Created oracle sig hash from block : %s", blockHash.String())
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle sig hash : %v", tests.Failed, err)
	}
	oracleSig, err := oracleKey.Key.Sign(oracleSigHash)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create oracle signature : %v", tests.Failed, err)
	}
	receiver := actions.AssetReceiverField{
		Address:               userKey.Address.Bytes(),
		Quantity:              transferAmount,
		OracleSigAlgorithm:    1,
		OracleConfirmationSig: oracleSig.Bytes(),
		OracleSigBlockHeight:  uint32(blockHeight),
		OracleSigExpiry:       expiry,
	}
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, &receiver)

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	bitcoinTransferAmount := uint64(50000)
	bitcoinTransferData := actions.AssetTransferField{
		ContractIndex: uint32(0x0000ffff),
		AssetType:     "BSV",
	}

	bitcoinTransferData.AssetSenders = append(bitcoinTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: bitcoinTransferAmount})

	bitcoinTransferData.AssetReceivers = append(bitcoinTransferData.AssetReceivers,
		&actions.AssetReceiverField{
			Address:  issuerKey.Address.Bytes(),
			Quantity: bitcoinTransferAmount,
		})

	transferData.Assets = append(transferData.Assets, &bitcoinTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From issuer
	transferInputHash := fundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0),
		make([]byte, 130)))

	// From user
	bitcoinInputHash := bitcoinFundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(bitcoinInputHash, 0),
		make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(52000, script))

	// Data output
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	test.RPCNode.SaveTX(ctx, transferTx)
	t.Logf("Transfer Tx : %s", transferTx.TxHash().String())

	test.NodeConfig.PreprocessThreads = 4

	tracer := filters.NewTracer()
	holdingsChannel := &holdings.CacheChannel{}
	txFilter := filters.NewTxFilter(tracer, true)
	test.Scheduler = &scheduler.Scheduler{}

	server := listeners.NewServer(test.Wallet, a, &test.NodeConfig, test.MasterDB,
		test.RPCNode, nil, test.Headers, test.Scheduler, tracer, test.UTXOs, txFilter,
		holdingsChannel)

	if err := server.Load(ctx); err != nil {
		t.Fatalf("Failed to load server : %s", err)
	}

	if err := server.SyncWallet(ctx); err != nil {
		t.Fatalf("Failed to load wallet : %s", err)
	}

	server.SetAlternateResponder(respondTx)
	server.SetInSync()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			t.Logf("Server failed : %s", err)
		}
	}()

	time.Sleep(time.Second)

	if _, err := server.HandleTx(ctx, transferTx); err != nil {
		t.Fatalf("\t%s\tContract handle failed : %v", tests.Failed, err)
	}

	if err := server.HandleTxState(ctx, spynodeHandlers.ListenerMsgTxStateSafe, *transferTx.TxHash()); err != nil {
		t.Fatalf("\t%s\tContract offer handle state failed : %v", tests.Failed, err)
	}

	var firstResponse *wire.MsgTx // Request tx is re-broadcast now
	var response *wire.MsgTx
	for {
		if firstResponse == nil {
			firstResponse = getResponse()
			time.Sleep(time.Millisecond)
			continue
		}
		response = getResponse()
		if response != nil {
			break
		}

		time.Sleep(time.Millisecond)
	}

	var responseMsg actions.Action
	for _, output := range response.TxOut {
		responseMsg, err = protocol.Deserialize(output.PkScript, test.NodeConfig.IsTest)
		if err == nil {
			break
		}
	}
	if responseMsg == nil {
		t.Fatalf("\t%s\tTransfer response doesn't contain tokenized op return", tests.Failed)
	}
	if responseMsg.Code() != "M2" {
		t.Fatalf("\t%s\tTransfer response not a reject : %s", tests.Failed, responseMsg.Code())
	}
	reject, ok := responseMsg.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert response to rejection", tests.Failed)
	}
	if reject.RejectionCode != actions.RejectionsInvalidSignature {
		t.Fatalf("\t%s\tWrong reject code for Transfer reject", tests.Failed)
	}

	// Find refund output
	found := false
	for _, output := range response.TxOut {
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}
		if address.Equal(userKey.Address) && output.Value >= bitcoinTransferAmount {
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
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, false, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := actions.Transfer{}

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	transferInputHash := fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2500, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
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
	issuerHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], issuerKey.Address, v.Now)
	if err != nil {
		t.Fatalf("\t%s\tFailed to get holding : %s", tests.Failed, err)
	}
	if issuerHolding.FinalizedBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed,
			issuerHolding.FinalizedBalance, testTokenQty-transferAmount)
	}

	t.Logf("\t%s\tIssuer asset balance : %d", tests.Success, issuerHolding.FinalizedBalance)

	userHolding, err := holdings.GetHolding(ctx, test.MasterDB, test.ContractKey.Address,
		&testAssetCodes[0], userKey.Address, v.Now)
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
	mockUpContract(t, ctx, "Test Contract", "This is a mock contract and means nothing.", "I",
		1, "John Bitcoin", true, true, false, false, false)
	mockUpAsset(t, ctx, false, true, true, 1000, 0, &sampleAssetPayload, true, false, false)

	user2Holding := uint64(100)
	mockUpHolding(t, ctx, user2Key.Address, user2Holding)

	fundingTx := tests.MockFundingTx(ctx, test.RPCNode, 100012, issuerKey.Address)
	funding2Tx := tests.MockFundingTx(ctx, test.RPCNode, 256, user2Key.Address)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := actions.Transfer{}

	assetTransferData := actions.AssetTransferField{
		ContractIndex: 0, // first output
		AssetType:     testAssetType,
		AssetCode:     testAssetCodes[0].Bytes(),
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders,
		&actions.QuantityIndexField{Index: 1, Quantity: user2Holding})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers,
		&actions.AssetReceiverField{Address: userKey.Address.Bytes(),
			Quantity: transferAmount + user2Holding})

	transferData.Assets = append(transferData.Assets, &assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(1)

	// From issuer
	transferInputHash := fundingTx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transferInputHash, 0), make([]byte, 130)))
	// From user2
	transfer2InputHash := funding2Tx.TxHash()
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(transfer2InputHash, 0), make([]byte, 130)))

	// To contract
	script, _ := test.ContractKey.Address.LockingScript()
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(2000, script))

	// Data output
	var err error
	script, err = protocol.Serialize(&transferData, test.NodeConfig.IsTest)
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

	reject, ok := rejectItx.MsgProto.(*actions.Rejection)
	if !ok {
		t.Fatalf("\t%s\tFailed to convert reject data", tests.Failed)
	}

	if reject.RejectionCode != actions.RejectionsAssetNotPermitted {
		t.Fatalf("\t%s\tRejection code incorrect : %d", tests.Failed, reject.RejectionCode)
	}

	t.Logf("\t%s\tVerified rejection code", tests.Success)
}
