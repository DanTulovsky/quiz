// Package serviceinterfaces defines service interfaces for dependency injection and testing.
package serviceinterfaces

import (
	"context"
)

// TranslateRequest represents a translation request
type TranslateRequest struct {
	Text           string `json:"text"`
	TargetLanguage string `json:"target_language"`
	SourceLanguage string `json:"source_language,omitempty"`
}

// TranslateResponse represents a translation response
type TranslateResponse struct {
	TranslatedText string  `json:"translated_text"`
	SourceLanguage string  `json:"source_language"`
	TargetLanguage string  `json:"target_language"`
	Confidence     float64 `json:"confidence,omitempty"`
}

// TranslationService defines the interface for translation services
type TranslationService interface {
	// Translate translates text using the configured translation provider
	Translate(ctx context.Context, req TranslateRequest) (*TranslateResponse, error)

	// ValidateLanguageCode validates that a language code is properly formatted
	ValidateLanguageCode(langCode string) error

	// GetSupportedLanguages returns a list of supported target languages for translation
	GetSupportedLanguages() []string
}
