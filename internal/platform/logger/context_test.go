package logger

import (
	"context"
	"regexp"
	"testing"

	"go.uber.org/zap"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	if ctx.Value(KeyLogger) == nil {
		t.Errorf("Want not nil, got nil")
	}

	if ctx.Value(KeyRequestID) == "" {
		t.Errorf("Expected request ID value to be non-empty")
	}

	requestID := ctx.Value(KeyRequestID).(string)

	if len(requestID) != 36 {
		t.Errorf("Got %v, want %v", len(requestID), 36)
	}
}

func TestContextWithRequestID(t *testing.T) {
	ctx := context.Background()

	gotNotSet := RequestIDFromContext(ctx)

	pattern := "unknown/[[:ascii:]]{36}"
	match, _ := regexp.MatchString(pattern, gotNotSet)

	if !match {
		t.Errorf("%v did not match %v", gotNotSet, pattern)
	}

	want := "foo"
	ctx = ContextWithRequestID(ctx, want)

	if ctx.Value(KeyLogger) == nil {
		t.Errorf("Want not nil, got nil")
	}

	got := ctx.Value(KeyRequestID)
	if got != want {
		t.Errorf("Got %v, want %v", got, want)
	}

	got = RequestIDFromContext(ctx)
	if got != want {
		t.Errorf("Got %v, want %v", got, want)
	}
}

func TestContextWithLogger(t *testing.T) {
	ctx := context.Background()

	logger, _ := zap.NewProduction()
	ctx = ContextWithLogger(ctx, logger)

	if ctx.Value(KeyLogger) != logger {
		t.Errorf("Want %v, got %v", logger, ctx.Value(KeyLogger))
	}
}

func TestNewLoggerFromContext(t *testing.T) {
	ctx := NewContext()

	logger := NewLoggerFromContext(ctx)

	if logger == nil {
		t.Errorf("Want non-nil Logger")
	}
}

func TestNewLoggerFromContext_nilLogger(t *testing.T) {
	ctx := context.Background()

	logger := NewLoggerFromContext(ctx)

	if logger == nil {
		t.Errorf("Want non-nil Logger")
	}
}

func TestNewContextWithNamedLogger(t *testing.T) {
	ctx := NewContextWithNamedLogger("foo")

	requestID := RequestIDFromContext(ctx)

	if len(requestID) == 0 {
		t.Errorf("Expected non-zero length requestID")
	}

	logger := NewLoggerFromContext(ctx)

	if logger == nil {
		t.Errorf("Want non-nil Logger")
	}
}
