package observability

import (
	"errors"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	contextutils "quizapp/internal/utils"
)

// GinMiddleware creates OpenTelemetry middleware for Gin HTTP requests
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// GinMiddlewareWithErrorHandling creates OpenTelemetry middleware with automatic error attribute addition and detailed logging
func GinMiddlewareWithErrorHandling(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use the existing OpenTelemetry middleware
		otelgin.Middleware(serviceName)(c)

		// After the request is processed, check for errors
		c.Next()

		// Get the span from context and add error attributes for failed requests
		if span := trace.SpanFromContext(c.Request.Context()); span != nil {
			statusCode := c.Writer.Status()
			if statusCode >= 400 {
				// Determine error severity based on status code and error types
				severity := determineErrorSeverity(statusCode, c.Errors)

				// Create a more descriptive error message based on status code
				var errorMsg string
				switch {
				case statusCode >= 500:
					errorMsg = "server error"
				case statusCode >= 400:
					errorMsg = "client error"
				default:
					errorMsg = "request failed"
				}

				// Add error details from Gin's error context if available
				if len(c.Errors) > 0 {
					for _, err := range c.Errors {
						if appErr, ok := err.Err.(*contextutils.AppError); ok {
							errorMsg = appErr.Message
							severity = string(appErr.Severity)
							break
						}
						errorMsg = err.Error()
					}
				}

				// Record the error with stack trace
				span.RecordError(errors.New(errorMsg), trace.WithStackTrace(true))
				span.SetStatus(codes.Error, errorMsg)

				// Add additional attributes for better debugging
				span.SetAttributes(
					attribute.Int("http.status_code", statusCode),
					attribute.String("http.method", c.Request.Method),
					attribute.String("http.path", c.Request.URL.Path),
					attribute.String("error.handler", c.HandlerName()),
					attribute.String("error.severity", severity),
				)

				// Add user context if available
				session := sessions.Default(c)
				if userID, ok := session.Get("user_id").(int); ok {
					span.SetAttributes(attribute.Int("error.user_id", userID))
				}

				// Add request body size for debugging
				if c.Request.ContentLength > 0 {
					span.SetAttributes(attribute.Int64("error.request_size", c.Request.ContentLength))
				}

				// Add specific error attributes based on error types
				if len(c.Errors) > 0 {
					for _, err := range c.Errors {
						if appErr, ok := err.Err.(*contextutils.AppError); ok {
							span.SetAttributes(
								attribute.String("error.code", string(appErr.Code)),
								attribute.Bool("error.retryable", contextutils.IsRetryable(appErr)),
							)
							break
						}
					}
				}

				// Add server error specific attributes
				if statusCode >= 500 {
					span.SetAttributes(
						attribute.Bool("error.server_error", true),
					)
				}
			}
		}
	}
}

// determineErrorSeverity determines the severity level based on status code and error types
func determineErrorSeverity(statusCode int, errors []*gin.Error) string {
	// Check for AppError types first
	for _, err := range errors {
		if appErr, ok := err.Err.(*contextutils.AppError); ok {
			return string(appErr.Severity)
		}
	}

	// Fallback to status code based severity
	switch {
	case statusCode >= 500:
		return string(contextutils.SeverityError)
	case statusCode >= 400:
		return string(contextutils.SeverityWarn)
	default:
		return string(contextutils.SeverityInfo)
	}
}
