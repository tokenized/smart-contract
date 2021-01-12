package node

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how event values or stored/retrieved.
const KeyValues ctxKey = 1

// Values represent state for each event.
type Values struct {
	Now        protocol.Timestamp
	StatusCode int
	Error      bool
}

// BitcoinHeaders provides functions for retrieving information about headers on the currently
// longest chain.
type BitcoinHeaders interface {
	LastHeight(ctx context.Context) int
	Hash(ctx context.Context, height int) (*bitcoin.Hash32, error)
	Time(ctx context.Context, height int) (uint32, error)
}

// A Handler is a type that handles a transaction within our own little mini framework.
type Handler func(ctx context.Context, w *ResponseWriter, itx *inspector.Transaction, wk *wallet.Key) error

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers. Feel free to add any configuration
// data/logic on this App struct
type App struct {
	*protomux.ProtoMux
	config   *Config
	mw       []Middleware
	masterDB *db.DB
	wallet   wallet.WalletInterface
}

// Node configuration
type Config struct {
	ContractProviderID string
	Version            string
	FeeAddress         bitcoin.RawAddress
	Net                bitcoin.Network
	FeeRate            float32
	DustFeeRate        float32
	MinFeeRate         float32
	RequestTimeout     uint64 // Nanoseconds until a request to another contract times out and the original request is rejected.
	PreprocessThreads  int
	IsTest             bool
}

// New creates an App value that handle a set of routes for the application.
func New(config *Config, masterDB *db.DB, wallet wallet.WalletInterface, mw ...Middleware) *App {
	return &App{
		ProtoMux: protomux.New(),
		config:   config,
		mw:       mw,
		masterDB: masterDB,
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
	h := func(ctx context.Context, itx *inspector.Transaction) error {
		// Prepare response writer
		w := &ResponseWriter{
			Mux:      a.ProtoMux,
			Config:   a.config,
			MasterDB: a.masterDB,
		}

		// For each address controlled by this wallet
		walletKeys := a.wallet.ListAll()
		handled := false
		for _, walletKey := range walletKeys {
			if !itx.IsRelevant(walletKey.Address) {
				continue
			}

			// Set the context with the required values to process the event.
			v := Values{
				Now: protocol.CurrentTimestamp(),
			}
			ctx = context.WithValue(ctx, KeyValues, &v)

			// Call the wrapped handler functions.
			handled = true
			if err := handler(ctx, w, itx, walletKey); err != nil {
				return err
			}
		}

		if !handled {
			Log(ctx, "Unrelated tx : %s", itx.Hash.String())
		}

		return nil
	}

	// Add this handler for the specified verb and event.
	a.ProtoMux.Handle(verb, event, h)
}
