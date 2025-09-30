package contextutils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "error with details",
			appError: &AppError{
				Code:     ErrorCodeInvalidInput,
				Severity: SeverityError,
				Message:  "Invalid input",
				Details:  "Field 'email' is required",
			},
			expected: "INVALID_INPUT: Invalid input - Field 'email' is required",
		},
		{
			name: "error without details",
			appError: &AppError{
				Code:     ErrorCodeRecordNotFound,
				Severity: SeverityInfo,
				Message:  "Record not found",
			},
			expected: "RECORD_NOT_FOUND: Record not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appError.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	appErr := &AppError{
		Code:     ErrorCodeInternalError,
		Severity: SeverityError,
		Message:  "Internal error",
		Cause:    cause,
	}

	assert.Equal(t, cause, appErr.Unwrap())
}

func TestAppError_Is(t *testing.T) {
	err1 := &AppError{Code: ErrorCodeInvalidInput}
	err2 := &AppError{Code: ErrorCodeInvalidInput}
	err3 := &AppError{Code: ErrorCodeRecordNotFound}

	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(errors.New("regular error")))
}

func TestNewAppError(t *testing.T) {
	err := NewAppError(ErrorCodeInvalidInput, SeverityWarn, "Invalid input", "Field required")

	assert.Equal(t, ErrorCodeInvalidInput, err.Code)
	assert.Equal(t, SeverityWarn, err.Severity)
	assert.Equal(t, "Invalid input", err.Message)
	assert.Equal(t, "Field required", err.Details)
	assert.Nil(t, err.Cause)
}

func TestNewAppErrorWithCause(t *testing.T) {
	cause := errors.New("database error")
	err := NewAppErrorWithCause(ErrorCodeDatabaseConnection, SeverityError, "DB connection failed", "Connection timeout", cause)

	assert.Equal(t, ErrorCodeDatabaseConnection, err.Code)
	assert.Equal(t, SeverityError, err.Severity)
	assert.Equal(t, "DB connection failed", err.Message)
	assert.Equal(t, "Connection timeout", err.Details)
	assert.Equal(t, cause, err.Cause)
}

func TestWrapError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := WrapError(nil, "context")
		assert.Nil(t, result)
	})

	t.Run("AppError wrapping", func(t *testing.T) {
		original := &AppError{
			Code:     ErrorCodeRecordNotFound,
			Severity: SeverityInfo,
			Message:  "Record not found",
		}

		wrapped := WrapError(original, "additional context")

		appErr, ok := wrapped.(*AppError)
		assert.True(t, ok)
		assert.Equal(t, ErrorCodeRecordNotFound, appErr.Code)
		assert.Equal(t, SeverityInfo, appErr.Severity)
		assert.Equal(t, "additional context", appErr.Message)
		assert.Contains(t, appErr.Details, "Record not found")
		assert.Equal(t, original, appErr.Cause)
	})

	t.Run("regular error wrapping", func(t *testing.T) {
		original := errors.New("database error")
		wrapped := WrapError(original, "context")

		appErr, ok := wrapped.(*AppError)
		assert.True(t, ok)
		assert.Equal(t, ErrorCodeInternalError, appErr.Code)
		assert.Equal(t, SeverityError, appErr.Severity)
		assert.Equal(t, "context", appErr.Message)
		assert.Equal(t, "database error", appErr.Details)
		assert.Equal(t, original, appErr.Cause)
	})
}

func TestWrapErrorf(t *testing.T) {
	original := errors.New("database error")
	wrapped := WrapErrorf(original, "failed to process %s", "user123")

	appErr, ok := wrapped.(*AppError)
	assert.True(t, ok)
	assert.Equal(t, ErrorCodeInternalError, appErr.Code)
	assert.Equal(t, "failed to process user123", appErr.Message)
	assert.Equal(t, "database error", appErr.Details)
}

func TestErrorWithContextf(t *testing.T) {
	err := ErrorWithContextf("user not found: %s", "john")

	appErr, ok := err.(*AppError)
	assert.True(t, ok)
	assert.Equal(t, ErrorCodeInternalError, appErr.Code)
	assert.Equal(t, SeverityError, appErr.Severity)
	assert.Equal(t, "user not found: john", appErr.Message)
}

func TestIsError(t *testing.T) {
	err := &AppError{Code: ErrorCodeInvalidInput}
	target := &AppError{Code: ErrorCodeInvalidInput}

	assert.True(t, IsError(err, target))
	assert.False(t, IsError(err, &AppError{Code: ErrorCodeRecordNotFound}))
	assert.False(t, IsError(errors.New("regular error"), target))
}

func TestAsError(t *testing.T) {
	t.Run("successful conversion", func(t *testing.T) {
		original := &AppError{Code: ErrorCodeInvalidInput}
		var target *AppError

		assert.True(t, AsError(original, &target))
		assert.Equal(t, ErrorCodeInvalidInput, target.Code)
	})

	t.Run("failed conversion", func(t *testing.T) {
		original := errors.New("regular error")
		var target *AppError

		assert.False(t, AsError(original, &target))
		assert.Nil(t, target)
	})
}

func TestGetErrorCode(t *testing.T) {
	appErr := &AppError{Code: ErrorCodeInvalidInput}
	regularErr := errors.New("regular error")

	assert.Equal(t, ErrorCodeInvalidInput, GetErrorCode(appErr))
	assert.Equal(t, ErrorCodeInternalError, GetErrorCode(regularErr))
}

func TestGetErrorSeverity(t *testing.T) {
	appErr := &AppError{Severity: SeverityWarn}
	regularErr := errors.New("regular error")

	assert.Equal(t, SeverityWarn, GetErrorSeverity(appErr))
	assert.Equal(t, SeverityError, GetErrorSeverity(regularErr))
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable timeout error",
			err:      &AppError{Code: ErrorCodeTimeout, Severity: SeverityWarn},
			expected: true,
		},
		{
			name:     "retryable service unavailable",
			err:      &AppError{Code: ErrorCodeServiceUnavailable, Severity: SeverityError},
			expected: true,
		},
		{
			name:     "retryable database connection",
			err:      &AppError{Code: ErrorCodeDatabaseConnection, Severity: SeverityError},
			expected: true,
		},
		{
			name:     "non-retryable validation error",
			err:      &AppError{Code: ErrorCodeInvalidInput, Severity: SeverityWarn},
			expected: false,
		},
		{
			name:     "fatal error",
			err:      &AppError{Code: ErrorCodeTimeout, Severity: SeverityFatal},
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestGetErrorLocalizedMessage(t *testing.T) {
	t.Run("AppError", func(t *testing.T) {
		err := &AppError{
			Code:     ErrorCodeInvalidInput,
			Severity: SeverityWarn,
			Message:  "Invalid input",
			Details:  "Field required",
		}

		// Test English (default)
		msg := GetErrorLocalizedMessage(err, "en")
		assert.Contains(t, msg, "Invalid input")

		// Test Spanish
		msg = GetErrorLocalizedMessage(err, "es")
		assert.Contains(t, msg, "Entrada inválida")
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		msg := GetErrorLocalizedMessage(err, "en")
		assert.Equal(t, "An error occurred", msg)
	})
}

func TestAppError_ToJSON(t *testing.T) {
	err := &AppError{
		Code:     ErrorCodeInvalidInput,
		Severity: SeverityWarn,
		Message:  "Invalid input",
		Details:  "Field required",
		Cause:    errors.New("underlying error"),
	}

	json := err.ToJSON()

	assert.Equal(t, "INVALID_INPUT", json["code"])
	assert.Equal(t, "Invalid input", json["message"])
	assert.Equal(t, "warn", json["severity"])
	assert.Equal(t, "Field required", json["details"])
	assert.Equal(t, false, json["retryable"]) // Invalid input is not retryable
	assert.NotContains(t, json, "cause")      // Cause only included for error/fatal severity
}

func TestAppError_ToJSONWithLocale(t *testing.T) {
	err := &AppError{
		Code:     ErrorCodeInvalidInput,
		Severity: SeverityWarn,
		Message:  "Invalid input",
		Details:  "Field required",
	}

	json := err.ToJSONWithLocale("es")

	assert.Equal(t, "INVALID_INPUT", json["code"])
	assert.Equal(t, "warn", json["severity"])
	// Message should be localized
	assert.Contains(t, json["message"], "Entrada inválida")
}

func TestParseLocale(t *testing.T) {
	tests := []struct {
		input    string
		expected Locale
	}{
		{"en", LocaleEnglish},
		{"en-US", LocaleEnglish},
		{"EN", LocaleEnglish},
		{"es", LocaleSpanish},
		{"es-MX", LocaleSpanish},
		{"fr-CA", LocaleFrench},
		{"", LocaleEnglish}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLocale(tt.input))
		})
	}
}
