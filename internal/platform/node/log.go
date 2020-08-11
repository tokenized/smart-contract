package node

import (
	"context"
	"strings"

	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/spynode"
	"github.com/tokenized/pkg/txbuilder"
)

// ContextWithDevelopmentLogger wraps the context with a development logger configuration.
func ContextWithDevelopmentLogger(ctx context.Context, format string) context.Context {
	var logConfig *logger.Config
	if strings.ToUpper(format) == "TEXT" {
		logConfig = logger.NewDevelopmentTextConfig()
	} else {
		logConfig = logger.NewDevelopmentConfig()
	}

	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

// ContextWithDevelopmentFileLogger wraps the context with a development file logger configuration.
func ContextWithDevelopmentFileLogger(ctx context.Context, logFileName string, format string) context.Context {
	var logConfig *logger.Config
	if strings.ToUpper(format) == "TEXT" {
		logConfig = logger.NewDevelopmentTextConfig()
	} else {
		logConfig = logger.NewDevelopmentConfig()
	}

	logConfig.Main.AddFile(logFileName)
	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

// ContextWithProductionLogger wraps the context with a production logger configuration.
func ContextWithProductionLogger(ctx context.Context, format string) context.Context {
	var logConfig *logger.Config
	if strings.ToUpper(format) == "TEXT" {
		logConfig = logger.NewDevelopmentTextConfig()
	} else {
		logConfig = logger.NewDevelopmentConfig()
	}

	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

// ContextWithDevelopmentFileLogger wraps the context with a production file logger configuration.
func ContextWithProductionFileLogger(ctx context.Context, logFileName string, format string) context.Context {
	var logConfig *logger.Config
	if strings.ToUpper(format) == "TEXT" {
		logConfig = logger.NewDevelopmentTextConfig()
	} else {
		logConfig = logger.NewDevelopmentConfig()
	}

	logConfig.Main.AddFile(logFileName)
	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)

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

// LogDepth adds a specified level entry to the log with file data at the specified depth offset in the stack.
func LogDepth(ctx context.Context, level logger.Level, depth int, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, level, depth+1, format, values...)
}
