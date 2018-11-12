package node

import (
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/internal/broadcaster"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// TXHandler exists to handle the TX command.
type TXHandler struct {
	Config      config.Config
	Network     network.NetworkInterface
	Wallet      wallet.Wallet
	Inspector   inspector.InspectorService
	Broadcaster broadcaster.BroadcastService
	Validator   validator.ValidatorService
	Request     request.RequestService
	Response    response.ResponseService
	mapLock     mapLock
}

// NewTXHandler returns a new TXHandler with the given Config.
func NewTXHandler(config config.Config,
	network network.NetworkInterface,
	wallet wallet.Wallet,
	inspector inspector.InspectorService,
	broadcaster broadcaster.BroadcastService,
	validator validator.ValidatorService,
	request request.RequestService,
	response response.ResponseService) TXHandler {
	return TXHandler{
		Config:      config,
		Network:     network,
		Wallet:      wallet,
		Inspector:   inspector,
		Broadcaster: broadcaster,
		Validator:   validator,
		Request:     request,
		Response:    response,
		mapLock:     newMapLock(),
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (h TXHandler) Handle(ctx context.Context, m wire.Message) error {
	msg, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	return h.handle(ctx, msg)
}

// handle processes the MsgTx.
//
// There is no response for this handler.
func (h TXHandler) handle(ctx context.Context, tx *wire.MsgTx) error {

	// Decorate the Context with the hash of the TX we are processing
	ctx = logger.ContextWithTXHash(ctx, tx.TxHash().String())
	ts := time.Now()

	// Inspector: Does this transaction concern the protocol?
	itx, err := h.Inspector.MakeTransaction(tx)
	if err != nil || itx == nil {
		return nil
	}

	// Filter by Contract PKH and Request-type action
	itx, err = h.Request.PreFilter(ctx, itx)
	if err != nil || itx == nil {
		return nil
	}

	// we don't care about non-Tokenized tx's, so taking metrics here. The
	// ts was taken at the beginning of the function.
	defer logger.Elapsed(ctx, ts, "TXHandler.handle")

	// Introduce Inputs and UTXOs in the Transaction
	itx, err = h.Inspector.PromoteTransaction(itx)
	if err != nil {
		return nil
	}

	// To ensure multiple messages do not modify the same Contract in
	// parallel, use a mutex to prevent parallel access on a contract
	// address.
	mtx := h.mapLock.get(h.Wallet.PublicAddress)
	mtx.Lock()
	defer mtx.Unlock()

	// Validator: Check this request, return the related Contract
	rejectTx, contract, err := h.Validator.CheckAndFetch(ctx, itx)
	if err != nil {
		return nil
	}

	// Validator: Message is a reject
	if rejectTx != nil {
		_, _ = h.Broadcaster.Announce(ctx, rejectTx)
		return nil
	}
	if contract == nil {
		return nil
	}

	// Request: Grab me a response
	resItx, err := h.Request.Process(ctx, itx, contract)
	if err != nil {
		return nil
	}

	// Response: Process response
	err = h.Response.Process(ctx, resItx, contract)
	if err != nil {
		return nil
	}

	// Broadcaster: Broadcast response
	_, err = h.Broadcaster.Announce(ctx, resItx.MsgTx)
	if err != nil {
		return nil
	}

	// there is nothing to return, because this handler doesn't return
	// messages back to the peer. Any messaging was handled by the Service.
	return nil
}
