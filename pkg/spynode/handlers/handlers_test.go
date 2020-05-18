package handlers

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	handlerStorage "github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

func TestHandlers(test *testing.T) {
	testBlockCount := 10
	reorgDepth := 5

	// Setup context
	logConfig := logger.NewDevelopmentConfig()
	logConfig.IsText = true
	ctx := logger.ContextWithLogConfig(context.Background(), logConfig)

	// For logging to test from within functions
	ctx = context.WithValue(ctx, 999, test)
	// Use this to get the test value from within non-test code.
	// testValue := ctx.Value(999)
	// test, ok := testValue.(*testing.T)
	// if ok {
	// test.Logf("Test Debug Message")
	// }

	// Setup storage
	storageConfig := storage.NewConfig("standalone", "./tmp/test")
	store := storage.NewFilesystemStorage(storageConfig)

	// Setup config
	startHash, err := bitcoin.NewHash32FromStr("0000000000000000000000000000000000000000000000000000000000000000")
	config, err := data.NewConfig(bitcoin.MainNet, "test", "Tokenized Test", startHash.String(), 8, 2000, 10)
	if err != nil {
		test.Errorf("Failed to create config : %v", err)
	}

	// Setup state
	state := data.NewState()
	state.SetStartHeight(1)

	// Create peer repo
	peerRepo := handlerStorage.NewPeerRepository(store)
	if err := peerRepo.Load(ctx); err != nil {
		test.Errorf("Failed to initialize peer repo : %v", err)
	}

	// Create block repo
	t := uint32(time.Now().Unix())
	blockRepo := handlerStorage.NewBlockRepository(config, store)
	if err := blockRepo.Initialize(ctx, t); err != nil {
		test.Errorf("Failed to initialize block repo : %v", err)
	}

	// Create tx repo
	txRepo := handlerStorage.NewTxRepository(store)
	// Clear any pre-existing data
	for i := 0; i <= testBlockCount; i++ {
		txRepo.ClearBlock(ctx, i)
	}

	// Create reorg repo
	reorgRepo := handlerStorage.NewReorgRepository(store)

	// TxTracker
	txTracker := data.NewTxTracker()

	// Create mempool
	memPool := data.NewMemPool()

	// Setup listeners
	testListener := TestListener{test: test, state: state, blocks: blockRepo, txs: txRepo,
		height: 0, txTracker: txTracker}
	listeners := []Listener{&testListener}

	confTxChannel := TxChannel{}
	confTxChannel.Open(100)

	unconfTxChannel := TxChannel{}
	unconfTxChannel.Open(100)

	txStateChannel := TxStateChannel{}
	txStateChannel.Open(100)

	// Create handlers
	testHandlers := NewTrustedCommandHandlers(ctx, config, state, peerRepo, blockRepo, nil, txRepo,
		reorgRepo, txTracker, memPool, &unconfTxChannel, &txStateChannel, listeners, nil)

	test.Logf("Testing Blocks")

	// Build a bunch of headers
	blocks := make([]*wire.MsgBlock, 0, testBlockCount)
	txs := make([]*wire.MsgTx, 0, testBlockCount)
	headersMsg := wire.NewMsgHeaders()
	zeroHash, _ :=
		bitcoin.NewHash32FromStr("0000000000000000000000000000000000000000000000000000000000000000")
	previousHash, err := blockRepo.Hash(ctx, 0)
	if err != nil {
		test.Errorf("Failed to get genesis hash : %s", err)
		return
	}
	state.SetLastHash(*previousHash)
	for i := 0; i < testBlockCount; i++ {
		height := i

		// Create coinbase tx to make a valid block
		tx := wire.NewMsgTx(2)
		outpoint := wire.NewOutPoint(zeroHash, 0xffffffff)
		script := make([]byte, 5)
		script[0] = 4 // push 4 bytes
		// Push 4 byte height
		script[1] = byte((height >> 24) & 0xff)
		script[2] = byte((height >> 16) & 0xff)
		script[3] = byte((height >> 8) & 0xff)
		script[4] = byte((height >> 0) & 0xff)
		input := wire.NewTxIn(outpoint, script)
		tx.AddTxIn(input)
		txs = append(txs, tx)

		merkleRoot := tx.TxHash()
		header := wire.NewBlockHeader(1, previousHash, merkleRoot, 0, 0)
		header.Timestamp = time.Unix(int64(t), 0)
		t += 600
		block := wire.NewMsgBlock(header)
		if err := block.AddTransaction(tx); err != nil {
			test.Errorf("Failed to add tx to block (%d) : %s", height, err)
		}

		blocks = append(blocks, block)
		if err := headersMsg.AddBlockHeader(header); err != nil {
			test.Errorf("Failed to add header to headers message : %v", err)
		}
		hash := header.BlockHash()
		previousHash = hash
	}

	// Send headers to handlers
	if err := handleMessage(ctx, testHandlers, headersMsg); err != nil {
		test.Errorf("Failed to process headers message : %v", err)
	}

	// Send corresponding blocks
	if err := sendBlocks(ctx, testHandlers, blocks, 0, &testListener); err != nil {
		test.Errorf("Failed to send block messages : %v", err)
	}

	verify(ctx, test, blocks, blockRepo, testBlockCount)

	test.Logf("Testing Reorg")

	// Cause a reorg
	reorgHeadersMsg := wire.NewMsgHeaders()
	reorgBlocks := make([]*wire.MsgBlock, 0, testBlockCount)
	hash := blocks[testBlockCount-reorgDepth].Header.BlockHash()
	previousHash = hash
	test.Logf("Reorging to (%d) : %s", (testBlockCount-reorgDepth)+1, previousHash.String())
	for i := testBlockCount - reorgDepth; i < testBlockCount; i++ {
		height := (testBlockCount - reorgDepth) + 1 + i

		// Create coinbase tx to make a valid block
		tx := wire.NewMsgTx(2)
		outpoint := wire.NewOutPoint(zeroHash, 0xffffffff)
		script := make([]byte, 5)
		script[0] = 4 // push 4 bytes
		// Push 4 byte height
		script[1] = byte((height >> 24) & 0xff)
		script[2] = byte((height >> 16) & 0xff)
		script[3] = byte((height >> 8) & 0xff)
		script[4] = byte((height >> 0) & 0xff)
		input := wire.NewTxIn(outpoint, script)
		tx.AddTxIn(input)
		txs = append(txs, tx)

		merkleRoot := tx.TxHash()
		header := wire.NewBlockHeader(int32(wire.ProtocolVersion), previousHash, merkleRoot, 0, 1)
		block := wire.NewMsgBlock(header)
		if err := block.AddTransaction(tx); err != nil {
			test.Errorf(fmt.Sprintf("Failed to add tx to block (%d)", height), err)
		}

		reorgBlocks = append(reorgBlocks, block)
		if err := reorgHeadersMsg.AddBlockHeader(header); err != nil {
			test.Errorf("Failed to add header to reorg headers message : %v", err)
		}
		hash := header.BlockHash()
		previousHash = hash
	}

	// Send reorg headers to handlers
	if err := handleMessage(ctx, testHandlers, reorgHeadersMsg); err != nil {
		test.Errorf("Failed to process reorg headers message : %v", err)
	}

	// Send corresponding reorg blocks
	if err := sendBlocks(ctx, testHandlers, reorgBlocks, (testBlockCount-reorgDepth)+1, &testListener); err != nil {
		test.Errorf("Failed to send reorg block messages : %v", err)
	}

	// Check reorg
	activeReorg, err := reorgRepo.GetActive(ctx)
	if err != nil {
		test.Errorf("Failed to get active reorg : %v", err)
	}

	if activeReorg == nil {
		test.Errorf("No active reorg found")
	}

	err = reorgRepo.ClearActive(ctx)
	if err != nil {
		test.Errorf("Failed to clear active reorg : %v", err)
	}

	activeReorg, err = reorgRepo.GetActive(ctx)
	if err != nil {
		test.Errorf("Failed to get active reorg after clear : %v", err)
	}
	if activeReorg != nil {
		test.Errorf("Active reorg was not cleared")
	}

	// Update headers array for reorg
	blocks = blocks[:(testBlockCount-reorgDepth)+1]
	for _, hash := range reorgBlocks {
		blocks = append(blocks, hash)
	}

	test.Logf("Block count %d = %d", len(blocks), testBlockCount+1)
	verify(ctx, test, blocks, blockRepo, testBlockCount+1)
}

func handleMessage(ctx context.Context, handlers map[string]CommandHandler, msg wire.Message) error {
	h, ok := handlers[msg.Command()]
	if !ok {
		// no handler for this command
		return nil
	}

	_, err := h.Handle(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func sendBlocks(ctx context.Context, handlers map[string]CommandHandler, blocks []*wire.MsgBlock,
	startHeight int, listener *TestListener) error {

	for i, block := range blocks {
		// Convert from MsgBlock to MsgParseBlock for the handler.
		var buf bytes.Buffer
		if err := block.BtcEncode(&buf, wire.ProtocolVersion); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to encode block (%d) message", startHeight+i))
		}

		parseBlock := &wire.MsgParseBlock{}
		if err := parseBlock.BtcDecode(bytes.NewReader(buf.Bytes()), wire.ProtocolVersion); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to decode block (%d) message", startHeight+i))
		}

		// Send block to handlers
		if err := handleMessage(ctx, handlers, parseBlock); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process block (%d) message", startHeight+i))
		}
	}

	if err := listener.ProcessBlocks(ctx); err != nil {
		return errors.Wrap(err, "process blocks")
	}

	return nil
}

func verify(ctx context.Context, test *testing.T, blocks []*wire.MsgBlock, blockRepo *handlerStorage.BlockRepository, testBlockCount int) {
	if blockRepo.LastHeight() != len(blocks) {
		test.Errorf("Block repo height %d doesn't match added %d", blockRepo.LastHeight(), len(blocks))
	}

	if !blocks[len(blocks)-1].Header.BlockHash().Equal(blockRepo.LastHash()) {
		test.Errorf("Block repo last hash doesn't match last added")
	}

	for i := 0; i < testBlockCount; i++ {
		hash := blocks[i].Header.BlockHash()
		height, _ := blockRepo.Height(hash)
		if height != i+1 {
			test.Errorf("Block repo height %d should be %d : %s", height, i+1, hash.String())
		}
	}

	for i := 0; i < testBlockCount; i++ {
		hash, err := blockRepo.Hash(ctx, i+1)
		if err != nil || hash == nil {
			test.Errorf("Block repo hash failed at height %d", i+1)
		} else if !hash.Equal(blocks[i].Header.BlockHash()) {
			test.Errorf("Block repo hash %d should : %s", i+1, blocks[i].Header.BlockHash().String())
		}
	}

	// Save repo
	if err := blockRepo.Save(ctx); err != nil {
		test.Errorf("Failed to save block repo : %v", err)
	}

	// Load repo
	if err := blockRepo.Load(ctx); err != nil {
		test.Errorf("Failed to load block repo : %v", err)
	}

	if blockRepo.LastHeight() != len(blocks) {
		test.Errorf("Block repo height %d doesn't match added %d after reload", blockRepo.LastHeight(), len(blocks))
	}

	if !blocks[len(blocks)-1].Header.BlockHash().Equal(blockRepo.LastHash()) {
		test.Errorf("Block repo last hash doesn't match last added after reload")
	}

	for i := 0; i < testBlockCount; i++ {
		hash := blocks[i].Header.BlockHash()
		height, _ := blockRepo.Height(hash)
		if height != i+1 {
			test.Errorf("Block repo height %d should be %d : %s", height, i+1, hash.String())
		}
	}

	for i := 0; i < testBlockCount; i++ {
		hash, err := blockRepo.Hash(ctx, i+1)
		if err != nil || hash == nil {
			test.Errorf("Block repo hash failed at height %d", i+1)
		} else if !hash.Equal(blocks[i].Header.BlockHash()) {
			test.Errorf("Block repo hash %d should : %s", i+1, blocks[i].Header.BlockHash().String())
		}
	}

	test.Logf("Verified %d blocks", len(blocks))
}

type TestListener struct {
	test      *testing.T
	state     *data.State
	blocks    *handlerStorage.BlockRepository
	txs       *handlerStorage.TxRepository
	height    int
	txTracker *data.TxTracker
}

// This is called when a block is being processed.
// It is responsible for any cleanup as a result of a block.
func (listener *TestListener) ProcessBlocks(ctx context.Context) error {

	for {
		block := listener.state.NextBlock()

		if block == nil {
			break
		}

		header := block.GetHeader()
		hash := header.BlockHash()

		if listener.blocks.Contains(hash) {
			height, _ := listener.blocks.Height(hash)
			logger.Warn(ctx, "Already have block (%d) : %s", height, hash.String())
			return errors.New("block not added")
		}

		if header.PrevBlock != *listener.blocks.LastHash() {
			// Ignore this as it can happen when there is a reorg.
			logger.Warn(ctx, "Not next block : %s", hash.String())
			logger.Warn(ctx, "Previous hash : %s", header.PrevBlock.String())
			return errors.New("not next block") // Unknown or out of order block
		}

		// Add to repo
		if err := listener.blocks.Add(ctx, &header); err != nil {
			return err
		}

		if err := listener.state.FinalizeBlock(*hash); err != nil {
			return err
		}
	}

	return nil
}

// Spynode listener interface
func (listener *TestListener) HandleBlock(ctx context.Context, msgType int, block *BlockMessage) error {
	switch msgType {
	case ListenerMsgBlock:
		listener.test.Logf("New Block (%d) : %s", block.Height, block.Hash.String())
	case ListenerMsgBlockRevert:
		listener.test.Logf("Reverted Block (%d) : %s", block.Height, block.Hash.String())
	}

	return nil
}

func (listener *TestListener) HandleTx(ctx context.Context, msg *wire.MsgTx) (bool, error) {
	listener.test.Logf("Tx : %s", msg.TxHash().String())
	return true, nil
}

func (listener *TestListener) HandleTxState(ctx context.Context, msgType int, txid bitcoin.Hash32) error {
	switch msgType {
	case ListenerMsgTxStateConfirm:
		listener.test.Logf("Tx confirm : %s", txid.String())
	case ListenerMsgTxStateRevert:
		listener.test.Logf("Tx revert : %s", txid.String())
	case ListenerMsgTxStateCancel:
		listener.test.Logf("Tx cancel : %s", txid.String())
	case ListenerMsgTxStateUnsafe:
		listener.test.Logf("Tx unsafe : %s", txid.String())
	}

	return nil
}

func (listener *TestListener) HandleInSync(ctx context.Context) error {
	listener.test.Logf("In Sync")
	return nil
}
