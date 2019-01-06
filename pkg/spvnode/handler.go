package spvnode

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"
)

// CommandHandler defines an interface for handing commands received from
// BCH peers over the network.
type CommandHandler interface {
	Handle(context.Context, wire.Message) ([]wire.Message, error)
}

type Listener interface {
	Handle(context.Context, wire.Message) error
}

// newCommandHandlers returns a mapping of commands and Handler's.
func newCommandHandlers(config Config,
	blockService *BlockService,
	listeners map[string]Listener) map[string]CommandHandler {

	return map[string]CommandHandler{
		wire.CmdPing:    NewPingHandler(config),
		wire.CmdVersion: NewVersionHandler(config),
		wire.CmdInv:     NewInvHandler(config),
		wire.CmdTx:      NewTXHandler(config, blockService, listeners[ListenerTX]),
		wire.CmdBlock:   NewBlockHandler(config, blockService, listeners[ListenerBlock]),
		// wire.CmdGetHeaders: NewGetHeadersHandler(config, blockService),
		// wire.CmdHeaders:    NewHeadersHandler(config, blockService),
	}
}
