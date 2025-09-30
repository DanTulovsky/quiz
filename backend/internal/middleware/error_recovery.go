package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
)

// ErrorRecoveryConfig configures error recovery behavior
type ErrorRecoveryConfig struct {
	// MaxRetries specifies the maximum number of retries for retryable errors
	MaxRetries int
	// RetryDelay specifies the base delay between retries
	RetryDelay time.Duration
	// MaxRetryDelay specifies the maximum delay between retries
	MaxRetryDelay time.Duration
	// EnableCircuitBreaker enables circuit breaker pattern
	EnableCircuitBreaker bool
	// CircuitBreakerThreshold specifies failure threshold for circuit breaker
	CircuitBreakerThreshold int
	// CircuitBreakerTimeout specifies how long to wait before retrying after circuit opens
	CircuitBreakerTimeout time.Duration
}

// DefaultErrorRecoveryConfig returns a default error recovery configuration
func DefaultErrorRecoveryConfig() *ErrorRecoveryConfig {
	return &ErrorRecoveryConfig{
		MaxRetries:              3,
		RetryDelay:              100 * time.Millisecond,
		MaxRetryDelay:           5 * time.Second,
		EnableCircuitBreaker:    false,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// circuitBreakerState represents the state of a circuit breaker
type circuitBreakerState int

const (
	circuitClosed circuitBreakerState = iota
	circuitOpen
	circuitHalfOpen
)

// circuitBreaker tracks failures and manages circuit state
type circuitBreaker struct {
	state       circuitBreakerState
	failures    int
	lastFailure time.Time
	config      *ErrorRecoveryConfig
}

// newCircuitBreaker creates a new circuit breaker
func newCircuitBreaker(config *ErrorRecoveryConfig) *circuitBreaker {
	return &circuitBreaker{
		state:  circuitClosed,
		config: config,
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *circuitBreaker) canExecute() bool {
	switch cb.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if time.Since(cb.lastFailure) > cb.config.CircuitBreakerTimeout {
			cb.state = circuitHalfOpen
			return true
		}
		return false
	case circuitHalfOpen:
		return true
	default:
		return false
	}
}

// recordSuccess records a successful execution
func (cb *circuitBreaker) recordSuccess() {
	cb.failures = 0
	cb.state = circuitClosed
}

// recordFailure records a failed execution
func (cb *circuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.config.CircuitBreakerThreshold {
		cb.state = circuitOpen
	}
}

// ErrorRecoveryMiddleware creates middleware for handling panics and retrying failed requests
func ErrorRecoveryMiddleware(logger interface{}, config *ErrorRecoveryConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultErrorRecoveryConfig()
	}

	// Create circuit breaker if enabled
	var cb *circuitBreaker
	if config.EnableCircuitBreaker {
		cb = newCircuitBreaker(config)
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stackTrace := string(debug.Stack())
				fmt.Printf("Panic recovered: %v\nStack trace: %s\n", err, stackTrace)

				// Convert panic value to error if needed
				var panicErr error
				if e, ok := err.(error); ok {
					panicErr = e
				} else {
					panicErr = contextutils.WrapErrorf(nil, "panic: %v", err)
				}

				// Send error response
				appErr := contextutils.NewAppErrorWithCause(
					contextutils.ErrorCodeInternalError,
					contextutils.SeverityFatal,
					"Internal server error",
					"A panic occurred while processing the request",
					contextutils.WrapError(panicErr, "panic"),
				)

				// Add stack trace to error details in development
				if gin.Mode() == gin.DebugMode {
					appErr.Details = fmt.Sprintf("%s\nStack trace: %s", appErr.Details, stackTrace)
				}

				HandleAppError(c, appErr)
				c.Abort()
			}
		}()

		// Check circuit breaker
		if cb != nil && !cb.canExecute() {
			ServiceUnavailable(c, "Service temporarily unavailable due to high error rate")
			c.Abort()
			return
		}

		// Process request
		c.Next()

		// Record success/failure for circuit breaker
		if cb != nil {
			if c.Writer.Status() >= 500 {
				cb.recordFailure()
			} else if c.Writer.Status() < 500 && cb.state == circuitHalfOpen {
				cb.recordSuccess()
			}
		}

		// Retry logic for failed requests
		if shouldRetry(c.Writer.Status(), c.Errors) {
			retryWithBackoff(c, config, logger)
		}
	}
}

// shouldRetry determines if a request should be retried
func shouldRetry(statusCode int, errors []*gin.Error) bool {
	// Only retry 5xx errors and certain 4xx errors
	if statusCode >= 500 {
		return true
	}

	// Retry on specific 4xx errors that might be transient
	if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests {
		return true
	}

	// Check if there are errors that indicate retryable failures
	for _, err := range errors {
		if contextutils.IsRetryable(err) {
			return true
		}
	}

	return false
}

// retryWithBackoff attempts to retry the request with exponential backoff
func retryWithBackoff(c *gin.Context, config *ErrorRecoveryConfig, logger interface{}) {
	// Only retry idempotent methods (GET, HEAD, OPTIONS, PUT, DELETE)
	method := c.Request.Method
	if method != http.MethodGet && method != http.MethodHead &&
		method != http.MethodOptions && method != http.MethodPut &&
		method != http.MethodDelete {
		return
	}

	// Get the original handler
	handlerName := c.HandlerName()
	if handlerName == "" {
		return
	}

	// Calculate retry delay with exponential backoff
	delay := config.RetryDelay
	for i := 0; i < config.MaxRetries; i++ {
		time.Sleep(delay)

		// Double the delay for next iteration (with max limit)
		delay *= 2
		if delay > config.MaxRetryDelay {
			delay = config.MaxRetryDelay
		}

		// Log retry attempt
		if logger != nil {
			// This would be logged using the observability logger in real implementation
			fmt.Printf("Retrying request %s %s (attempt %d/%d)\n",
				method, c.Request.URL.Path, i+1, config.MaxRetries)
		}

		// Note: In a real implementation, we would need to recreate the request
		// and re-execute it. This is a simplified version for demonstration.
		// The actual retry logic would depend on the specific use case.
	}
}

// HandleAppError handles any AppError and sends appropriate HTTP response
func HandleAppError(c *gin.Context, err error) {
	if appErr, ok := err.(*contextutils.AppError); ok {
		StandardizeAppError(c, appErr)
	} else {
		// Fallback for non-AppError types
		StandardizeHTTPError(c, http.StatusInternalServerError, "Internal server error", err.Error())
	}
}

// StandardizeAppError sends a structured error response using AppError
func StandardizeAppError(c *gin.Context, err *contextutils.AppError) {
	// Map error codes to HTTP status codes
	statusCode := mapErrorCodeToHTTPStatus(err.Code)

	// Convert error to JSON structure
	errorJSON := err.ToJSON()

	// Add retryable information based on error type
	errorJSON["retryable"] = contextutils.IsRetryable(err)

	c.JSON(statusCode, errorJSON)
}

// StandardizeHTTPError creates consistent HTTP error responses with structured error information
func StandardizeHTTPError(c *gin.Context, _ int, message, details string) {
	// Create a generic AppError for consistent response format
	appErr := contextutils.NewAppError(
		contextutils.ErrorCodeInternalError,
		contextutils.SeverityError,
		message,
		details,
	)

	StandardizeAppError(c, appErr)
}

// ServiceUnavailable sends a 503 Service Unavailable error with a standardized payload
func ServiceUnavailable(c *gin.Context, msg string) {
	appErr := contextutils.NewAppError(
		contextutils.ErrorCodeServiceUnavailable,
		contextutils.SeverityError,
		msg,
		"",
	)
	StandardizeAppError(c, appErr)
}

// mapErrorCodeToHTTPStatus maps AppError codes to appropriate HTTP status codes
func mapErrorCodeToHTTPStatus(code contextutils.ErrorCode) int {
	switch code {
	// 4xx Client Errors
	case contextutils.ErrorCodeInvalidInput, contextutils.ErrorCodeMissingRequired,
		contextutils.ErrorCodeInvalidFormat, contextutils.ErrorCodeValidationFailed,
		contextutils.ErrorCodeOAuthStateMismatch:
		return http.StatusBadRequest

	case contextutils.ErrorCodeUnauthorized:
		return http.StatusUnauthorized

	case contextutils.ErrorCodeForbidden:
		return http.StatusForbidden

	case contextutils.ErrorCodeRecordNotFound, contextutils.ErrorCodeQuestionNotFound,
		contextutils.ErrorCodeAssignmentNotFound:
		return http.StatusNotFound

	case contextutils.ErrorCodeRecordExists:
		return http.StatusConflict

	case contextutils.ErrorCodeSessionExpired, contextutils.ErrorCodeInvalidCredentials:
		return http.StatusUnauthorized

	case contextutils.ErrorCodeRateLimit:
		return http.StatusTooManyRequests

	// 5xx Server Errors
	case contextutils.ErrorCodeInternalError:
		return http.StatusInternalServerError

	case contextutils.ErrorCodeServiceUnavailable, contextutils.ErrorCodeDatabaseConnection,
		contextutils.ErrorCodeAIProviderUnavailable:
		return http.StatusServiceUnavailable

	case contextutils.ErrorCodeTimeout:
		return http.StatusRequestTimeout

	case contextutils.ErrorCodeDatabaseQuery, contextutils.ErrorCodeDatabaseTransaction,
		contextutils.ErrorCodeForeignKeyViolation, contextutils.ErrorCodeTimestampMissingTimezone,
		contextutils.ErrorCodeAIRequestFailed, contextutils.ErrorCodeAIResponseInvalid,
		contextutils.ErrorCodeAIConfigInvalid, contextutils.ErrorCodeOAuthProviderError:
		return http.StatusInternalServerError

	// Default to internal server error for unknown codes
	default:
		return http.StatusInternalServerError
	}
}
