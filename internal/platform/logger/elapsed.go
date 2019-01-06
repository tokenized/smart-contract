package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Elapsed write elapsed time in milliseconds to the Logger.
func Elapsed(ctx context.Context, t time.Time, message string) {
	logger := NewLoggerFromContext(ctx)

	// get the elapsed time in milliseconds
	ms := float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)

	logger.Info(message, zap.Float64("elapsed", ms))
}
