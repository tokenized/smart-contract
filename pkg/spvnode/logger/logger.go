package logger

import (
	"context"

	"go.uber.org/zap"
)

const (
	fieldRequestID = "request_id"
	fieldTXHash    = "tx_hash"
)

func NewLoggerWithContext() (context.Context, *zap.SugaredLogger) {
	ctx := NewContext()
	logger := NewLoggerFromContext(ctx).Sugar()
	return ctx, logger
}

// NewLoggerFromContext returns the Logger from the Context. If a Logger doesn't
// exist one is added.
func NewLoggerFromContext(ctx context.Context) *zap.Logger {
	logger := ctx.Value(KeyLogger)

	if logger == nil {
		logger = newLogger(ctx)
	}

	return logger.(*zap.Logger)
}

// newLogger returns a Logger with the RequestID from the Context as a
// field.
func newLogger(ctx context.Context) *zap.Logger {
	logger, _ := zap.NewProduction()

	// Add the request ID to the logger
	requestID := RequestIDFromContext(ctx)
	logger = logger.With(zap.String(fieldRequestID, requestID))

	txHash := TXHashFromContext(ctx)
	if len(txHash) > 0 {
		logger = logger.With(zap.String(fieldTXHash, txHash))
	}

	return logger
}
