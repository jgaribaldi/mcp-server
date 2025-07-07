package logger

import (
	"log/slog"
	"os"
)

// Logger holds the structured logger instance
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level   string
	Format  string
	Service string
	Version string
}

// New creates a new structured logger with the given configuration
func New(cfg Config) (*Logger, error) {
	var level slog.Level
	switch cfg.Level {
	case "DEBUG", "debug":
		level = slog.LevelDebug
	case "INFO", "info":
		level = slog.LevelInfo
	case "WARN", "warn":
		level = slog.LevelWarn
	case "ERROR", "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	
	// Add contextual fields
	logger = logger.With(
		slog.String("service", cfg.Service),
		slog.String("version", cfg.Version),
	)

	return &Logger{Logger: logger}, nil
}

// NewDefault creates a logger with default configuration
func NewDefault() (*Logger, error) {
	return New(Config{
		Level:   "info",
		Format:  "json",
		Service: "mcp-server",
		Version: "dev",
	})
}