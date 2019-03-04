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

	// Full message for a transaction broadcast on the network.
	// The listener must return true for txs that are relevant to ensure spynode sends further
	//   notifications for that tx.
	ListenerMsgTx = 1 // msgValue is wire.MsgTx

	// New block was mined.
	ListenerMsgBlock = 2 // msgValue is handlers.BlockMessage

	// Transaction is included the latest block.
	ListenerMsgTxConfirm = 3 // msgValue is chainhash.Hash

	// Block reverted due to a reorg.
	ListenerMsgBlockRevert = 4 // msgValue is handlers.BlockMessage

	// Transaction reverted due to a reorg.
	// This will be for confirmed transactions.
	ListenerMsgTxRevert = 5 // msgValue is chainhash.Hash

	// Transaction reverted due to a double spend.
	// This will be for unconfirmed transactions.
	// A conflicting transaction was mined.
	ListenerMsgTxCancel = 6 // msgValue is chainhash.Hash

	// Transaction conflicts with another tx.
	// These will come in at least sets of two when more than one tx spending the same input is
	//   seen.
	// If a confirm is later seen for one of these tx, then it can be assumed reliable.
	ListenerMsgTxUnsafe = 7 // msgValue is chainhash.Hash

	// This message means that the node is current with the network.
	// All data from this point forward is live.
	ListenerMsgInSync = 8 // msgValue is nil
)

type Listener interface {
	Handle(ctx context.Context, msgType int, msgValue interface{}) (bool, error)
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
