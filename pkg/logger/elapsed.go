package logger

import (
	"context"
	"time"
)

// Elapsed write elapsed time in milliseconds to the Logger.
func Elapsed(ctx context.Context, start time.Time, message string) {
	// get the elapsed time in milliseconds
	ms := float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond)

	LogDepth(ctx, LevelVerbose, 1, "%s : %0.3f ms", message, ms)
}
