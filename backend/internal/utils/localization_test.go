package contextutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalizedMessages_AddMessage_GetMessage(t *testing.T) {
	lm := NewLocalizedMessages()

	// Test adding and retrieving messages
	lm.AddMessage(ErrorCodeInvalidInput, LocaleEnglish, "Invalid input")
	lm.AddMessage(ErrorCodeInvalidInput, LocaleSpanish, "Entrada inválida")

	// Test English message
	msg := lm.GetMessage(ErrorCodeInvalidInput, LocaleEnglish)
	assert.Equal(t, "Invalid input", msg)

	// Test Spanish message
	msg = lm.GetMessage(ErrorCodeInvalidInput, LocaleSpanish)
	assert.Equal(t, "Entrada inválida", msg)

	// Test fallback to English for unsupported locale
	msg = lm.GetMessage(ErrorCodeInvalidInput, LocaleFrench)
	assert.Equal(t, "Invalid input", msg)

	// Test unknown error code
	msg = lm.GetMessage(ErrorCode("UNKNOWN_ERROR"), LocaleEnglish)
	assert.Equal(t, "An error occurred", msg)
}

func TestLocalizedMessages_GetMessageWithDetails(t *testing.T) {
	lm := NewLocalizedMessages()
	lm.AddMessage(ErrorCodeRecordNotFound, LocaleEnglish, "Record not found")

	msg := lm.GetMessageWithDetails(ErrorCodeRecordNotFound, LocaleEnglish, "User with ID 123")
	assert.Equal(t, "Record not found: User with ID 123", msg)

	// Test without details
	msg = lm.GetMessageWithDetails(ErrorCodeRecordNotFound, LocaleEnglish, "")
	assert.Equal(t, "Record not found", msg)
}

func TestLocalizedMessages_LoadMessagesFromJSON(t *testing.T) {
	jsonData := `{
		"INVALID_INPUT": {
			"en": "Invalid input",
			"es": "Entrada inválida"
		},
		"RECORD_NOT_FOUND": {
			"en": "Record not found",
			"fr": "Enregistrement non trouvé"
		}
	}`

	lm := NewLocalizedMessages()
	err := lm.LoadMessagesFromJSON(jsonData)
	assert.NoError(t, err)

	// Test loaded messages
	msg := lm.GetMessage(ErrorCodeInvalidInput, LocaleEnglish)
	assert.Equal(t, "Invalid input", msg)

	msg = lm.GetMessage(ErrorCodeInvalidInput, LocaleSpanish)
	assert.Equal(t, "Entrada inválida", msg)

	msg = lm.GetMessage(ErrorCodeRecordNotFound, LocaleFrench)
	assert.Equal(t, "Enregistrement non trouvé", msg)
}

func TestLocalizedMessages_LoadMessagesFromJSON_InvalidJSON(t *testing.T) {
	lm := NewLocalizedMessages()
	err := lm.LoadMessagesFromJSON(`invalid json`)
	assert.Error(t, err)
}

func TestLocalizedMessages_GetSupportedLocales(t *testing.T) {
	lm := NewLocalizedMessages()
	lm.AddMessage(ErrorCodeInvalidInput, LocaleEnglish, "Invalid input")
	lm.AddMessage(ErrorCodeInvalidInput, LocaleSpanish, "Spanish message")
	lm.AddMessage(ErrorCodeRecordNotFound, LocaleFrench, "Record not found")

	locales := lm.GetSupportedLocales()
	assert.Len(t, locales, 3)
	assert.Contains(t, locales, LocaleEnglish)
	assert.Contains(t, locales, LocaleSpanish)
	assert.Contains(t, locales, LocaleFrench)
}

func TestGetDefaultMessage(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrorCodeInvalidInput, "Invalid input"},
		{ErrorCodeRecordNotFound, "Record not found"},
		{ErrorCodeUnauthorized, "Unauthorized access"},
		{ErrorCodeInternalError, "Internal server error"},
		{ErrorCode("UNKNOWN"), "An error occurred"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			msg := getDefaultMessage(tt.code)
			assert.Equal(t, tt.expected, msg)
		})
	}
}

func TestGlobalLocalizedMessages(t *testing.T) {
	// Test that global messages work
	msg := GetLocalizedMessage(ErrorCodeInvalidInput, LocaleEnglish)
	assert.Contains(t, msg, "Invalid input")

	msg = GetLocalizedMessage(ErrorCodeInvalidInput, LocaleSpanish)
	assert.Contains(t, msg, "Entrada inválida")

	// Test record not found in Spanish
	msg = GetLocalizedMessage(ErrorCodeRecordNotFound, LocaleSpanish)
	assert.Contains(t, msg, "Registro no encontrado")
}

func TestSetGlobalLocalizedMessages(t *testing.T) {
	// Create custom messages
	customMessages := NewLocalizedMessages()
	customMessages.AddMessage(ErrorCodeInvalidInput, LocaleEnglish, "Custom invalid input")

	// Set as global
	SetGlobalLocalizedMessages(customMessages)

	// Test that global messages are now custom
	msg := GetLocalizedMessage(ErrorCodeInvalidInput, "en")
	assert.Equal(t, "Custom invalid input", msg)
}
