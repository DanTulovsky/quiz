package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/serviceinterfaces"
	contextutils "quizapp/internal/utils"
)

// TranslationServiceInterface defines the interface for translation services
type TranslationServiceInterface = serviceinterfaces.TranslationService

// GoogleTranslationService handles translation requests using Google Translate API
type GoogleTranslationService struct {
	config     *config.Config
	httpClient *http.Client
}

// NewGoogleTranslationService creates a new Google translation service instance
func NewGoogleTranslationService(config *config.Config) *GoogleTranslationService {
	return &GoogleTranslationService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GoogleTranslateRequest represents the request format for Google Translate API
type GoogleTranslateRequest struct {
	Q      []string `json:"q"`
	Target string   `json:"target"`
	Source string   `json:"source,omitempty"`
	Format string   `json:"format"`
}

// GoogleTranslateResponse represents the response format from Google Translate API
type GoogleTranslateResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText         string `json:"translatedText"`
			DetectedSourceLanguage string `json:"detectedSourceLanguage"`
		} `json:"translations"`
	} `json:"data"`
}

// Translate translates text using the configured translation provider
func (s *GoogleTranslationService) Translate(ctx context.Context, req serviceinterfaces.TranslateRequest) (*serviceinterfaces.TranslateResponse, error) {
	if !s.config.Translation.Enabled {
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Translation service is disabled", "")
	}

	providerConfig, exists := s.config.Translation.Providers[s.config.Translation.DefaultProvider]
	if !exists {
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Translation provider not configured", "")
	}

	switch providerConfig.Code {
	case "google":
		return s.translateGoogle(ctx, req, providerConfig)
	default:
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Unsupported translation provider: "+providerConfig.Code, "")
	}
}

// translateGoogle translates text using Google Translate API
func (s *GoogleTranslationService) translateGoogle(ctx context.Context, req serviceinterfaces.TranslateRequest, providerConfig config.TranslationProviderConfig) (*serviceinterfaces.TranslateResponse, error) {
	if providerConfig.APIKey == "" {
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Google Translate API key not configured", "")
	}

	// Prepare request
	requestBody := GoogleTranslateRequest{
		Q:      []string{req.Text},
		Target: req.TargetLanguage,
		Format: "text",
	}

	if req.SourceLanguage != "" {
		requestBody.Source = req.SourceLanguage
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to marshal request")
	}

	// Build URL
	url := fmt.Sprintf("%s%s?key=%s", providerConfig.BaseURL, providerConfig.APIEndpoint, providerConfig.APIKey)

	// Make request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, contextutils.WrapError(err, "translation request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError,
			fmt.Sprintf("Google Translate API error: %d - %s", resp.StatusCode, string(body)), "")
	}

	// Parse response
	var googleResp GoogleTranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, contextutils.WrapError(err, "failed to decode response")
	}

	if len(googleResp.Data.Translations) == 0 {
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "No translation returned from Google Translate API", "")
	}

	translation := googleResp.Data.Translations[0]

	// Determine source language
	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = translation.DetectedSourceLanguage
		if sourceLanguage == "" {
			sourceLanguage = "auto"
		}
	}

	return &serviceinterfaces.TranslateResponse{
		TranslatedText: translation.TranslatedText,
		SourceLanguage: sourceLanguage,
		TargetLanguage: req.TargetLanguage,
	}, nil
}

// ValidateLanguageCode validates that a language code is properly formatted
func (s *GoogleTranslationService) ValidateLanguageCode(langCode string) error {
	if len(langCode) < 2 || len(langCode) > 10 {
		return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Language code must be 2-10 characters", "")
	}

	// Basic validation - should be alphanumeric with possible hyphens
	for _, char := range langCode {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
			return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Invalid language code format", "")
		}
	}

	return nil
}

// GetSupportedLanguages returns a list of supported target languages for translation
func (s *GoogleTranslationService) GetSupportedLanguages() []string {
	// Common languages supported by Google Translate API
	return []string{
		"af", "sq", "am", "ar", "hy", "az", "eu", "be", "bn", "bs", "bg", "ca", "ceb", "ny", "zh", "zh-CN", "zh-TW",
		"co", "hr", "cs", "da", "nl", "en", "eo", "et", "tl", "fi", "fr", "fy", "gl", "ka", "de", "el", "gu", "ht",
		"ha", "haw", "iw", "hi", "hmn", "hu", "is", "ig", "id", "ga", "it", "ja", "jw", "kn", "kk", "km", "ko", "ku",
		"ky", "lo", "la", "lv", "lt", "lb", "mk", "mg", "ms", "ml", "mt", "mi", "mr", "mn", "my", "ne", "no", "ps",
		"fa", "pl", "pt", "pa", "ro", "ru", "sm", "gd", "sr", "st", "sn", "sd", "si", "sk", "sl", "so", "es", "su",
		"sw", "sv", "tg", "ta", "te", "th", "tr", "uk", "ur", "uz", "vi", "cy", "xh", "yi", "yo", "zu",
	}
}

// NoopTranslationService is a no-operation implementation for testing and development
type NoopTranslationService struct{}

// NewNoopTranslationService creates a new noop translation service instance
func NewNoopTranslationService() *NoopTranslationService {
	return &NoopTranslationService{}
}

// Translate returns the original text unchanged (no-op)
func (s *NoopTranslationService) Translate(_ context.Context, req serviceinterfaces.TranslateRequest) (*serviceinterfaces.TranslateResponse, error) {
	return &serviceinterfaces.TranslateResponse{
		TranslatedText: req.Text,
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Confidence:     1.0,
	}, nil
}

// ValidateLanguageCode validates that a language code is properly formatted
func (s *NoopTranslationService) ValidateLanguageCode(langCode string) error {
	if len(langCode) < 2 || len(langCode) > 10 {
		return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Language code must be 2-10 characters", "")
	}

	// Basic validation - should be alphanumeric with possible hyphens
	for _, char := range langCode {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
			return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Invalid language code format", "")
		}
	}

	return nil
}

// GetSupportedLanguages returns a list of supported target languages for translation
func (s *NoopTranslationService) GetSupportedLanguages() []string {
	// Return a subset of common languages for testing
	return []string{
		"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh",
	}
}

// NewTranslationService creates a translation service based on configuration
// For testing environments, it returns a noop service if translation is disabled
// For production, it returns a Google translation service if properly configured
func NewTranslationService(config *config.Config) TranslationServiceInterface {
	if !config.Translation.Enabled {
		return NewNoopTranslationService()
	}

	providerConfig, exists := config.Translation.Providers[config.Translation.DefaultProvider]
	if !exists {
		// Fallback to noop if provider not configured
		return NewNoopTranslationService()
	}

	switch providerConfig.Code {
	case "google":
		return NewGoogleTranslationService(config)
	default:
		// Fallback to noop for unsupported providers
		return NewNoopTranslationService()
	}
}
