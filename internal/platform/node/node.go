package node

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
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
	Now        protocol.Timestamp
	StatusCode int
	Error      bool
}

// A Handler is a type that handles a transaction within our own little mini framework.
// type Handler func(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error
type Handler func(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	*protomux.ProtoMux
	config *Config
	mw     []Middleware
	wallet wallet.WalletInterface
}

// Node configuration
type Config struct {
	ContractProviderID string
	Version            string
	FeePKH             *protocol.PublicKeyHash
	FeeValue           uint64
	DustLimit          uint64
	ChainParams        chaincfg.Params
	FeeRate            float32
	RequestTimeout     uint64 // Nanoseconds until a request to another contract times out and the original request is rejected.
}

// New creates an App value that handle a set of routes for the application.
func New(config *Config, wallet wallet.WalletInterface, mw ...Middleware) *App {
	return &App{
		ProtoMux: protomux.New(),
		config:   config,
		mw:       mw,
		wallet:   wallet,
	}
}

// Handle is our mechanism for mounting Handlers for a given event
// this makes for really easy, convenient event handling.
func (a *App) Handle(verb, event string, handler Handler, mw ...Middleware) {

	// Wrap up the application-wide first, this will call the first function
	// of each middleware which will return a function of type Handler.
	handler = wrapMiddleware(wrapMiddleware(handler, mw), a.mw)

	// The function to execute for each event.
	h := func(ctx context.Context, itx *inspector.Transaction, pkhs []string) error {

		// Start trace span.
		ctx, span := trace.StartSpan(ctx, "internal.platform.node")

		// Set the context with the required values to
		// process the event.
		v := Values{
			TraceID: span.SpanContext().TraceID.String(),
			Now:     protocol.CurrentTimestamp(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Prepare response writer
		w := &ResponseWriter{
			Mux:    a.ProtoMux,
			Config: a.config,
		}

		// For each address controlled by this wallet
		rootKeys, _ := a.wallet.List(pkhs)
		for _, rootKey := range rootKeys {
			// Call the wrapped handler functions.
			if err := handler(ctx, w, itx, rootKey); err != nil {
				return err
			}
		}

		return nil
	}

	// Add this handler for the specified verb and event.
	a.ProtoMux.Handle(verb, event, h)
}
