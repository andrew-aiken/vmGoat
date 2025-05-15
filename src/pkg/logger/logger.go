package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is the interface that wraps the basic logging methods
type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
	Fatal() *zerolog.Event
}

var (
	// DefaultLogger is the default logger instance
	DefaultLogger Logger
)

// Config holds the logger configuration
type Config struct {
	// Debug mode will only show timestamp and level and set log level to debug
	Debug bool
}

// Init initializes the logger with the specified configuration
func Init(config Config) {
	// Set log level based on debug mode
	logLevel := zerolog.InfoLevel
	if config.Debug {
		logLevel = zerolog.DebugLevel
	}

	// Configure zerolog
	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	// Configure output format based on debug mode
	if config.Debug {
		// In debug mode, only show timestamp and level
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			// PartsOrder: []string{
			// 	zerolog.TimestampFieldName,
			// 	zerolog.LevelFieldName,
			// 	zerolog.MessageFieldName,
			// },
		}
		log.Logger = log.Output(writer)
	} else {
		// Pretty console writer in development
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			PartsOrder: []string{
				// zerolog.TimestampFieldName,
				// zerolog.LevelFieldName,
				zerolog.MessageFieldName,
			},
		})
	}

	// Create a pointer to the logger
	logger := &log.Logger
	DefaultLogger = logger
}

// Get returns the default logger instance
func Get() Logger {
	if DefaultLogger == nil {
		// Initialize with default config if not initialized
		Init(Config{
			Debug: false,
		})
	}
	return DefaultLogger
}
