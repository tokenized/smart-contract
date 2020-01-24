package logger

import (
	"context"
	"fmt"
)

// Logger defines an interface that is compatible with golang's standard log system. To use it
//   configure a context using this packages setup functions, then create a LoggerObject with
//   NewLoggerObject and that context. It is then mostly interchangeable with a golang log object
//   though you still need to use this interface as the type in function parameters.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Panicln(v ...interface{})
}

type LoggerObject struct {
	ctx context.Context
}

func NewLoggerObject(ctx context.Context) *LoggerObject {
	return &LoggerObject{ctx}
}

func (l *LoggerObject) Print(v ...interface{}) {
	LogDepth(l.ctx, LevelInfo, 1, fmt.Sprint(v...))
}

func (l *LoggerObject) Printf(format string, v ...interface{}) {
	LogDepth(l.ctx, LevelInfo, 1, format, v...)
}

func (l *LoggerObject) Println(v ...interface{}) {
	LogDepth(l.ctx, LevelInfo, 1, fmt.Sprint(v...))
}

func (l *LoggerObject) Fatal(v ...interface{}) {
	LogDepth(l.ctx, LevelFatal, 1, fmt.Sprint(v...))
}

func (l *LoggerObject) Fatalf(format string, v ...interface{}) {
	LogDepth(l.ctx, LevelFatal, 1, format, v...)
}

func (l *LoggerObject) Fatalln(v ...interface{}) {
	LogDepth(l.ctx, LevelFatal, 1, fmt.Sprint(v...))
}

func (l *LoggerObject) Panic(v ...interface{}) {
	LogDepth(l.ctx, LevelPanic, 1, fmt.Sprint(v...))
}

func (l *LoggerObject) Panicf(format string, v ...interface{}) {
	LogDepth(l.ctx, LevelPanic, 1, format, v...)
}

func (l *LoggerObject) Panicln(v ...interface{}) {
	LogDepth(l.ctx, LevelPanic, 1, fmt.Sprint(v...))
}
