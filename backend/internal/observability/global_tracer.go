package observability

import (
	"context"
	"fmt"

	"quizapp/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var globalTracer trace.Tracer

// InitGlobalTracer initializes the global tracer for the application.
func InitGlobalTracer() {
	globalTracer = otel.Tracer("quiz-app")
}

// GetGlobalTracer returns the global tracer instance for the application.
func GetGlobalTracer() trace.Tracer {
	if globalTracer == nil {
		// Fallback to default tracer if not initialized
		globalTracer = otel.Tracer("quiz-app")
	}
	return globalTracer
}

// TraceFunction starts a new span with a descriptive name for the given service and function.
func TraceFunction(ctx context.Context, serviceName, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := GetGlobalTracer()
	spanName := fmt.Sprintf("%s.%s", serviceName, functionName)
	return tracer.Start(ctx, spanName, trace.WithAttributes(attributes...))
}

// TraceFunctionWithErrorHandling starts a new span and automatically adds error attributes if the function panics or returns an error.
func TraceFunctionWithErrorHandling(ctx context.Context, serviceName, functionName string, fn func() error, attributes ...attribute.KeyValue) error {
	_, span := TraceFunction(ctx, serviceName, functionName, attributes...)
	defer func() {
		if err := recover(); err != nil {
			span.SetAttributes(
				attribute.Bool("error", true),
				attribute.String("error.type", "panic"),
				attribute.String("error.message", fmt.Sprintf("%v", err)),
			)
			span.End()
			panic(err) // re-panic
		}
	}()

	err := fn()
	if err != nil {
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
	}
	span.End()
	return err
}

// TraceAIFunction starts a new span for an AI service function.
func TraceAIFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "ai", functionName, attributes...)
}

// TraceUserFunction starts a new span for a user service function.
func TraceUserFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "user", functionName, attributes...)
}

// TraceQuestionFunction starts a new span for a question service function.
func TraceQuestionFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "question", functionName, attributes...)
}

// TraceWorkerFunction starts a new span for a worker service function.
func TraceWorkerFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "worker", functionName, attributes...)
}

// TraceLearningFunction starts a new span for a learning service function.
func TraceLearningFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "learning", functionName, attributes...)
}

// TraceHandlerFunction starts a new span for a handler function.
func TraceHandlerFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "handler", functionName, attributes...)
}

// TraceVarietyFunction starts a new span for a variety service function.
func TraceVarietyFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "variety", functionName, attributes...)
}

// TraceOAuthFunction starts a new span for an OAuth service function.
func TraceOAuthFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "oauth", functionName, attributes...)
}

// TraceCleanupFunction starts a new span for a cleanup service function.
func TraceCleanupFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "cleanup", functionName, attributes...)
}

// TraceDatabaseFunction starts a new span for a database function.
func TraceDatabaseFunction(ctx context.Context, functionName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return TraceFunction(ctx, "database", functionName, attributes...)
}

// AttributeQuestion returns a tracing attribute for a question's ID.
func AttributeQuestion(q *models.Question) attribute.KeyValue {
	return attribute.String("question.id", fmt.Sprintf("%d", q.ID))
}

// AttributeQuestionID returns a tracing attribute for a question ID.
func AttributeQuestionID(id int) attribute.KeyValue {
	return attribute.Int("question.id", id)
}

// AttributeUserID returns a tracing attribute for a user ID.
func AttributeUserID(id int) attribute.KeyValue {
	return attribute.Int("user.id", id)
}

// AttributeLanguage returns a tracing attribute for a language.
func AttributeLanguage(lang string) attribute.KeyValue {
	return attribute.String("language", lang)
}

// AttributeLevel returns a tracing attribute for a level.
func AttributeLevel(level string) attribute.KeyValue {
	return attribute.String("level", level)
}

// AttributeQuestionType returns a tracing attribute for a question type.
func AttributeQuestionType(qType interface{}) attribute.KeyValue {
	return attribute.String("question.type", fmt.Sprintf("%v", qType))
}

// AttributeLimit returns a tracing attribute for a limit value.
func AttributeLimit(limit int) attribute.KeyValue {
	return attribute.Int("limit", limit)
}

// AttributePage returns a tracing attribute for a page value.
func AttributePage(page int) attribute.KeyValue {
	return attribute.Int("page", page)
}

// AttributePageSize returns a tracing attribute for a page size value.
func AttributePageSize(size int) attribute.KeyValue {
	return attribute.Int("page_size", size)
}

// AttributeSearch returns a tracing attribute for a search value.
func AttributeSearch(search string) attribute.KeyValue {
	return attribute.String("search", search)
}

// AttributeTypeFilter returns a tracing attribute for a type filter value.
func AttributeTypeFilter(typeFilter string) attribute.KeyValue {
	return attribute.String("type_filter", typeFilter)
}

// AttributeStatusFilter returns a tracing attribute for a status filter value.
func AttributeStatusFilter(statusFilter string) attribute.KeyValue {
	return attribute.String("status_filter", statusFilter)
}
