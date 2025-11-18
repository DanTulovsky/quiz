// Package observability provides OpenTelemetry tracing, metrics, and structured logging
// with trace correlation for the quiz application.
package observability

import (
	"context"
	"os"

	"quizapp/internal/config"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps the zap logger with OpenTelemetry context support
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new logger with OpenTelemetry context support and OTLP export
func NewLogger(cfg *config.OpenTelemetryConfig) *Logger {
	return NewLoggerWithLevel(cfg, zap.InfoLevel)
}

// NewLoggerWithLevel creates a new logger with OpenTelemetry context support and OTLP export
func NewLoggerWithLevel(cfg *config.OpenTelemetryConfig, level zapcore.Level) *Logger {
	// If logging is disabled, return a no-op logger
	if cfg == nil || !cfg.EnableLogging {
		return &Logger{Logger: zap.NewNop()}
	}

	// Create a basic zap logger for stdout
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.StacktraceKey = "stacktrace"

	// Use development config if in development mode
	if os.Getenv("ENV") == "development" {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(level)
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		// Fallback to a basic logger if config fails
		zapLogger = zap.NewExample()
	}

	// If OTLP logging is enabled, set up the OTLP exporter
	if cfg.EnableLogging && cfg.Endpoint != "" {
		// Log that we're attempting to set up OTLP export
		zapLogger.Info("Setting up OTLP logging", zap.String("endpoint", cfg.Endpoint), zap.String("protocol", cfg.Protocol))

		// Create OTLP exporter with proper endpoint format
		endpoint := cfg.Endpoint

		// Set up resource attributes
		res, err := resource.New(context.Background(),
			resource.WithAttributes(
				semconv.ServiceName(cfg.ServiceName),
				semconv.ServiceVersion(cfg.ServiceVersion),
			),
		)
		if err != nil {
			// Log the error but continue with stdout logging
			zapLogger.Error("Failed to create otel resource", zap.Error(err))
		} else {
			exporter, err := otlploggrpc.New(context.Background(),
				otlploggrpc.WithEndpoint(endpoint),
				otlploggrpc.WithInsecure(),
			)
			if err != nil {
				// Log the error but continue with stdout logging
				zapLogger.Error("Failed to create OTLP exporter", zap.Error(err), zap.String("endpoint", endpoint))
			} else {
				zapLogger.Info("Successfully created OTLP exporter", zap.String("endpoint", endpoint))

				// Create batch processor
				processor := log.NewBatchProcessor(exporter)

				// Create logger provider with resource
				provider := log.NewLoggerProvider(
					log.WithProcessor(processor),
					log.WithResource(res),
				)

				// Create OpenTelemetry core
				otelCore := otelzap.NewCore("quizapp", otelzap.WithLoggerProvider(provider))

				// Create a new zap logger with both stdout and OTLP cores
				cores := []zapcore.Core{
					zapLogger.Core(),
					otelCore,
				}

				// Create a new logger with multiple cores
				multiCore := zapcore.NewTee(cores...)
				zapLogger = zap.New(multiCore)

				zapLogger.Info("OTLP logging successfully configured", zap.String("endpoint", endpoint))
			}
		}
	} else {
		zapLogger.Info("OTLP logging not enabled", zap.Bool("enable_logging", cfg.EnableLogging), zap.String("endpoint", cfg.Endpoint))
	}

	return &Logger{Logger: zapLogger}
}

// Debug logs a debug message with context
func (l *Logger) Debug(ctx context.Context, msg string, fields ...map[string]interface{}) {
	l.logWithContext(ctx, zap.DebugLevel, msg, fields...)
}

// Info logs an info message with context
func (l *Logger) Info(ctx context.Context, msg string, fields ...map[string]interface{}) {
	l.logWithContext(ctx, zap.InfoLevel, msg, fields...)
}

// Warn logs a warning message with context
func (l *Logger) Warn(ctx context.Context, msg string, fields ...map[string]interface{}) {
	l.logWithContext(ctx, zap.WarnLevel, msg, fields...)
}

// Error logs an error message with context
func (l *Logger) Error(ctx context.Context, msg string, err error, fields ...map[string]interface{}) {
	// Merge fields with error information
	allFields := l.mergeFields(fields...)
	if err != nil {
		allFields["error"] = err.Error()
	}
	l.logWithContext(ctx, zap.ErrorLevel, msg, allFields)
}

// logWithContext logs a message with OpenTelemetry context correlation
func (l *Logger) logWithContext(ctx context.Context, level zapcore.Level, msg string, fields ...map[string]interface{}) {
	// Merge all fields into a single map
	allFields := l.mergeFields(fields...)

	// Add trace context if available
	if span := trace.SpanFromContext(ctx); span != nil {
		spanContext := span.SpanContext()
		if spanContext.IsValid() {
			allFields["trace_id"] = spanContext.TraceID().String()
			allFields["span_id"] = spanContext.SpanID().String()
		}
	}

	// Convert fields to zap fields
	zapFields := make([]zap.Field, 0, len(allFields))
	for k, v := range allFields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	// Log with the appropriate level
	switch level {
	case zap.DebugLevel:
		l.Logger.Debug(msg, zapFields...)
	case zap.InfoLevel:
		l.Logger.Info(msg, zapFields...)
	case zap.WarnLevel:
		l.Logger.Warn(msg, zapFields...)
	case zap.ErrorLevel:
		l.Logger.Error(msg, zapFields...)
	default:
		l.Logger.Info(msg, zapFields...)
	}
}

// mergeFields merges multiple field maps into a single map
func (l *Logger) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return map[string]interface{}{}
	}

	if len(fields) == 1 {
		// Handle nil field map
		if fields[0] == nil {
			return map[string]interface{}{}
		}
		return fields[0]
	}

	// Merge multiple field maps
	merged := make(map[string]interface{})
	for _, fieldMap := range fields {
		// Skip nil field maps
		if fieldMap == nil {
			continue
		}
		for k, v := range fieldMap {
			merged[k] = v
		}
	}
	return merged
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
