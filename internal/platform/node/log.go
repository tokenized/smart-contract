package node

import (
	"context"
	"os"
	"path"

	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/txbuilder"
	spynodeBootstrap "github.com/tokenized/spynode/cmd/spynoded/bootstrap"
)

// ContextWithLogger wraps the context with a development logger configuration.
func ContextWithLogger(ctx context.Context, isDevelopment, isText bool,
	filePath string) context.Context {

	if len(filePath) > 0 {
		os.MkdirAll(path.Dir(os.Getenv("CLIENT_LOG_FILE_PATH")), os.ModePerm)
	}

	logConfig := logger.NewConfig(isDevelopment, isText, filePath)

	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)
	logConfig.EnableSubSystem(spynodeBootstrap.SubSystem)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

// ContextWithNoLogger removes the logger configuration from the context object.
func ContextWithNoLogger(ctx context.Context) context.Context {
	return logger.ContextWithNoLogger(ctx)
}

// ContextWithOutLogSubSystem removes the logger subsystem configuration from the context.
func ContextWithOutLogSubSystem(ctx context.Context) context.Context {
	return logger.ContextWithOutLogSubSystem(ctx)
}

// ContextWithLogTrace wraps the context with a logger trace value.
func ContextWithLogTrace(ctx context.Context, trace string) context.Context {
	return logger.ContextWithLogTrace(ctx, trace)
}

// Log adds an info level entry to the log.
func Log(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelInfo, 1, format, values...)
}

// LogVerbose adds a verbose level entry to the log.
func LogVerbose(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelVerbose, 1, format, values...)
}

// LogWarn adds a warning level entry to the log.
func LogWarn(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelWarn, 1, format, values...)
}

// LogError adds a error level entry to the log.
func LogError(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelError, 1, format, values...)
}

// LogDepth adds a specified level entry to the log with file data at the specified depth offset in
// the stack.
func LogDepth(ctx context.Context, level logger.Level, depth int, format string,
	values ...interface{}) error {
	return logger.LogDepth(ctx, level, depth+1, format, values...)
}
