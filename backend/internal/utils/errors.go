// Package contextutils provides error handling utilities and standardized error types
// for consistent error management across the quiz application.
package contextutils

import (
	"context"
	"fmt"
	"strings"
)

// ErrorCode represents a standardized error code for API responses
type ErrorCode string

const (
	// Database error codes

	// ErrorCodeDatabaseConnection indicates a database connection error
	ErrorCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION_ERROR"
	// ErrorCodeDatabaseQuery indicates a database query error
	ErrorCodeDatabaseQuery ErrorCode = "DATABASE_QUERY_ERROR"
	// ErrorCodeDatabaseTransaction indicates a database transaction error
	ErrorCodeDatabaseTransaction ErrorCode = "DATABASE_TRANSACTION_ERROR"
	// ErrorCodeRecordNotFound indicates that a requested record was not found
	ErrorCodeRecordNotFound ErrorCode = "RECORD_NOT_FOUND"
	// ErrorCodeRecordExists indicates that a record already exists (duplicate key)
	ErrorCodeRecordExists ErrorCode = "RECORD_ALREADY_EXISTS"
	// ErrorCodeForeignKeyViolation indicates a foreign key constraint violation
	ErrorCodeForeignKeyViolation ErrorCode = "FOREIGN_KEY_VIOLATION"

	// Validation error codes

	// ErrorCodeInvalidInput indicates that the provided input is invalid
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"
	// ErrorCodeMissingRequired indicates that a required field is missing
	ErrorCodeMissingRequired ErrorCode = "MISSING_REQUIRED_FIELD"
	// ErrorCodeInvalidFormat indicates that the input format is invalid
	ErrorCodeInvalidFormat ErrorCode = "INVALID_FORMAT"
	// ErrorCodeValidationFailed indicates that validation has failed
	ErrorCodeValidationFailed ErrorCode = "VALIDATION_FAILED"

	// Authentication error codes

	// ErrorCodeUnauthorized indicates that the user is not authorized
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrorCodeForbidden indicates that the user is forbidden from accessing the resource
	ErrorCodeForbidden ErrorCode = "FORBIDDEN"
	// ErrorCodeInvalidCredentials indicates that the provided credentials are invalid
	ErrorCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	// ErrorCodeSessionExpired indicates that the user session has expired
	ErrorCodeSessionExpired ErrorCode = "SESSION_EXPIRED"

	// Service error codes

	// ErrorCodeServiceUnavailable indicates that the service is temporarily unavailable
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	// ErrorCodeTimeout indicates that a request has timed out
	ErrorCodeTimeout ErrorCode = "REQUEST_TIMEOUT"
	// ErrorCodeRateLimit indicates that the rate limit has been exceeded
	ErrorCodeRateLimit ErrorCode = "RATE_LIMIT_EXCEEDED"
	// ErrorCodeQuotaExceeded indicates that the usage quota has been exceeded
	ErrorCodeQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"
	// ErrorCodeInternalError indicates an internal server error
	ErrorCodeInternalError ErrorCode = "INTERNAL_SERVER_ERROR"
	// ErrorCodeAssignmentNotFound indicates that a question assignment was not found
	ErrorCodeAssignmentNotFound ErrorCode = "ASSIGNMENT_NOT_FOUND"
	// ErrorCodeConflict indicates that an operation conflicts with the current state
	ErrorCodeConflict ErrorCode = "CONFLICT"

	// Question error codes

	// ErrorCodeTimestampMissingTimezone indicates that a timestamp is missing timezone information
	ErrorCodeTimestampMissingTimezone ErrorCode = "TIMESTAMP_MISSING_TIMEZONE"
	// ErrorCodeNoQuestionsAvailable indicates that no questions are available
	ErrorCodeNoQuestionsAvailable ErrorCode = "NO_QUESTIONS_AVAILABLE"
	// ErrorCodeQuestionAlreadyAnswered indicates that the question has already been answered
	ErrorCodeQuestionAlreadyAnswered ErrorCode = "QUESTION_ALREADY_ANSWERED"
	// ErrorCodeQuestionNotFound indicates that the requested question was not found
	ErrorCodeQuestionNotFound ErrorCode = "QUESTION_NOT_FOUND"
	// ErrorCodeInvalidAnswerIndex indicates that the answer index is invalid
	ErrorCodeInvalidAnswerIndex ErrorCode = "INVALID_ANSWER_INDEX"
	// ErrorCodeGenerationLimitReached indicates that the daily generation limit has been reached
	ErrorCodeGenerationLimitReached ErrorCode = "GENERATION_LIMIT_REACHED"

	// AI Service error codes

	// ErrorCodeAIProviderUnavailable indicates that the AI provider is unavailable
	ErrorCodeAIProviderUnavailable ErrorCode = "AI_PROVIDER_UNAVAILABLE"
	// ErrorCodeAIRequestFailed indicates that the AI request failed
	ErrorCodeAIRequestFailed ErrorCode = "AI_REQUEST_FAILED"
	// ErrorCodeAIResponseInvalid indicates that the AI response is invalid
	ErrorCodeAIResponseInvalid ErrorCode = "AI_RESPONSE_INVALID"
	// ErrorCodeAIConfigInvalid indicates that the AI configuration is invalid
	ErrorCodeAIConfigInvalid ErrorCode = "AI_CONFIG_INVALID"

	// OAuth error codes

	// ErrorCodeOAuthCodeExpired indicates that the OAuth authorization code has expired
	ErrorCodeOAuthCodeExpired ErrorCode = "OAUTH_CODE_EXPIRED"
	// ErrorCodeOAuthStateMismatch indicates that the OAuth state parameter does not match
	ErrorCodeOAuthStateMismatch ErrorCode = "OAUTH_STATE_MISMATCH"
	// ErrorCodeOAuthProviderError indicates an error from the OAuth provider
	ErrorCodeOAuthProviderError ErrorCode = "OAUTH_PROVIDER_ERROR"
)

// SeverityLevel represents the severity of an error for logging and monitoring
type SeverityLevel string

const (
	// SeverityDebug indicates debug-level errors for development
	SeverityDebug SeverityLevel = "debug"
	// SeverityInfo indicates informational errors
	SeverityInfo SeverityLevel = "info"
	// SeverityWarn indicates warning-level errors
	SeverityWarn SeverityLevel = "warn"
	// SeverityError indicates error-level issues
	SeverityError SeverityLevel = "error"
	// SeverityFatal indicates fatal errors that require immediate attention
	SeverityFatal SeverityLevel = "fatal"
)

// AppError represents a structured error with code, severity, and context
type AppError struct {
	Code     ErrorCode
	Severity SeverityLevel
	Message  string
	Details  string
	Cause    error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is
func (e *AppError) Is(target error) bool {
	if appErr, ok := target.(*AppError); ok {
		return e.Code == appErr.Code
	}
	return false
}

// Error types for consistent error handling with associated codes and severity
var (
	// Database errors
	ErrDatabaseConnection = &AppError{
		Code:     ErrorCodeDatabaseConnection,
		Severity: SeverityError,
		Message:  "Database connection failed",
	}

	ErrDatabaseQuery = &AppError{
		Code:     ErrorCodeDatabaseQuery,
		Severity: SeverityError,
		Message:  "Database query failed",
	}

	ErrDatabaseTransaction = &AppError{
		Code:     ErrorCodeDatabaseTransaction,
		Severity: SeverityError,
		Message:  "Database transaction failed",
	}

	ErrRecordNotFound = &AppError{
		Code:     ErrorCodeRecordNotFound,
		Severity: SeverityInfo,
		Message:  "Record not found",
	}

	ErrRecordExists = &AppError{
		Code:     ErrorCodeRecordExists,
		Severity: SeverityInfo,
		Message:  "Record already exists",
	}

	ErrForeignKeyViolation = &AppError{
		Code:     ErrorCodeForeignKeyViolation,
		Severity: SeverityError,
		Message:  "Foreign key constraint violation",
	}

	// Validation errors
	ErrInvalidInput = &AppError{
		Code:     ErrorCodeInvalidInput,
		Severity: SeverityWarn,
		Message:  "Invalid input",
	}

	ErrMissingRequired = &AppError{
		Code:     ErrorCodeMissingRequired,
		Severity: SeverityWarn,
		Message:  "Missing required field",
	}

	ErrInvalidFormat = &AppError{
		Code:     ErrorCodeInvalidFormat,
		Severity: SeverityWarn,
		Message:  "Invalid format",
	}

	ErrValidationFailed = &AppError{
		Code:     ErrorCodeValidationFailed,
		Severity: SeverityWarn,
		Message:  "Validation failed",
	}

	// Authentication errors
	ErrUnauthorized = &AppError{
		Code:     ErrorCodeUnauthorized,
		Severity: SeverityWarn,
		Message:  "Unauthorized",
	}

	ErrForbidden = &AppError{
		Code:     ErrorCodeForbidden,
		Severity: SeverityWarn,
		Message:  "Forbidden",
	}

	ErrInvalidCredentials = &AppError{
		Code:     ErrorCodeInvalidCredentials,
		Severity: SeverityWarn,
		Message:  "Invalid credentials",
	}

	ErrSessionExpired = &AppError{
		Code:     ErrorCodeSessionExpired,
		Severity: SeverityInfo,
		Message:  "Session expired",
	}

	// Service errors
	ErrServiceUnavailable = &AppError{
		Code:     ErrorCodeServiceUnavailable,
		Severity: SeverityError,
		Message:  "Service unavailable",
	}

	ErrTimeout = &AppError{
		Code:     ErrorCodeTimeout,
		Severity: SeverityWarn,
		Message:  "Request timeout",
	}

	ErrRateLimit = &AppError{
		Code:     ErrorCodeRateLimit,
		Severity: SeverityWarn,
		Message:  "Rate limit exceeded",
	}

	ErrQuotaExceeded = &AppError{
		Code:     ErrorCodeQuotaExceeded,
		Severity: SeverityWarn,
		Message:  "Usage quota exceeded",
	}

	ErrInternalError = &AppError{
		Code:     ErrorCodeInternalError,
		Severity: SeverityError,
		Message:  "Internal server error",
	}

	ErrAssignmentNotFound = &AppError{
		Code:     ErrorCodeAssignmentNotFound,
		Severity: SeverityInfo,
		Message:  "Assignment not found",
	}

	ErrConflict = &AppError{
		Code:     ErrorCodeConflict,
		Severity: SeverityWarn,
		Message:  "Operation conflicts with current state",
	}

	// Question errors
	ErrTimestampMissingTimezone = &AppError{
		Code:     ErrorCodeTimestampMissingTimezone,
		Severity: SeverityError,
		Message:  "Timestamp missing timezone",
	}

	ErrNoQuestionsAvailable = &AppError{
		Code:     ErrorCodeNoQuestionsAvailable,
		Severity: SeverityInfo,
		Message:  "No questions available for assignment",
	}

	ErrQuestionAlreadyAnswered = &AppError{
		Code:     ErrorCodeQuestionAlreadyAnswered,
		Severity: SeverityInfo,
		Message:  "Question already answered",
	}

	ErrQuestionNotFound = &AppError{
		Code:     ErrorCodeQuestionNotFound,
		Severity: SeverityInfo,
		Message:  "Question not found",
	}

	ErrInvalidAnswerIndex = &AppError{
		Code:     ErrorCodeInvalidAnswerIndex,
		Severity: SeverityWarn,
		Message:  "Invalid answer index",
	}

	ErrGenerationLimitReached = &AppError{
		Code:     ErrorCodeGenerationLimitReached,
		Severity: SeverityInfo,
		Message:  "Daily generation limit reached",
	}

	// AI Service errors
	ErrAIProviderUnavailable = &AppError{
		Code:     ErrorCodeAIProviderUnavailable,
		Severity: SeverityError,
		Message:  "AI provider unavailable",
	}

	ErrAIRequestFailed = &AppError{
		Code:     ErrorCodeAIRequestFailed,
		Severity: SeverityError,
		Message:  "AI request failed",
	}

	ErrAIResponseInvalid = &AppError{
		Code:     ErrorCodeAIResponseInvalid,
		Severity: SeverityError,
		Message:  "AI response invalid",
	}

	ErrAIConfigInvalid = &AppError{
		Code:     ErrorCodeAIConfigInvalid,
		Severity: SeverityError,
		Message:  "AI configuration invalid",
	}

	// OAuth errors
	ErrOAuthCodeExpired = &AppError{
		Code:     ErrorCodeOAuthCodeExpired,
		Severity: SeverityWarn,
		Message:  "OAuth code expired",
	}

	ErrOAuthStateMismatch = &AppError{
		Code:     ErrorCodeOAuthStateMismatch,
		Severity: SeverityError,
		Message:  "OAuth state mismatch",
	}

	ErrOAuthProviderError = &AppError{
		Code:     ErrorCodeOAuthProviderError,
		Severity: SeverityError,
		Message:  "OAuth provider error",
	}
)

// NewAppError creates a new AppError with the specified code, severity, message and details
func NewAppError(code ErrorCode, severity SeverityLevel, message, details string) *AppError {
	return &AppError{
		Code:     code,
		Severity: severity,
		Message:  message,
		Details:  details,
	}
}

// NewAppErrorWithCause creates a new AppError with an underlying cause
func NewAppErrorWithCause(code ErrorCode, severity SeverityLevel, message, details string, cause error) *AppError {
	return &AppError{
		Code:     code,
		Severity: severity,
		Message:  message,
		Details:  details,
		Cause:    cause,
	}
}

// WrapError wraps an error with additional context, preserving AppError structure if possible
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}

	// If it's already an AppError, wrap it with additional details
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:     appErr.Code,
			Severity: appErr.Severity,
			Message:  context,
			Details:  appErr.Error(),
			Cause:    appErr,
		}
	}

	// For regular errors, create a generic internal error wrapper
	return &AppError{
		Code:     ErrorCodeInternalError,
		Severity: SeverityError,
		Message:  context,
		Details:  err.Error(),
		Cause:    err,
	}
}

// WrapErrorf wraps an error with formatted context, preserving AppError structure if possible
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	// Handle %w verb for error wrapping by using fmt.Errorf
	if strings.Contains(format, "%w") {
		// Use fmt.Errorf to properly handle %w verb
		wrappedErr := fmt.Errorf(format, args...)

		// If it's already an AppError, wrap it with the formatted message
		if appErr, ok := err.(*AppError); ok {
			return &AppError{
				Code:     appErr.Code,
				Severity: appErr.Severity,
				Message:  wrappedErr.Error(),
				Details:  appErr.Error(),
				Cause:    wrappedErr,
			}
		}

		// For regular errors, wrap with the formatted error
		return &AppError{
			Code:     ErrorCodeInternalError,
			Severity: SeverityError,
			Message:  wrappedErr.Error(),
			Details:  err.Error(),
			Cause:    wrappedErr,
		}
	}

	// If it's already an AppError, wrap it with additional details
	if appErr, ok := err.(*AppError); ok {
		context := fmt.Sprintf(format, args...)
		return &AppError{
			Code:     appErr.Code,
			Severity: appErr.Severity,
			Message:  context,
			Details:  appErr.Error(),
			Cause:    appErr,
		}
	}

	// For regular errors, create a generic internal error wrapper
	context := fmt.Sprintf(format, args...)
	return &AppError{
		Code:     ErrorCodeInternalError,
		Severity: SeverityError,
		Message:  context,
		Details:  err.Error(),
		Cause:    err,
	}
}

// ErrorWithContextf creates a new error with formatted context
func ErrorWithContextf(format string, args ...interface{}) error {
	return &AppError{
		Code:     ErrorCodeInternalError,
		Severity: SeverityError,
		Message:  fmt.Sprintf(format, args...),
	}
}

// IsError checks if an error matches a specific AppError type
func IsError(err error, target *AppError) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == target.Code
	}
	return false
}

// AsError attempts to convert an error to an AppError
func AsError(err error, target **AppError) bool {
	if appErr, ok := err.(*AppError); ok {
		*target = appErr
		return true
	}
	return false
}

// GetErrorCode returns the error code from an error if it's an AppError, otherwise returns a default code
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrorCodeInternalError
}

// GetErrorSeverity returns the severity level from an error if it's an AppError, otherwise returns error
func GetErrorSeverity(err error) SeverityLevel {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Severity
	}
	return SeverityError
}

// IsRetryable determines if an error should be retried based on its type and severity
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		// Only retry certain types of errors that are likely transient
		switch appErr.Code {
		case ErrorCodeTimeout, ErrorCodeServiceUnavailable, ErrorCodeDatabaseConnection:
			return appErr.Severity != SeverityFatal
		}
	}
	return false
}

// GetErrorLocalizedMessage returns a localized message for the error
func GetErrorLocalizedMessage(err error, locale string) string {
	if appErr, ok := err.(*AppError); ok {
		return GetLocalizedMessageWithDetails(appErr.Code, ParseLocale(locale), appErr.Details)
	}
	return "An error occurred"
}

// ToJSON converts an AppError to a JSON-serializable structure for API responses
func (e *AppError) ToJSON() map[string]interface{} {
	result := map[string]interface{}{
		"code":     string(e.Code),
		"message":  e.Message,
		"severity": string(e.Severity),
		"error":    e.Message, // Include error field for backward compatibility
	}

	if e.Details != "" {
		result["details"] = e.Details
	}

	// Add retryable information
	result["retryable"] = IsRetryable(e)

	if e.Cause != nil {
		// Only include cause in debug mode or for certain error types
		switch e.Severity {
		case SeverityError, SeverityFatal:
			result["cause"] = e.Cause.Error()
		}
	}

	return result
}

// ToJSONWithLocale converts an AppError to a JSON-serializable structure with localized messages
func (e *AppError) ToJSONWithLocale(locale string) map[string]interface{} {
	result := e.ToJSON()
	// Replace the message with localized version and update error field too
	localizedMessage := GetLocalizedMessage(e.Code, ParseLocale(locale))
	result["message"] = localizedMessage
	result["error"] = localizedMessage // Keep error field in sync
	return result
}

// ContextKey represents a context key type for passing values through context
type ContextKey string

const (
	// UserIDKey is used to store user ID in context for usage tracking
	UserIDKey ContextKey = "userID"
	// APIKeyIDKey is used to store API key ID in context for usage tracking
	APIKeyIDKey ContextKey = "apiKeyID"
)

// GetUserIDFromContext extracts the user ID from context, returning 0 if not found
func GetUserIDFromContext(ctx context.Context) int {
	if userID, ok := ctx.Value(UserIDKey).(int); ok {
		return userID
	}
	return 0 // Default fallback
}

// GetAPIKeyIDFromContext extracts the API key ID from context, returning nil if not found
func GetAPIKeyIDFromContext(ctx context.Context) *int {
	if apiKeyID, ok := ctx.Value(APIKeyIDKey).(*int); ok {
		return apiKeyID
	}
	return nil // Default fallback
}

// WithUserID returns a new context with the user ID set
func WithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithAPIKeyID returns a new context with the API key ID set
func WithAPIKeyID(ctx context.Context, apiKeyID int) context.Context {
	return context.WithValue(ctx, APIKeyIDKey, &apiKeyID)
}
