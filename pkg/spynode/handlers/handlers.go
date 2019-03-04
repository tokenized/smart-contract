package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type BlockMessage struct {
	Hash   chainhash.Hash
	Height int
}

const (
	// Listener message types (msgType parameter).

	// New block was mined.
	ListenerMsgBlock = 1 // msgValue is handlers.BlockMessage

	// Block reverted due to a reorg.
	ListenerMsgBlockRevert = 2 // msgValue is handlers.BlockMessage

	// Transaction is included the latest block.
	ListenerMsgTxStateConfirm = 3 // msgValue is chainhash.Hash

	// Transaction reverted due to a reorg.
	// This will be for confirmed transactions.
	ListenerMsgTxStateRevert = 5 // msgValue is chainhash.Hash

	// Transaction reverted due to a double spend.
	// This will be for unconfirmed transactions.
	// A conflicting transaction was mined.
	ListenerMsgTxStateCancel = 6 // msgValue is chainhash.Hash

	// Transaction conflicts with another tx.
	// These will come in at least sets of two when more than one tx spending the same input is
	//   seen.
	// If a confirm is later seen for one of these tx, then it can be assumed reliable.
	ListenerMsgTxStateUnsafe = 7 // msgValue is chainhash.Hash
)

type Listener interface {
	// Block add and revert messages.
	HandleBlock(ctx context.Context, msgType int, block *BlockMessage) error

	// Full message for a transaction broadcast on the network.
	// Return true for txs that are relevant to ensure spynode sends further notifications for
	//   that tx.
	HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error)

	// Tx confirm, cancel, unsafe, and revert messages.
	HandleTxState(ctx context.Context, msgType int, txid chainhash.Hash) error

	// When in sync with network
	HandleInSync(ctx context.Context) error
}

// CommandHandler defines an interface for handing commands/messages received from
// peers over the Bitcoin P2P network.
type CommandHandler interface {
	Handle(context.Context, wire.Message) ([]wire.Message, error)
}

// Procesess cleanup for blocks
// Used to clean TxTrackers when blocks are confirmed.
type BlockProcessor interface {
	ProcessBlock(context.Context, *wire.MsgBlock) error
}

type StateReady interface {
	IsReady() bool
}

// NewCommandHandlers returns a mapping of commands and Handler's.
func NewTrustedCommandHandlers(ctx context.Context, config data.Config, state *data.State, peers *storage.PeerRepository, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository, tracker *data.TxTracker, memPool *data.MemPool, listeners []Listener, txFilters []TxFilter, blockProcessor BlockProcessor) map[string]CommandHandler {
	return map[string]CommandHandler{
		wire.CmdPing:    NewPingHandler(),
		wire.CmdVersion: NewVersionHandler(state),
		wire.CmdAddr:    NewAddressHandler(peers),
		wire.CmdInv:     NewInvHandler(state, tracker, memPool),
		wire.CmdTx:      NewTXHandler(state, memPool, txRepo, listeners, txFilters),
		wire.CmdBlock:   NewBlockHandler(state, memPool, blockRepo, txRepo, listeners, txFilters, blockProcessor),
		wire.CmdHeaders: NewHeadersHandler(config, state, blockRepo, txRepo, listeners),
		wire.CmdReject:  NewRejectHandler(),
	}
}

// NewUntrustedCommandHandlers returns a mapping of commands and Handler's.
func NewUntrustedCommandHandlers(ctx context.Context, state *data.UntrustedState, peers *storage.PeerRepository, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository, tracker *data.TxTracker, memPool *data.MemPool, listeners []Listener, txFilters []TxFilter) map[string]CommandHandler {
	return map[string]CommandHandler{
		wire.CmdPing:    NewPingHandler(),
		wire.CmdVersion: NewUntrustedVersionHandler(state),
		wire.CmdAddr:    NewAddressHandler(peers),
		wire.CmdInv:     NewUntrustedInvHandler(state, tracker, memPool),
		wire.CmdTx:      NewTXHandler(state, memPool, txRepo, listeners, txFilters),
		wire.CmdHeaders: NewUntrustedHeadersHandler(state, blockRepo),
		wire.CmdReject:  NewRejectHandler(),
	}
}
