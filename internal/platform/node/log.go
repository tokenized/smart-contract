package node

import (
	"context"
	"io"
	"strings"

	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/rpcnode"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/pkg/spynode"
	"github.com/tokenized/pkg/txbuilder"
)

func ContextWithDevelopmentLogger(ctx context.Context, writer io.Writer, format string) context.Context {
	logConfig := logger.NewDevelopmentConfig()
	logConfig.Main.SetWriter(writer)
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.Main.MinLevel = logger.LevelVerbose
	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)

	if strings.ToUpper(format) == "TEXT" {
		logConfig.IsText = true
	}

	// Configure spynode logs
	logConfig.SubSystems[spynode.SubSystem] = logger.NewDevelopmentSystemConfig()
	logConfig.SubSystems[spynode.SubSystem].Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.SubSystems[spynode.SubSystem].MinLevel = logger.LevelVerbose
	logConfig.SubSystems[spynode.SubSystem].SetWriter(writer)

	return logger.ContextWithLogConfig(ctx, logConfig)
}

func ContextWithProductionLogger(ctx context.Context, writer io.Writer, format string) context.Context {
	logConfig := logger.NewProductionConfig()
	logConfig.Main.SetWriter(writer)
	logConfig.Main.Format |= logger.IncludeSystem | logger.IncludeMicro
	logConfig.EnableSubSystem(rpcnode.SubSystem)
	logConfig.EnableSubSystem(txbuilder.SubSystem)
	logConfig.EnableSubSystem(spynode.SubSystem)
	logConfig.EnableSubSystem(scheduler.SubSystem)

	if strings.ToUpper(format) == "TEXT" {
		logConfig.IsText = true
	}

	return logger.ContextWithLogConfig(ctx, logConfig)
}

func ContextWithNoLogger(ctx context.Context) context.Context {
	return logger.ContextWithNoLogger(ctx)
}

func ContextWithOutLogSubSystem(ctx context.Context) context.Context {
	return logger.ContextWithOutLogSubSystem(ctx)
}

func ContextWithLogTrace(ctx context.Context, trace string) context.Context {
	return logger.ContextWithLogTrace(ctx, trace)
}

// Log adds an info level entry to the log.
func Log(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelInfo, 1, format, values...)
}

// Log adds a verbose level entry to the log.
func LogVerbose(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelVerbose, 1, format, values...)
}

// Log adds a warning level entry to the log.
func LogWarn(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelWarn, 1, format, values...)
}

// Log adds a error level entry to the log.
func LogError(ctx context.Context, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, logger.LevelError, 1, format, values...)
}

// Log adds a specified level entry to the log with file data at the specified depth offset in the stack.
func LogDepth(ctx context.Context, level logger.Level, depth int, format string, values ...interface{}) error {
	return logger.LogDepth(ctx, level, depth+1, format, values...)
}
