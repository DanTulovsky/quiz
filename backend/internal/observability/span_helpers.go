package observability

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// FinishSpan ends a span and records any error pointed to by errPtr.
// Use with a named error return: `defer observability.FinishSpan(span, &err)`
func FinishSpan(span trace.Span, errPtr *error) {
	if span == nil {
		return
	}
	if errPtr != nil && *errPtr != nil {
		span.RecordError(*errPtr, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, (*errPtr).Error())
	}
	span.End()
}
