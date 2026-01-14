// Package logging provides unified logging interface for OpenEAAP.
// It supports structured logging with JSON format, log levels, trace ID
// injection, and contextual fields using zap logger.
package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ============================================================================
// Logger Interface
// ============================================================================

// Logger defines the unified logging interface
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...Field)

	// With adds fields to logger context
	With(fields ...Field) Logger

	// WithContext adds trace ID from context
	WithContext(ctx context.Context) Logger

	// Sync flushes any buffered log entries
	Sync() error
}

// Field represents a log field
type Field = zapcore.Field

// ============================================================================
// ZapLogger Implementation
// ============================================================================

// ZapLogger wraps zap.Logger to implement Logger interface
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger creates a new ZapLogger instance
func NewZapLogger(cfg LogConfig) (*ZapLogger, error) {
	zapConfig := buildZapConfig(cfg)

	logger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &ZapLogger{logger: logger}, nil
}

// NewZapLoggerWithRotation creates logger with file rotation
func NewZapLoggerWithRotation(cfg LogConfig) (*ZapLogger, error) {
	encoder := buildEncoder(cfg)

	// Build core with rotation
	writer := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,    // megabytes
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,
	}

	level := parseLogLevel(cfg.Level)
	core := zapcore.NewCore(encoder, zapcore.AddSync(writer), level)

	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &ZapLogger{logger: logger}, nil
}

// Debug logs a debug message
func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, fields...)
}

// With adds fields to logger context
func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{logger: l.logger.With(fields...)}
}

// WithContext adds trace ID from context
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// ============================================================================
// Configuration
// ============================================================================

// LogConfig defines logging configuration
type LogConfig struct {
	// Log level (debug, info, warn, error, fatal)
	Level string

	// Log format (json, console)
	Format string

	// Output (stdout, stderr, file)
	Output string

	// File path (if output is file)
	FilePath string

	// Max file size in MB
	MaxSize int

	// Max backup files
	MaxBackups int

	// Max age in days
	MaxAge int

	// Enable compression
	Compress bool

	// Enable development mode
	Development bool

	// Enable caller info
	EnableCaller bool

	// Enable stacktrace
	EnableStacktrace bool
}

// buildZapConfig builds zap configuration from LogConfig
func buildZapConfig(cfg LogConfig) zap.Config {
	level := parseLogLevel(cfg.Level)

	var zapConfig zap.Config

	if cfg.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.Encoding = cfg.Format

	// Configure output
	switch cfg.Output {
	case "stdout":
		zapConfig.OutputPaths = []string{"stdout"}
	case "stderr":
		zapConfig.OutputPaths = []string{"stderr"}
	case "file":
		if cfg.FilePath != "" {
			zapConfig.OutputPaths = []string{cfg.FilePath}
		} else {
			zapConfig.OutputPaths = []string{"stdout"}
		}
	default:
		zapConfig.OutputPaths = []string{"stdout"}
	}

	zapConfig.ErrorOutputPaths = []string{"stderr"}

	// Configure encoder
	zapConfig.EncoderConfig = buildEncoderConfig(cfg)

	return zapConfig
}

// buildEncoder builds zapcore encoder
func buildEncoder(cfg LogConfig) zapcore.Encoder {
	encoderConfig := buildEncoderConfig(cfg)

	if cfg.Format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// buildEncoderConfig builds encoder configuration
func buildEncoderConfig(cfg LogConfig) zapcore.EncoderConfig {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Development {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	}

	return encoderConfig
}

// parseLogLevel parses string log level to zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// ============================================================================
// Context Integration
// ============================================================================

// Context keys for logging
type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetTraceID retrieves trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// extractContextFields extracts logging fields from context
func extractContextFields(ctx context.Context) []Field {
	var fields []Field

	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if userID := GetUserID(ctx); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	return fields
}

// ============================================================================
// Field Constructors
// ============================================================================

// String creates a string field
func String(key, val string) Field {
	return zap.String(key, val)
}

// Int creates an int field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Float64 creates a float64 field
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Error creates an error field
func Error(err error) Field {
	return zap.Error(err)
}

// Time creates a time field
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Duration creates a duration field
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Any creates a field from any value
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Strings creates a string array field
func Strings(key string, val []string) Field {
	return zap.Strings(key, val)
}

// Namespace creates a namespace field
func Namespace(key string) Field {
	return zap.Namespace(key)
}

// ============================================================================
// Global Logger
// ============================================================================

var globalLogger Logger

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() Logger {
	if globalLogger == nil {
		// Create default logger if not set
		cfg := LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}
		logger, _ := NewZapLogger(cfg)
		globalLogger = logger
	}
	return globalLogger
}

// ============================================================================
// Convenience Functions
// ============================================================================

// Debug logs a debug message using global logger
func Debug(msg string, fields ...Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Info logs an info message using global logger
func Info(msg string, fields ...Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Warn logs a warning message using global logger
func Warn(msg string, fields ...Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// ErrorLog logs an error message using global logger
func ErrorLog(msg string, fields ...Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Fatal logs a fatal message using global logger
func Fatal(msg string, fields ...Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

// ============================================================================
// Logger Factory
// ============================================================================

// NewLogger creates a new logger with default configuration
func NewLogger() (Logger, error) {
	cfg := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	return NewZapLogger(cfg)
}

// NewDevelopmentLogger creates a logger for development
func NewDevelopmentLogger() (Logger, error) {
	cfg := LogConfig{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		Development: true,
	}
	return NewZapLogger(cfg)
}

// NewProductionLogger creates a logger for production
func NewProductionLogger() (Logger, error) {
	cfg := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	return NewZapLogger(cfg)
}

// NewFileLogger creates a logger that writes to file
func NewFileLogger(filepath string) (Logger, error) {
	cfg := LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		FilePath:   filepath,
		MaxSize:    100,  // 100 MB
		MaxBackups: 3,
		MaxAge:     7,    // 7 days
		Compress:   true,
	}
	return NewZapLoggerWithRotation(cfg)
}

// ============================================================================
// No-op Logger
// ============================================================================

// NoopLogger is a logger that does nothing
type NoopLogger struct{}

// NewNoopLogger creates a no-op logger
func NewNoopLogger() Logger {
	return &NoopLogger{}
}

func (l *NoopLogger) Debug(msg string, fields ...Field)            {}
func (l *NoopLogger) Info(msg string, fields ...Field)             {}
func (l *NoopLogger) Warn(msg string, fields ...Field)             {}
func (l *NoopLogger) Error(msg string, fields ...Field)            {}
func (l *NoopLogger) Fatal(msg string, fields ...Field)            { os.Exit(1) }
func (l *NoopLogger) With(fields ...Field) Logger                  { return l }
func (l *NoopLogger) WithContext(ctx context.Context) Logger       { return l }
func (l *NoopLogger) Sync() error                                  { return nil }

//Personal.AI order the ending
