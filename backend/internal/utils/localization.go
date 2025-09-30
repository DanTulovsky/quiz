package contextutils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Locale represents a language locale (e.g., "en", "es", "fr")
type Locale string

const (
	// LocaleEnglish represents English language
	LocaleEnglish Locale = "en"
	// LocaleSpanish represents Spanish language
	LocaleSpanish Locale = "es"
	// LocaleFrench represents French language
	LocaleFrench Locale = "fr"
	// LocaleGerman represents German language
	LocaleGerman Locale = "de"
	// LocaleItalian represents Italian language
	LocaleItalian Locale = "it"
)

// LocalizedMessages contains localized error messages for different locales
type LocalizedMessages struct {
	messages map[ErrorCode]map[Locale]string
}

// NewLocalizedMessages creates a new instance of localized messages
func NewLocalizedMessages() *LocalizedMessages {
	return &LocalizedMessages{
		messages: make(map[ErrorCode]map[Locale]string),
	}
}

// AddMessage adds a localized message for a specific error code and locale
func (lm *LocalizedMessages) AddMessage(code ErrorCode, locale Locale, message string) {
	if lm.messages[code] == nil {
		lm.messages[code] = make(map[Locale]string)
	}
	lm.messages[code][locale] = message
}

// GetMessage returns the localized message for an error code and locale
func (lm *LocalizedMessages) GetMessage(code ErrorCode, locale Locale) string {
	// Try to get the message for the specific locale
	if localeMessages, exists := lm.messages[code]; exists {
		if message, exists := localeMessages[locale]; exists {
			return message
		}

		// Fallback to English if the specific locale doesn't have a message
		if message, exists := localeMessages[LocaleEnglish]; exists {
			return message
		}
	}

	// Fallback to a default message
	return getDefaultMessage(code)
}

// GetMessageWithDetails returns a localized message with additional details
func (lm *LocalizedMessages) GetMessageWithDetails(code ErrorCode, locale Locale, details string) string {
	message := lm.GetMessage(code, locale)
	if details != "" {
		return fmt.Sprintf("%s: %s", message, details)
	}
	return message
}

// getDefaultMessage returns a default English message for error codes
func getDefaultMessage(code ErrorCode) string {
	switch code {
	case ErrorCodeDatabaseConnection:
		return "Database connection failed"
	case ErrorCodeDatabaseQuery:
		return "Database query failed"
	case ErrorCodeDatabaseTransaction:
		return "Database transaction failed"
	case ErrorCodeRecordNotFound:
		return "Record not found"
	case ErrorCodeRecordExists:
		return "Record already exists"
	case ErrorCodeForeignKeyViolation:
		return "Foreign key constraint violation"
	case ErrorCodeInvalidInput:
		return "Invalid input"
	case ErrorCodeMissingRequired:
		return "Missing required field"
	case ErrorCodeInvalidFormat:
		return "Invalid format"
	case ErrorCodeValidationFailed:
		return "Validation failed"
	case ErrorCodeUnauthorized:
		return "Unauthorized access"
	case ErrorCodeForbidden:
		return "Access forbidden"
	case ErrorCodeInvalidCredentials:
		return "Invalid credentials"
	case ErrorCodeSessionExpired:
		return "Session expired"
	case ErrorCodeServiceUnavailable:
		return "Service temporarily unavailable"
	case ErrorCodeTimeout:
		return "Request timeout"
	case ErrorCodeRateLimit:
		return "Rate limit exceeded"
	case ErrorCodeInternalError:
		return "Internal server error"
	case ErrorCodeAssignmentNotFound:
		return "Assignment not found"
	case ErrorCodeTimestampMissingTimezone:
		return "Timestamp missing timezone"
	case ErrorCodeNoQuestionsAvailable:
		return "No questions available"
	case ErrorCodeQuestionAlreadyAnswered:
		return "Question already answered"
	case ErrorCodeQuestionNotFound:
		return "Question not found"
	case ErrorCodeInvalidAnswerIndex:
		return "Invalid answer index"
	case ErrorCodeAIProviderUnavailable:
		return "AI service unavailable"
	case ErrorCodeAIRequestFailed:
		return "AI request failed"
	case ErrorCodeAIResponseInvalid:
		return "AI response invalid"
	case ErrorCodeAIConfigInvalid:
		return "AI configuration invalid"
	case ErrorCodeOAuthCodeExpired:
		return "OAuth code expired"
	case ErrorCodeOAuthStateMismatch:
		return "OAuth state mismatch"
	case ErrorCodeOAuthProviderError:
		return "OAuth provider error"
	default:
		return "An error occurred"
	}
}

// LoadMessagesFromJSON loads localized messages from a JSON structure
func (lm *LocalizedMessages) LoadMessagesFromJSON(jsonData string) error {
	var data map[string]map[string]string
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return WrapError(err, "failed to parse localization JSON")
	}

	for codeStr, localeMessages := range data {
		code := ErrorCode(codeStr)
		for localeStr, message := range localeMessages {
			locale := Locale(localeStr)
			lm.AddMessage(code, locale, message)
		}
	}

	return nil
}

// GetSupportedLocales returns a list of supported locales
func (lm *LocalizedMessages) GetSupportedLocales() []Locale {
	locales := make(map[Locale]bool)

	for _, localeMessages := range lm.messages {
		for locale := range localeMessages {
			locales[locale] = true
		}
	}

	result := make([]Locale, 0, len(locales))
	for locale := range locales {
		result = append(result, locale)
	}

	return result
}

// ParseLocale parses a locale string (e.g., "en-US", "fr-CA") and returns the language part
func ParseLocale(localeStr string) Locale {
	// Handle locale formats like "en-US", "fr-CA", etc.
	parts := strings.Split(localeStr, "-")
	if len(parts) > 0 && parts[0] != "" {
		return Locale(strings.ToLower(parts[0]))
	}
	return LocaleEnglish // Default fallback
}

// Global instance of localized messages
var globalLocalizedMessages = NewLocalizedMessages()

// init loads default localized messages
func init() {
	// Load some basic localized messages
	globalLocalizedMessages.AddMessage(ErrorCodeInvalidInput, LocaleSpanish, "Entrada inválida")
	globalLocalizedMessages.AddMessage(ErrorCodeInvalidInput, LocaleFrench, "Entrée invalide")
	globalLocalizedMessages.AddMessage(ErrorCodeInvalidInput, LocaleGerman, "Ungültige Eingabe")

	globalLocalizedMessages.AddMessage(ErrorCodeRecordNotFound, LocaleSpanish, "Registro no encontrado")
	globalLocalizedMessages.AddMessage(ErrorCodeRecordNotFound, LocaleFrench, "Enregistrement non trouvé")
	globalLocalizedMessages.AddMessage(ErrorCodeRecordNotFound, LocaleGerman, "Datensatz nicht gefunden")

	globalLocalizedMessages.AddMessage(ErrorCodeUnauthorized, LocaleSpanish, "Acceso no autorizado")
	globalLocalizedMessages.AddMessage(ErrorCodeUnauthorized, LocaleFrench, "Accès non autorisé")
	globalLocalizedMessages.AddMessage(ErrorCodeUnauthorized, LocaleGerman, "Unbefugter Zugriff")

	globalLocalizedMessages.AddMessage(ErrorCodeInternalError, LocaleSpanish, "Error interno del servidor")
	globalLocalizedMessages.AddMessage(ErrorCodeInternalError, LocaleFrench, "Erreur interne du serveur")
	globalLocalizedMessages.AddMessage(ErrorCodeInternalError, LocaleGerman, "Interner Serverfehler")
}

// GetLocalizedMessage returns a localized error message using the global instance
func GetLocalizedMessage(code ErrorCode, locale Locale) string {
	return globalLocalizedMessages.GetMessage(code, locale)
}

// GetLocalizedMessageWithDetails returns a localized error message with details
func GetLocalizedMessageWithDetails(code ErrorCode, locale Locale, details string) string {
	return globalLocalizedMessages.GetMessageWithDetails(code, locale, details)
}

// SetGlobalLocalizedMessages sets the global localized messages instance
func SetGlobalLocalizedMessages(messages *LocalizedMessages) {
	globalLocalizedMessages = messages
}
