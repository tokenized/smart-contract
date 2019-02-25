package node

import (
	"context"
	"log"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/wire"
)

var (
	// StatusNoResponse occurs when there is no response.
	StatusNoResponse = 0

	// StatusOK occurs for a normal response
	StatusOK = 1

	// StatusRejected occurs for a rejected response
	StatusRejected = 2
)

// Error handles all error responses for the API.
func Error(cxt context.Context, log *log.Logger, mux *protomux.ProtoMux, err error) {
	// switch errors.Cause(err) {
	// }

	RespondError(cxt, log, mux, err, StatusNoResponse)
}

// RespondError sends JSON describing the error
func RespondError(ctx context.Context, log *log.Logger, mux *protomux.ProtoMux, err error, code int) {
	// Set up the Wire transaction
	tx := &wire.MsgTx{}

	Respond(ctx, log, mux, tx, code)
}

// Respond sends a TX to the network.
// If code is StatusNoResponse, data is expected to be nil.
func Respond(ctx context.Context, log *log.Logger, mux *protomux.ProtoMux, tx *wire.MsgTx, code int) {
	mux.Respond(ctx, tx)
}
