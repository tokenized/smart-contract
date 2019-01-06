package logger

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type key int

const (
	// KeyRequestID is the Request ID in the Context.
	KeyRequestID key = 0

	// KeyLogger is the Logger in the Context.
	KeyLogger key = 1

	// KeyTXHash is the key for the TXHash in the Context.
	KeyTXHash key = 2
)

// NewContext returns a fully configured Context with from a background
// Context, with a new RequestID set, and a Logger.
//
// The Logger will include the RequestID field.
func NewContext() context.Context {
	return ContextWithRequestID(context.Background(), "")
}

// NewContextWithRequestID returns a fully configured Context, the same
// as from NewContext, but with the given RequestID.
func NewContextWithRequestID(id string) context.Context {
	return ContextWithRequestID(context.Background(), id)
}

// ContextWithRequestID returns a fully configured Context from the given
// Context and RequestID.
//
// If the RequestID is an empty string, a RequestID will be generated.
//
// The Context will have a Logger, which will have the RequestID field set.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	if len(id) == 0 {
		uid, _ := uuid.NewRandom()
		id = uid.String()
	}

	// add the request ID
	ctx = context.WithValue(ctx, KeyRequestID, id)

	// add the logger which will have the request ID (and other optional
	// fields) set.
	logger := newLogger(ctx)

	return ContextWithLogger(ctx, logger)
}

// ContextWithTXHash returns a Context with the TXHash set.
//
// A Logger with the TXHash field set is associated with the Context.
func ContextWithTXHash(ctx context.Context, txHash string) context.Context {
	// add the request ID
	ctx = context.WithValue(ctx, KeyTXHash, txHash)

	// modify the existing logger to include the tx hash in the fields
	logger := NewLoggerFromContext(ctx)
	logger = logger.With(zap.String(fieldTXHash, txHash))

	return ContextWithLogger(ctx, logger)
}

// ContextWithLogger adds the Logger to the Context.
func ContextWithLogger(ctx context.Context,
	logger *zap.Logger) context.Context {

	return context.WithValue(ctx, KeyLogger, logger)
}

// NewContextWithNamedLogger returns a new Context with a named Logger.
func NewContextWithNamedLogger(name string) context.Context {
	ctx := NewContext()
	return ContextWithNamedLogger(ctx, name)
}

// ContextWithNamedLogger returns a Context with a new named Logger.
func ContextWithNamedLogger(ctx context.Context, name string) context.Context {
	logger := NewLoggerFromContext(ctx)
	logger = logger.Named(name)

	return context.WithValue(ctx, KeyLogger, logger)
}

// RequestIDFromContext returns the request ID from the Context.
//
// If the value was not set in the Context, "unknown" is returned. This can
// help find services that are not adding the RequestID.
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(KeyRequestID)

	if v == nil {
		// find these in the logs as it "breaks" the request id chain
		// we use for tracing actions.
		id, _ := uuid.NewRandom()
		return fmt.Sprintf("unknown/%s", id.String())
	}

	return v.(string)
}

// TXHashFromContext returns the Hash of the TX being processed if set,
// otherwise an empty string.
func TXHashFromContext(ctx context.Context) string {
	v := ctx.Value(KeyTXHash)

	if v == nil {
		return ""
	}

	return v.(string)
}
