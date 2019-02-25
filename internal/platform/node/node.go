package node

import (
	"context"
	"log"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"go.opencensus.io/trace"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how event values or stored/retrieved.
const KeyValues ctxKey = 1

// Values represent state for each event.
type Values struct {
	TraceID    string
	Now        time.Time
	StatusCode int
	Error      bool
}

// A Handler is a type that handles a transaction within our own little mini framework.
type Handler func(ctx context.Context, log *log.Logger, itx *inspector.Transaction, m protocol.OpReturnMessage) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	*protomux.ProtoMux
	log *log.Logger
	mw  []Middleware
}

// Node configuration
type Config struct {
	ContractProviderID string
	Version            string
	FeeAddress         string
	FeeValue           uint64
}

// New creates an App value that handle a set of routes for the application.
func New(log *log.Logger, mw ...Middleware) *App {
	return &App{
		ProtoMux: protomux.New(),
		log:      log,
		mw:       mw,
	}
}

// Handle is our mechanism for mounting Handlers for a given event
// this makes for really easy, convenient event handling.
func (a *App) Handle(verb, event string, handler Handler, mw ...Middleware) {

	// Wrap up the application-wide first, this will call the first function
	// of each middleware which will return a function of type Handler.
	handler = wrapMiddleware(wrapMiddleware(handler, mw), a.mw)

	// The function to execute for each event.
	h := func(itx *inspector.Transaction, m protocol.OpReturnMessage) {

		// Start trace span.
		ctx, span := trace.StartSpan(context.Background(), "internal.platform.node")

		// Set the context with the required values to
		// process the event.
		v := Values{
			TraceID: span.SpanContext().TraceID.String(),
			Now:     time.Now(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Call the wrapped handler functions.
		if err := handler(ctx, a.log, itx, m); err != nil {
			Error(ctx, a.log, a.ProtoMux, err)
		}
	}

	// Add this handler for the specified verb and event.
	a.ProtoMux.Handle(verb, event, h)
}
