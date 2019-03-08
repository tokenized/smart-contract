package logger

import (
	"context"
	"errors"
)

// Logger allows you to control logging with message levels and subsystem controls.
// Use the "Include" flags in the Format field to specify which fields should be included in each
//   log message.
// Subsystem log entries can be enabled per subsystem.
// For example the parent package can specify if they want to see logs from a subsystem and how
//   they want to see them.
//
// Sample Setup:
// // Create a log config and set it up.
// logConfig := logger.NewDevelopmentConfig()
// // Log to stderr (default) and main.log.
// // To only log to main.log call SetFile instead of AddFile.
// logConfig.Main.AddFile("./tmp/main.log")
// logConfig.Main.Format |= logger.IncludeSystem
// logConfig.EnableSubSystem(spynode.SubSystem)
//
// // Attach the log config to the context.
// ctx := logger.ContextWithLogConfig(context.Background(), logConfig)
//

type Level int

const (
	LevelDebug   Level = -2
	LevelVerbose Level = -1
	LevelInfo    Level = 0
	LevelWarn    Level = 1
	LevelError   Level = 2
	LevelFatal   Level = 3 // Calls exit
	LevelPanic   Level = 4 // Calls panic
)

// Log entry formatting (which prefix fields to include)
const (
	IncludeDate   = 0x01 // date in the local time zone: 2018/01/01
	IncludeTime   = 0x02 // time in the local time zone: 06:54:32
	IncludeMicro  = 0x04 // microseconds .123123
	IncludeFile   = 0x08 // file name and line number
	IncludeSystem = 0x10 // system name
	IncludeLevel  = 0x20 // level of log entry
)

// Returns a context with the logging config attached.
func ContextWithLogConfig(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, configKey, config)
}

// Returns a context with the logging subsystem attached.
func ContextWithLogSubSystem(ctx context.Context, subsystem string) context.Context {
	return context.WithValue(ctx, subSystemKey, subsystem)
}

// Returns a context with the logging subsystem cleared. Used when a context is passed back from a subsystem.
func ContextWithOutLogSubSystem(ctx context.Context) context.Context {
	return context.WithValue(ctx, subSystemKey, nil)
}

// Log an entry to the main Outputs if:
//   There is no subsystem specified or if the current subsystem is included in the attached
//     Config.IncludedSubSystems.
//   And the level is equal to or above the specified minimum logging level.
// Logs to the Config.SubSystems if the level is above minimum.
func Log(ctx context.Context, level Level, format string, values ...interface{}) error {
	return LogDepth(ctx, level, 1, format, values...)
}

// Debug adds a debug level entry to the log.
func Debug(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelDebug, 1, format, values...)
}

// Verbose adds a verbose level entry to the log.
func Verbose(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelVerbose, 1, format, values...)
}

// Info adds a info level entry to the log.
func Info(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelInfo, 1, format, values...)
}

// Warn adds a warn level entry to the log.
func Warn(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelWarn, 1, format, values...)
}

// Error adds a error level entry to the log.
func Error(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelError, 1, format, values...)
}

// Fatal adds a fatal level entry to the log and then calls os.Exit(1).
func Fatal(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelFatal, 1, format, values...)
}

// Panic adds a panic level entry to the log and then calls panic().
func Panic(ctx context.Context, format string, values ...interface{}) error {
	return LogDepth(ctx, LevelPanic, 1, format, values...)
}

// Same as Log, but the number of levels above the current call in the stack from which to get the
//   file name/line of code can be specified as depth.
func LogDepth(ctx context.Context, level Level, depth int, format string, values ...interface{}) error {
	configValue := ctx.Value(configKey)
	if configValue == nil {
		return nil // Config not specified. Log nothing.
	}

	config, ok := configValue.(*Config)
	if !ok {
		return errors.New("Invalid Config Type")
	}

	config.mutex.Lock()
	defer config.mutex.Unlock()

	subsystem := "Main"
	subsystemValue := ctx.Value(subSystemKey)
	if subsystemValue != nil {
		var ok bool
		subsystem, ok = subsystemValue.(string)
		if !ok {
			return errors.New("Invalid SubSystem Type")
		}
		// Log to subsystem specific config
		subConfig, subExists := config.SubSystems[subsystem]
		if subExists {
			if err := subConfig.log(subsystem, level, depth, format, values...); err != nil {
				return err
			}
		}

		include, includeExists := config.IncludedSubSystems[subsystem]
		if !includeExists || !include {
			return nil // Don't log to main config
		}
	}

	// Log to main config
	return config.Main.log(subsystem, level, depth, format, values...)
}

// Keys for context key/pairs
type loggerkey int

const (
	configKey    loggerkey = 1
	subSystemKey loggerkey = 2
)
