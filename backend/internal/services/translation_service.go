package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/serviceinterfaces"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
)

// TranslationServiceInterface defines the interface for translation services
type TranslationServiceInterface = serviceinterfaces.TranslationService

// GoogleTranslationService handles translation requests using Google Translate API
type GoogleTranslationService struct {
	config        *config.Config
	httpClient    *http.Client
	usageStatsSvc UsageStatsServiceInterface
	logger        *observability.Logger
}

// NewGoogleTranslationService creates a new Google translation service instance
func NewGoogleTranslationService(config *config.Config, usageStatsSvc UsageStatsServiceInterface, logger *observability.Logger) *GoogleTranslationService {
	return &GoogleTranslationService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		usageStatsSvc: usageStatsSvc,
		logger:        logger,
	}
}

// GoogleTranslateRequest represents the request format for Google Translate API
type GoogleTranslateRequest struct {
	Q      []string `json:"q"`
	Target string   `json:"target"`
	Source string   `json:"source,omitempty"`
	Format string   `json:"format"`
}

// normalizeLanguageCode converts language names to ISO codes for Google Translate API
func normalizeLanguageCode(lang string, languageLevels map[string]config.LanguageLevelConfig) string {
	// Check if it's a language name in our config
	for languageName, levelConfig := range languageLevels {
		if strings.EqualFold(languageName, lang) {
			return levelConfig.Code
		}
	}

	// If it's already a valid ISO code or unknown, return as-is
	return lang
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
func (s *GoogleTranslationService) Translate(ctx context.Context, req serviceinterfaces.TranslateRequest) (result *serviceinterfaces.TranslateResponse, err error) {
	ctx, span := observability.TraceTranslationFunction(ctx, "translate",
		attribute.String("translation.target_language", req.TargetLanguage),
		attribute.String("translation.source_language", req.SourceLanguage),
		attribute.Int("translation.text_length", len(req.Text)),
	)
	defer observability.FinishSpan(span, &err)

	if !s.config.Translation.Enabled {
		return nil, contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Translation service is disabled", "")
	}

	providerConfig, exists := s.config.Translation.Providers[s.config.Translation.DefaultProvider]
	if !exists {
		err = contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Translation provider not configured", "")
		return nil, err
	}

	switch providerConfig.Code {
	case "google":
		span.SetAttributes(attribute.String("translation.provider", providerConfig.Code))
		result, err = s.translateGoogle(ctx, req, providerConfig)
		return result, err
	default:
		err = contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Unsupported translation provider: "+providerConfig.Code, "")
		return nil, err
	}
}

// translateGoogle translates text using Google Translate API
func (s *GoogleTranslationService) translateGoogle(ctx context.Context, req serviceinterfaces.TranslateRequest, providerConfig config.TranslationProviderConfig) (result *serviceinterfaces.TranslateResponse, err error) {
	ctx, span := observability.TraceTranslationFunction(ctx, "translate_google",
		attribute.String("translation.provider", providerConfig.Code),
		attribute.String("translation.target_language", req.TargetLanguage),
		attribute.String("translation.source_language", req.SourceLanguage),
		attribute.Int("translation.text_length", len(req.Text)),
	)
	defer observability.FinishSpan(span, &err)

	// Check quota before making the request
	if err := s.usageStatsSvc.CheckQuota(ctx, providerConfig.Code, "translation", len(req.Text)); err != nil {
		return nil, err
	}

	if providerConfig.APIKey == "" {
		err = contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "Google Translate API key not configured", "")
		return nil, err
	}

	if req.SourceLanguage == "" || req.TargetLanguage == "" {
		err = contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Source and target language are required", "")
		return nil, err
	}

	if len(req.Text) == 0 {
		err = contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Text cannot be empty", "")
		return nil, err
	}

	if len(req.Text) > providerConfig.MaxTextLength {
		err = contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, fmt.Sprintf("Text cannot exceed %d characters", providerConfig.MaxTextLength), "")
		return nil, err
	}

	// Prepare request - normalize language codes for Google Translate API
	requestBody := GoogleTranslateRequest{
		Q:      []string{req.Text},
		Target: normalizeLanguageCode(req.TargetLanguage, s.config.LanguageLevels),
		Source: normalizeLanguageCode(req.SourceLanguage, s.config.LanguageLevels),
		Format: "text",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		err = contextutils.WrapError(err, "failed to marshal request")
		return nil, err
	}

	// Build URL
	url := fmt.Sprintf("%s%s?key=%s", providerConfig.BaseURL, providerConfig.APIEndpoint, providerConfig.APIKey)

	// Make request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = contextutils.WrapError(err, "failed to create request")
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq.WithContext(ctx))
	if err != nil {
		err = contextutils.WrapError(err, "translation request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError,
			fmt.Sprintf("Google Translate API error: %d - %s", resp.StatusCode, string(body)), "")
		return nil, err
	}

	// Parse response
	var googleResp GoogleTranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		err = contextutils.WrapError(err, "failed to decode response")
		return nil, err
	}

	if len(googleResp.Data.Translations) == 0 {
		err = contextutils.NewAppError(contextutils.ErrorCodeServiceUnavailable, contextutils.SeverityError, "No translation returned from Google Translate API", "")
		return nil, err
	}

	translation := googleResp.Data.Translations[0]

	result = &serviceinterfaces.TranslateResponse{
		TranslatedText: translation.TranslatedText,
		SourceLanguage: normalizeLanguageCode(req.SourceLanguage, s.config.LanguageLevels),
		TargetLanguage: normalizeLanguageCode(req.TargetLanguage, s.config.LanguageLevels),
	}

	// Record usage after successful translation
	if err := s.usageStatsSvc.RecordUsage(ctx, providerConfig.Code, "translation", len(req.Text), 1); err != nil {
		// Log the error but don't fail the translation request
		// The translation was successful, we just couldn't record the usage
		// This is a non-critical error that should be logged for monitoring
		s.logger.Warn(ctx, "Failed to record translation usage", map[string]interface{}{
			"service":    providerConfig.Code,
			"usage_type": "translation",
			"characters": len(req.Text),
			"requests":   1,
			"error":      err.Error(),
		})
	}

	return result, nil
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
func NewTranslationService(config *config.Config, usageStatsSvc UsageStatsServiceInterface, logger *observability.Logger) TranslationServiceInterface {
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
		return NewGoogleTranslationService(config, usageStatsSvc, logger)
	default:
		// Fallback to noop for unsupported providers
		return NewNoopTranslationService()
	}
}
