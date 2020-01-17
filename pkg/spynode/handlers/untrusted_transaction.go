package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// TXHandler exists to handle the tx command.
type UntrustedTXHandler struct {
	ready     StateReady
	txChannel *TxChannel
	memPool   *data.MemPool
	txs       *storage.TxRepository
	listeners []Listener
	txFilters []TxFilter
}

// NewTXHandler returns a new TXHandler with the given Config.
func NewUntrustedTXHandler(ready StateReady, txChannel *TxChannel, memPool *data.MemPool, txs *storage.TxRepository, listeners []Listener, txFilters []TxFilter) *UntrustedTXHandler {
	result := UntrustedTXHandler{
		ready:     ready,
		txChannel: txChannel,
		memPool:   memPool,
		txs:       txs,
		listeners: listeners,
		txFilters: txFilters,
	}
	return &result
}

// Handle implements the handler interface for transaction handler.
func (handler *UntrustedTXHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgTx)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgTx")
	}

	// Only notify of transactions when in sync or they might be duplicated, since there isn't a mempool yet.
	if !handler.ready.IsReady() {
		return nil, nil
	}

	handler.txChannel.Add(TxData{Msg: msg, Trusted: false, ConfirmedHeight: -1})
	return nil, nil
}
