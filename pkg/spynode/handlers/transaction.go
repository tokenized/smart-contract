package handlers

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers/storage"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// TXHandler exists to handle the tx command.
type TXHandler struct {
	ready     StateReady
	txChannel *TxChannel
	memPool   *data.MemPool
	txs       *storage.TxRepository
	listeners []Listener
	txFilters []TxFilter
}

type TxData struct {
	Msg             *wire.MsgTx
	Trusted         bool
	Safe            bool
	ConfirmedHeight int
}

// NewTXHandler returns a new TXHandler with the given Config.
func NewTXHandler(ready StateReady, txChannel *TxChannel, memPool *data.MemPool,
	txs *storage.TxRepository, listeners []Listener, txFilters []TxFilter) *TXHandler {

	result := TXHandler{
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
func (handler *TXHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgTx)
	if !ok {
		return nil, errors.New("Could not assert as *wire.MsgTx")
	}

	// Only notify of transactions when in sync or they might be duplicated, since there isn't a mempool yet.
	if !handler.ready.IsReady() {
		return nil, nil
	}

	handler.txChannel.Add(TxData{Msg: msg, Trusted: true, ConfirmedHeight: -1})
	return nil, nil
}

type TxChannel struct {
	Channel chan TxData
	lock    sync.Mutex
	open    bool
}

func (c *TxChannel) Add(tx TxData) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- tx
	return nil
}

func (c *TxChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan TxData, count)
	c.open = true
	return nil
}

func (c *TxChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}

type TxState struct {
	MsgType int
	TxId    bitcoin.Hash32
}

type TxStateChannel struct {
	Channel chan TxState
	lock    sync.Mutex
	open    bool
}

func (c *TxStateChannel) Add(tx TxState) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- tx
	return nil
}

func (c *TxStateChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan TxState, count)
	c.open = true
	return nil
}

func (c *TxStateChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
