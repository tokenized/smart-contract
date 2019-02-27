package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// SystemConfig defines the configuration the main system or a subsystem with custom settings.
type SystemConfig struct {
	Output   io.Writer // Output(s) for log entries (stderr, files, …)
	MinLevel Level     // Minimum level to log. Below this are ignored.
	Format   int       // Controls what is shown in log entry
}

// Creates a new system config with default production values.
//   Logs info level and above to stderr.
func NewProductionSystemConfig() *SystemConfig {
	result := SystemConfig{
		Output:   os.Stderr,
		MinLevel: Info,
		Format:   IncludeDate | IncludeTime | IncludeFile | IncludeLevel,
	}
	return &result
}

// Creates a new system config with default development values.
//   Logs verbose level and above to stderr.
func NewDevelopmentSystemConfig() *SystemConfig {
	result := SystemConfig{
		Output:   os.Stderr,
		MinLevel: Verbose,
		Format:   IncludeDate | IncludeTime | IncludeFile | IncludeLevel,
	}
	return &result
}

// Adds a file to the existing log outputs
func (config *SystemConfig) AddFile(filePath string) error {
	logFileName := filepath.FromSlash(filePath)
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return err
	}

	config.Output = io.MultiWriter(config.Output, logFile)
	return nil
}

// Sets a file as the only log output
func (config *SystemConfig) SetFile(filePath string) error {
	logFileName := filepath.FromSlash(filePath)
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return err
	}

	config.Output = logFile
	return nil
}

// Adds a writer to the existing log outputs
func (config *SystemConfig) AddWriter(writer io.Writer) {
	config.Output = io.MultiWriter(config.Output, writer)
}

// Sets a writer as the only log output
func (config *SystemConfig) SetWriter(writer io.Writer) {
	config.Output = writer
}

// Logs an entry based on the system config
func (config *SystemConfig) log(system string, level Level, depth int, format string, values ...interface{}) error {
	if config.MinLevel > level {
		return nil // Level is below minimum
	}

	// Create log entry
	now := time.Now()
	entry := make([]byte, 0, 1024)

	// Append Date
	if config.Format&IncludeDate != 0 {
		year, month, day := now.Date()
		entry = append(entry, fmt.Sprintf("%04d/%02d/%02d ", year, month, day)...)
	}

	// Append Time
	if config.Format&IncludeTime != 0 {
		hour, min, sec := now.Clock()
		entry = append(entry, fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)...)
		if config.Format&IncludeMicro == 0 {
			entry = append(entry, ' ')
		}
	}

	// Append microseconds
	if config.Format&IncludeMicro != 0 {
		if config.Format&IncludeTime != 0 {
			entry = append(entry, '.')
		}

		entry = append(entry, fmt.Sprintf("%06d", now.Nanosecond()/1e3)...)
		entry = append(entry, ' ')
	}

	// Append System
	if config.Format&IncludeSystem != 0 {
		entry = append(entry, '[')
		entry = append(entry, system...)
		entry = append(entry, ']')
		entry = append(entry, ' ')
	}

	// Append File
	if config.Format&IncludeFile != 0 {
		_, file, line, ok := runtime.Caller(2 + depth) // Code of interest is 2 levels up in stack
		if ok {
			file = filepath.Base(file)
		} else {
			file = "???"
			line = 0
		}

		entry = append(entry, file...)
		entry = append(entry, ':')
		entry = append(entry, fmt.Sprintf("%d ", line)...)
	}

	// Append Level
	if config.Format&IncludeLevel != 0 {
		switch level {
		case Debug:
			entry = append(entry, []byte("Debug - ")...)
		case Verbose:
			entry = append(entry, []byte("Verbose - ")...)
		case Info:
			entry = append(entry, []byte("Info - ")...)
		case Warn:
			entry = append(entry, []byte("Warn - ")...)
		case Error:
			entry = append(entry, []byte("Error - ")...)
		case Fatal:
			entry = append(entry, []byte("Fatal - ")...)
		case Panic:
			entry = append(entry, []byte("Panic - ")...)
		}
	}

	// Append actual log entry
	entry = append(entry, fmt.Sprintf(format, values...)...)

	// Append new line
	if entry[len(entry)-1] != '\n' {
		entry = append(entry, '\n')
	}

	// Write to output
	_, err := config.Output.Write(entry)

	if level == Fatal {
		os.Exit(1)
	}
	if level == Panic {
		panic(entry)
	}
	return err
}
