// Package metrics provides observability tools including logging, metrics, and tracing.
// This file implements structured logging using Go's log/slog package.
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sentra-lab/mocks/openai/internal/models"
)

// Logger is the global logger instance.
var Logger *slog.Logger

// LogLevel represents logging levels.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat represents log output formats.
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// LogConfig contains logger configuration.
type LogConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level LogLevel

	// Format is the output format (json or text)
	Format LogFormat

	// AddSource adds source file and line number to logs
	AddSource bool

	// Output is the output destination (default: os.Stdout)
	Output *os.File
}

// DefaultLogConfig returns default logger configuration.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:     LevelInfo,
		Format:    FormatJSON,
		AddSource: false,
		Output:    os.Stdout,
	}
}

// InitLogger initializes the global logger with the given configuration.
func InitLogger(config LogConfig) {
	opts := &slog.HandlerOptions{
		Level:     parseLogLevel(config.Level),
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	if config.Format == FormatJSON {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// parseLogLevel converts LogLevel string to slog.Level.
func parseLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context values.
func WithContext(ctx context.Context) *slog.Logger {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}

	// Extract common context values
	logger := Logger

	if requestID := ctx.Value("request_id"); requestID != nil {
		logger = logger.With("request_id", requestID)
	}

	if userID := ctx.Value("user_id"); userID != nil {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// LogRequest logs an incoming HTTP request.
func LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	logger := WithContext(ctx)
	logger.Info("http request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogError logs an API error with context.
func LogError(ctx context.Context, err models.APIError, requestID string) {
	logger := WithContext(ctx)
	logger.Error("api error",
		"request_id", requestID,
		"error_type", string(err.Type),
		"message", err.Message,
		"status_code", err.StatusCode,
		"retry_after", err.RetryAfter,
	)
}

// LogChatCompletion logs a chat completion request.
func LogChatCompletion(ctx context.Context, model string, inputTokens, outputTokens int, cost float64, duration time.Duration) {
	logger := WithContext(ctx)
	logger.Info("chat completion",
		"model", model,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"total_tokens", inputTokens+outputTokens,
		"cost_usd", fmt.Sprintf("%.6f", cost),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogCompletion logs a legacy completion request.
func LogCompletion(ctx context.Context, model string, inputTokens, outputTokens int, cost float64, duration time.Duration) {
	logger := WithContext(ctx)
	logger.Info("completion",
		"model", model,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"total_tokens", inputTokens+outputTokens,
		"cost_usd", fmt.Sprintf("%.6f", cost),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogEmbedding logs an embedding request.
func LogEmbedding(ctx context.Context, model string, inputTokens int, cost float64, duration time.Duration) {
	logger := WithContext(ctx)
	logger.Info("embedding",
		"model", model,
		"input_tokens", inputTokens,
		"cost_usd", fmt.Sprintf("%.6f", cost),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogImageGeneration logs an image generation request.
func LogImageGeneration(ctx context.Context, model string, numImages int, cost float64, duration time.Duration) {
	logger := WithContext(ctx)
	logger.Info("image generation",
		"model", model,
		"num_images", numImages,
		"cost_usd", fmt.Sprintf("%.6f", cost),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogRateLimit logs a rate limit event.
func LogRateLimit(ctx context.Context, apiKey string, limitType string, remaining int, resetIn time.Duration) {
	logger := WithContext(ctx)
	logger.Warn("rate limit",
		"api_key", maskAPIKey(apiKey),
		"limit_type", limitType, // "requests" or "tokens"
		"remaining", remaining,
		"reset_in_seconds", int(resetIn.Seconds()),
	)
}

// LogRateLimitExceeded logs when rate limit is exceeded.
func LogRateLimitExceeded(ctx context.Context, apiKey string, limitType string, resetIn time.Duration) {
	logger := WithContext(ctx)
	logger.Error("rate limit exceeded",
		"api_key", maskAPIKey(apiKey),
		"limit_type", limitType,
		"reset_in_seconds", int(resetIn.Seconds()),
	)
}

// LogCacheHit logs a cache hit.
func LogCacheHit(ctx context.Context, cacheType string, key string) {
	logger := WithContext(ctx)
	logger.Debug("cache hit",
		"cache_type", cacheType,
		"key", maskKey(key),
	)
}

// LogCacheMiss logs a cache miss.
func LogCacheMiss(ctx context.Context, cacheType string, key string) {
	logger := WithContext(ctx)
	logger.Debug("cache miss",
		"cache_type", cacheType,
		"key", maskKey(key),
	)
}

// LogStartup logs application startup.
func LogStartup(port int, mode string) {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}

	Logger.Info("server starting",
		"port", port,
		"mode", mode,
		"version", "1.0.0",
	)
}

// LogShutdown logs application shutdown.
func LogShutdown(reason string) {
	if Logger == nil {
		return
	}

	Logger.Info("server shutting down",
		"reason", reason,
	)
}

// LogStorageError logs a storage operation error.
func LogStorageError(ctx context.Context, operation string, key string, err error) {
	logger := WithContext(ctx)
	logger.Error("storage error",
		"operation", operation,
		"key", maskKey(key),
		"error", err.Error(),
	)
}

// LogTokenizerError logs a tokenizer error.
func LogTokenizerError(ctx context.Context, model string, err error) {
	logger := WithContext(ctx)
	logger.Error("tokenizer error",
		"model", model,
		"error", err.Error(),
	)
}

// LogFixtureLoad logs fixture loading.
func LogFixtureLoad(fixtureType string, count int, duration time.Duration) {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}

	Logger.Info("fixtures loaded",
		"type", fixtureType,
		"count", count,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogFixtureError logs a fixture loading error.
func LogFixtureError(fixtureType string, path string, err error) {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}

	Logger.Error("fixture load error",
		"type", fixtureType,
		"path", path,
		"error", err.Error(),
	)
}

// LogHealthCheck logs a health check result.
func LogHealthCheck(healthy bool, checks map[string]bool) {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}

	if healthy {
		Logger.Info("health check passed",
			"checks", checks,
		)
	} else {
		Logger.Warn("health check failed",
			"checks", checks,
		)
	}
}

// LogPanic logs a panic with stack trace.
func LogPanic(ctx context.Context, recovered interface{}, stack []byte) {
	logger := WithContext(ctx)
	logger.Error("panic recovered",
		"panic", fmt.Sprintf("%v", recovered),
		"stack", string(stack),
	)
}

// maskAPIKey masks an API key for logging (shows only first/last 4 chars).
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// maskKey masks a cache/storage key for logging (shows only first 16 chars).
func maskKey(key string) string {
	if len(key) <= 16 {
		return key
	}
	return key[:16] + "..."
}

// Debug logs a debug message.
func Debug(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// Info logs an info message.
func Info(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// Warn logs a warning message.
func Warn(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// Error logs an error message.
func Error(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// With returns a logger with additional attributes.
func With(args ...any) *slog.Logger {
	if Logger == nil {
		InitLogger(DefaultLogConfig())
	}
	return Logger.With(args...)
}