package handlers

import (
	"context"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type BlockMessage struct {
	Hash   bitcoin.Hash32
	Height int
	Time   time.Time
}

const (
	// Listener message types (msgType parameter).

	// New block was mined.
	ListenerMsgBlock = 1

	// Block reverted due to a reorg.
	ListenerMsgBlockRevert = 2

	// Transaction has seen no conflicting tx for specified delay.
	ListenerMsgTxStateSafe = 3

	// Transaction is included in the latest block announced with ListenerMsgBlock.
	ListenerMsgTxStateConfirm = 4

	// Transaction reverted due to a reorg.
	// This will be for confirmed transactions.
	ListenerMsgTxStateRevert = 5

	// Transaction reverted due to a double spend.
	// This will be for unconfirmed transactions.
	// A conflicting transaction was mined.
	ListenerMsgTxStateCancel = 6

	// Transaction conflicts with another tx.
	// These will come in at least sets of two when more than one tx spending the same input is
	//   seen.
	// If a confirm is later seen for one of these tx, then it can be assumed reliable.
	ListenerMsgTxStateUnsafe = 7
)

type Listener interface {
	// Block add and revert messages.
	HandleBlock(ctx context.Context, msgType int, block *BlockMessage) error

	// Full message for a transaction broadcast on the network.
	// Return true for txs that are relevant to ensure spynode sends further notifications for
	//   that tx.
	HandleTx(ctx context.Context, tx *wire.MsgTx) (bool, error)

	// Tx confirm, cancel, unsafe, and revert messages.
	HandleTxState(ctx context.Context, msgType int, txid bitcoin.Hash32) error

	// When in sync with network
	HandleInSync(ctx context.Context) error
}

// CommandHandler defines an interface for handing commands/messages received from
// peers over the Bitcoin P2P network.
type CommandHandler interface {
	Handle(context.Context, wire.Message) ([]wire.Message, error)
}

type StateReady interface {
	IsReady() bool
}

// NewCommandHandlers returns a mapping of commands and Handler's.
func NewTrustedCommandHandlers(ctx context.Context, config data.Config, state *data.State,
	peers *storage.PeerRepository, blockRepo *storage.BlockRepository, txRepo *storage.TxRepository,
	reorgRepo *storage.ReorgRepository, tracker *data.TxTracker, memPool *data.MemPool,
	unconfTxChannel *TxChannel, txStateChannel *TxStateChannel,
	listeners []Listener, txFilters []TxFilter) map[string]CommandHandler {

	return map[string]CommandHandler{
		wire.CmdPing:    NewPingHandler(),
		wire.CmdVersion: NewVersionHandler(state, config.NodeAddress),
		wire.CmdAddr:    NewAddressHandler(peers),
		wire.CmdInv:     NewInvHandler(state, txRepo, tracker, memPool),
		wire.CmdTx:      NewTXHandler(state, unconfTxChannel, memPool, txRepo, listeners, txFilters),
		wire.CmdBlock:   NewBlockHandler(state),
		wire.CmdHeaders: NewHeadersHandler(config, state, txStateChannel, blockRepo, txRepo,
			reorgRepo, listeners),
		wire.CmdReject: NewRejectHandler(),
	}
}

// NewUntrustedCommandHandlers returns a mapping of commands and Handler's.
func NewUntrustedCommandHandlers(ctx context.Context, trustedState *data.State,
	untrustedState *data.UntrustedState, peers *storage.PeerRepository,
	blockRepo *storage.BlockRepository, txRepo *storage.TxRepository, tracker *data.TxTracker,
	memPool *data.MemPool, txChannel *TxChannel, listeners []Listener,
	txFilters []TxFilter, address string) map[string]CommandHandler {

	return map[string]CommandHandler{
		wire.CmdPing:    NewPingHandler(),
		wire.CmdVersion: NewUntrustedVersionHandler(untrustedState, address),
		wire.CmdAddr:    NewAddressHandler(peers),
		wire.CmdInv:     NewUntrustedInvHandler(untrustedState, tracker, memPool),
		wire.CmdTx: NewUntrustedTXHandler(untrustedState, txChannel, memPool, txRepo,
			listeners, txFilters),
		wire.CmdBlock:   NewBlockHandler(trustedState),
		wire.CmdHeaders: NewUntrustedHeadersHandler(untrustedState, peers, address, blockRepo),
		wire.CmdReject:  NewRejectHandler(),
	}
}
