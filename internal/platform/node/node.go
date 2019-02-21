package node

import (
	"context"
	"log"
	"time"

	"github.com/tokenized/smart-contract/pkg/spvnode"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how request values or stored/retrieved.
const KeyValues ctxKey = 1

// Values represent state for each request.
type Values struct {
	TraceID    string
	Now        time.Time
	StatusCode int
	Error      bool
}

// A Handler is a type that handles a HTTP request within our own little mini
// framework.
type Handler func(ctx context.Context, log *log.Logger, m wire.Message) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	log  *log.Logger
	node *spvnode.Node
	mw   []Middleware
}

// Node configuration
type Config struct {
	ContractProviderID string
	Version            string
	FeeAddress         string
	FeeValue           uint64
}

// New creates an App value that handle a set of routes for the application.
func New(log *log.Logger, sn *spvnode.Node, mw ...Middleware) *App {
	return &App{
		log:  log,
		node: sn,
	}
}

// Handle is our mechanism for mounting Handlers for a given event
// this makes for really easy, convenient event handling.
func (a *App) Handle(event string, handler Handler, mw ...Middleware) {

	// Wrap up the application-wide first, this will call the first function
	// of each middleware which will return a function of type Handler.
	handler = wrapMiddleware(wrapMiddleware(handler, mw), a.mw)

	// The function to execute for each request.
	h := func(ctx context.Context, m wire.Message) error {

		// Start trace span.
		ctx, span := trace.StartSpan(ctx, "internal.platform.node")

		// Set the context with the required values to
		// process the request.
		v := Values{
			TraceID: span.SpanContext().TraceID.String(),
			Now:     time.Now(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Call the wrapped handler functions.
		if err := handler(ctx, a.log, m); err != nil {
			return err
		}

		return nil
	}

	// Add this handler for the specified event.
	a.node.RegisterListener(event, h)
}
