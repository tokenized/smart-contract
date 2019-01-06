package node

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/broadcaster"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// BlockHandler exists to handle the Block command.
type BlockHandler struct {
	Config      config.Config
	Network     network.NetworkInterface
	Inspector   inspector.InspectorService
	Broadcaster broadcaster.BroadcastService
	Validator   validator.ValidatorService
	Request     request.RequestService
	Response    response.ResponseService
}

// NewBlockHandler returns a new BlockHandler with the given Config.
func NewBlockHandler(config config.Config,
	network network.NetworkInterface,
	inspector inspector.InspectorService,
	broadcaster broadcaster.BroadcastService,
	validator validator.ValidatorService,
	request request.RequestService,
	response response.ResponseService) TXHandler {
	return TXHandler{
		Config:      config,
		Network:     network,
		Inspector:   inspector,
		Broadcaster: broadcaster,
		Validator:   validator,
		Request:     request,
		Response:    response,
	}
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (h BlockHandler) Handle(ctx context.Context, m wire.Message) error {
	tx, ok := m.(*wire.MsgBlock)
	if !ok {
		return errors.New("Could not assert as *wire.MsgBlock")
	}

	return h.handle(ctx, tx)
}

// handle processes the MsgBlock
func (h BlockHandler) handle(ctx context.Context, b *wire.MsgBlock) error {
	log := logger.NewLoggerFromContext(ctx).Sugar()
	log.Infof("Received block : %s", b.BlockHash())

	for _, tx := range b.Transactions {

		// Inspector: Does this transaction concern the protocol?
		itx, err := h.Inspector.MakeTransaction(tx)
		if err != nil || itx == nil {
			// log.Error(err)
			return nil
		}

		// if _, err := h.Service.ProcessMessage(ctx, itx); err != nil {
		// 	logger.Errorw("Failed to process message", zap.Error(err))
		// 	return err
		// }
	}

	return nil
}
