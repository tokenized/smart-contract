package protomux

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	// SEE is used for broadcast txs
	SEE = "SEE"

	// SAW is used for replayed txs (reserved)
	SAW = "SAW"

	// LOST is used for reorgs
	LOST = "LOST"

	// STOLE is used for double spends
	STOLE = "STOLE"
)

// Handler is the interface for this Protocol Mux
type Handler interface {
	Respond(ctx context.Context, m wire.Message) error
	Trigger(ctx context.Context, verb string, m wire.Message) error
}

// A Handler is a type that handles a protocol messages
type HandlerFunc func(itx *inspector.Transaction, m protocol.OpReturnMessage)

// A ResponderFunc will handle responses
type ResponderFunc func(ctx context.Context, tx *wire.MsgTx)

type ProtoMux struct {
	// Node          NodeInterface
	Responder     ResponderFunc
	SeeHandlers   map[string][]HandlerFunc
	LostHandlers  map[string][]HandlerFunc
	StoleHandlers map[string][]HandlerFunc
}

func New() *ProtoMux {
	pm := &ProtoMux{
		SeeHandlers:   make(map[string][]HandlerFunc),
		LostHandlers:  make(map[string][]HandlerFunc),
		StoleHandlers: make(map[string][]HandlerFunc),
	}

	return pm
}

// Handle registers a new handler
func (p *ProtoMux) Handle(verb, event string, handler HandlerFunc) {
	switch verb {
	case SEE:
		p.SeeHandlers[event] = append(p.SeeHandlers[event], handler)
	case LOST:
		p.LostHandlers[event] = append(p.LostHandlers[event], handler)
	case STOLE:
		p.StoleHandlers[event] = append(p.StoleHandlers[event], handler)
	default:
		panic("Unknown handler type")
	}
}

// Trigger fires a handler
func (p *ProtoMux) Trigger(ctx context.Context, verb string, m wire.Message) error {
	tx, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	var group map[string][]HandlerFunc

	switch verb {
	case SEE:
		group = p.SeeHandlers
	case LOST:
		group = p.LostHandlers
	case STOLE:
		group = p.StoleHandlers
	default:
		return errors.New("Unknown handler type")
	}

	// Check if transaction relates to protocol
	itx, err := inspector.NewTransactionFromWire(ctx, tx)
	if err != nil || !itx.IsTokenized() {
		return nil
	}

	// Locate the handlers from the group
	txAction := itx.MsgProto.Type()
	handlers, _ := group[txAction]

	// Notify the listeners
	for _, listener := range handlers {
		listener(itx, itx.MsgProto)
	}

	return nil
}

// Respond handles a response via the responder
func (p *ProtoMux) Respond(ctx context.Context, m wire.Message) error {
	tx, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	p.Responder(ctx, tx)

	return nil
}
