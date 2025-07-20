package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
	
	"github.com/lmittmann/tint"
)

// Logger holds the structured logger instance
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level     string
	Format    string
	Service   string
	Version   string
	UseEmojis bool
}

// EmojiHandler wraps another handler to add emoji prefixes to log messages
type EmojiHandler struct {
	handler slog.Handler
	emojis  map[slog.Level]string
}

func NewEmojiHandler(h slog.Handler) *EmojiHandler {
	return &EmojiHandler{
		handler: h,
		emojis: map[slog.Level]string{
			slog.LevelDebug: "🔍",  // Magnifying glass - investigating/debugging
			slog.LevelInfo:  "✅",  // Green tick - success/normal operations
			slog.LevelWarn:  "⚠️",   // Warning sign - caution
			slog.LevelError: "❌",  // Red X - errors/failures
		},
	}
}

func (h *EmojiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *EmojiHandler) Handle(ctx context.Context, r slog.Record) error {
	if emoji, ok := h.emojis[r.Level]; ok {
		newRecord := slog.NewRecord(r.Time, r.Level, emoji+" "+r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool {
			newRecord.AddAttrs(a)
			return true
		})
		return h.handler.Handle(ctx, newRecord)
	}
	return h.handler.Handle(ctx, r)
}

func (h *EmojiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &EmojiHandler{handler: h.handler.WithAttrs(attrs), emojis: h.emojis}
}

func (h *EmojiHandler) WithGroup(name string) slog.Handler {
	return &EmojiHandler{handler: h.handler.WithGroup(name), emojis: h.emojis}
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

	var handler slog.Handler
	
	switch cfg.Format {
	case "json":
		opts := &slog.HandlerOptions{Level: level}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "console":
		baseHandler := tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly, // "15:04:05" format
			NoColor:    false,         // Auto-detect terminal
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					var levelStr string
					switch a.Value.Any().(slog.Level) {
					case slog.LevelDebug:
						levelStr = "DBG"
					case slog.LevelInfo:
						levelStr = "INF"
					case slog.LevelWarn:
						levelStr = "WRN"
					case slog.LevelError:
						levelStr = "ERR"
					default:
						levelStr = a.Value.String()
					}
					boldLevel := "\033[1m" + levelStr + "\033[0m"
					return slog.Attr{Key: a.Key, Value: slog.StringValue(boldLevel)}
				}
				return a
			},
		})
		
		if cfg.UseEmojis {
			handler = NewEmojiHandler(baseHandler)
		} else {
			handler = baseHandler
		}
	default: // "text"
		opts := &slog.HandlerOptions{Level: level}
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	
	// Add contextual fields (skip for console to reduce noise)
	if cfg.Format != "console" {
		logger = logger.With(
			slog.String("service", cfg.Service),
			slog.String("version", cfg.Version),
		)
	}

	return &Logger{Logger: logger}, nil
}

// NewDefault creates a logger with default configuration
func NewDefault() (*Logger, error) {
	return New(Config{
		Level:     "info",
		Format:    "console",
		Service:   "mcp-server",
		Version:   "dev",
		UseEmojis: true,
	})
}