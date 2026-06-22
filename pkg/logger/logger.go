// pkg/logger/logger.go

package logger

import (
    "context"
    "log/slog"
    "os"
    "time"
)

// Logger is the global logger instance
var Logger *slog.Logger

// ContextKey defines the type for context keys used by the logger
type ContextKey string

const (
    CorrelationIDKey ContextKey = "correlation_id"
    WhoKey           ContextKey = "who"
    WhereKey         ContextKey = "where"
)

// Init initializes the global logger with the specified log level and format
func Init(level string) {
    var l slog.Level
    switch level {
    case "debug":
        l = slog.LevelDebug
    case "info":
        l = slog.LevelInfo
    case "warn":
        l = slog.LevelWarn
    case "error":
        l = slog.LevelError
    default:
        l = slog.LevelInfo
    }

    format := os.Getenv("LOG_FORMAT")
    if format == "" {
        format = "json"
    }

    var handler slog.Handler
    if format == "text" {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level:     l,
            AddSource: true,
        })
    } else {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:     l,
            AddSource: true,
        })
    }

    Logger = slog.New(handler)
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
        return id
    }
    return "unknown"
}

// GetWho extracts the IAM principal from context
func GetWho(ctx context.Context) string {
    if who, ok := ctx.Value(WhoKey).(string); ok {
        return who
    }
    return "unknown"
}

// GetWhere extracts the source IP from context
func GetWhere(ctx context.Context) string {
    if where, ok := ctx.Value(WhereKey).(string); ok {
        return where
    }
    return "unknown"
}

// logStructured is the internal function that logs structured JSON
func logStructured(ctx context.Context, level slog.Level, msg string, args ...interface{}) {
    if Logger == nil {
        Init("info")
    }

    correlationID := GetCorrelationID(ctx)
    who := GetWho(ctx)
    where := GetWhere(ctx)

    // Build the log entry with standard fields
    logAttrs := []slog.Attr{
        slog.String("timestamp", time.Now().UTC().Format(time.RFC3339Nano)),
        slog.String("correlation_id", correlationID),
        slog.String("who", who),
        slog.String("where", where),
    }

    // Add extra args with validation
    if len(args) > 0 {
        for i := 0; i < len(args)-1; i += 2 {
            if key, ok := args[i].(string); ok {
                logAttrs = append(logAttrs, slog.Any(key, args[i+1]))
            } else {
                // Skip malformed key-value pair
                Logger.Warn("Malformed log argument - expected key string", "key", args[i])
            }
        }
    }

    // Log the message
    Logger.LogAttrs(ctx, level, msg, logAttrs...)
}

// Debug logs a debug message with structured fields
func Debug(ctx context.Context, msg string, args ...interface{}) {
    logStructured(ctx, slog.LevelDebug, msg, args...)
}

// Info logs an info message with structured fields
func Info(ctx context.Context, msg string, args ...interface{}) {
    logStructured(ctx, slog.LevelInfo, msg, args...)
}

// Warn logs a warning message with structured fields
func Warn(ctx context.Context, msg string, args ...interface{}) {
    logStructured(ctx, slog.LevelWarn, msg, args...)
}

// Error logs an error message with structured fields
func Error(ctx context.Context, msg string, args ...interface{}) {
    logStructured(ctx, slog.LevelError, msg, args...)
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithWho adds an IAM principal to the context
func WithWho(ctx context.Context, who string) context.Context {
    return context.WithValue(ctx, WhoKey, who)
}

// WithWhere adds a source IP to the context
func WithWhere(ctx context.Context, where string) context.Context {
    return context.WithValue(ctx, WhereKey, where)
}