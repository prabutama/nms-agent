package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Level maps to slog.Level.
type Level slog.Level

const (
	LevelDebug = Level(slog.LevelDebug)
	LevelInfo  = Level(slog.LevelInfo)
	LevelWarn  = Level(slog.LevelWarn)
	LevelError = Level(slog.LevelError)
)

// Logger wraps slog.Logger to provide a stable API for the agent.
type Logger struct {
	inner *slog.Logger
	level Level
}

// Config configures the Logger.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // text, json
	Output io.Writer
}

func (c Config) WithDefaults() Config {
	out := c
	if out.Level == "" {
		out.Level = "info"
	}
	if out.Format == "" {
		out.Format = "text"
	}
	if out.Output == nil {
		out.Output = os.Stderr
	}
	return out
}

// New creates a Logger from Config.
func New(cfg Config) (*Logger, error) {
	cfg = cfg.WithDefaults()

	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level %q (supported: debug, info, warn, error)", cfg.Level)
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(cfg.Output, opts)
	case "json":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		return nil, fmt.Errorf("unknown log format %q (supported: text, json)", cfg.Format)
	}

	return &Logger{
		inner: slog.New(handler),
		level: Level(lvl),
	}, nil
}

// NewNop returns a Logger that discards all output. Useful for tests.
func NewNop() *Logger {
	return &Logger{inner: slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	if l.inner != nil {
		l.inner.Debug(msg, keysAndValues...)
	}
}

// Info logs at info level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	if l.inner != nil {
		l.inner.Info(msg, keysAndValues...)
	}
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, keysAndValues ...any) {
	if l.inner != nil {
		l.inner.Warn(msg, keysAndValues...)
	}
}

// Error logs at error level.
func (l *Logger) Error(msg string, keysAndValues ...any) {
	if l.inner != nil {
		l.inner.Error(msg, keysAndValues...)
	}
}
