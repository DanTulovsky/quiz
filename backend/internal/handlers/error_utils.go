package handlers

import (
	"fmt"
	"net/http"

	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
)

// StandardizeHTTPError creates consistent HTTP error responses with structured error information
func StandardizeHTTPError(c *gin.Context, statusCode int, message, details string) {
	// Map HTTP status code to appropriate error code
	var errorCode contextutils.ErrorCode
	var severity contextutils.SeverityLevel

	switch statusCode {
	case http.StatusBadRequest:
		errorCode = contextutils.ErrorCodeInvalidInput
		severity = contextutils.SeverityWarn
	case http.StatusUnauthorized:
		errorCode = contextutils.ErrorCodeUnauthorized
		severity = contextutils.SeverityWarn
	case http.StatusForbidden:
		errorCode = contextutils.ErrorCodeForbidden
		severity = contextutils.SeverityWarn
	case http.StatusNotFound:
		errorCode = contextutils.ErrorCodeRecordNotFound
		severity = contextutils.SeverityInfo
	case http.StatusConflict:
		errorCode = contextutils.ErrorCodeRecordExists
		severity = contextutils.SeverityInfo
	case http.StatusServiceUnavailable:
		errorCode = contextutils.ErrorCodeServiceUnavailable
		severity = contextutils.SeverityError
	default:
		errorCode = contextutils.ErrorCodeInternalError
		severity = contextutils.SeverityError
	}

	// Create an AppError with appropriate code
	appErr := contextutils.NewAppError(
		errorCode,
		severity,
		message,
		details,
	)

	// Send response with the original status code
	c.JSON(statusCode, appErr.ToJSON())
}

// StandardizeStructuredError sends a structured machine-readable error payload.
func StandardizeStructuredError(c *gin.Context, _ int, code, message, details string, retryable bool) {
	appErr := contextutils.NewAppError(
		contextutils.ErrorCode(code),
		contextutils.SeverityError,
		message,
		details,
	)

	StandardizeAppErrorWithRetry(c, appErr, retryable)
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

// StandardizeAppErrorWithRetry sends a structured error response with explicit retryable flag
func StandardizeAppErrorWithRetry(c *gin.Context, err *contextutils.AppError, retryable bool) {
	// Map error codes to HTTP status codes
	statusCode := mapErrorCodeToHTTPStatus(err.Code)

	// Convert error to JSON structure
	errorJSON := err.ToJSON()
	errorJSON["retryable"] = retryable

	c.JSON(statusCode, errorJSON)
}

// HandleValidationError handles input validation errors consistently
func HandleValidationError(c *gin.Context, field string, value interface{}, reason string) {
	appErr := contextutils.NewAppError(
		contextutils.ErrorCodeInvalidInput,
		contextutils.SeverityWarn,
		fmt.Sprintf("Invalid %s", field),
		fmt.Sprintf("Value '%v' is invalid: %s", value, reason),
	)

	StandardizeAppError(c, appErr)
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
