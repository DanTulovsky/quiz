package services

import (
	"context"
	"os"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/serviceinterfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTranslationCacheRepository is a mock implementation of TranslationCacheRepository for testing
type MockTranslationCacheRepository struct{}

func (m *MockTranslationCacheRepository) GetCachedTranslation(_ context.Context, _, _, _ string) (*models.TranslationCache, error) {
	return nil, nil // Always return cache miss for unit tests
}

func (m *MockTranslationCacheRepository) SaveTranslation(_ context.Context, _, _, _, _, _ string) error {
	return nil // Always succeed for unit tests
}

func (m *MockTranslationCacheRepository) CleanupExpiredTranslations(_ context.Context) (int64, error) {
	return 0, nil
}

func TestNoopTranslationService_Translate(t *testing.T) {
	service := NewNoopTranslationService()

	tests := []struct {
		name     string
		request  serviceinterfaces.TranslateRequest
		expected serviceinterfaces.TranslateResponse
	}{
		{
			name: "simple translation",
			request: serviceinterfaces.TranslateRequest{
				Text:           "Hello world",
				TargetLanguage: "es",
				SourceLanguage: "en",
			},
			expected: serviceinterfaces.TranslateResponse{
				TranslatedText: "Hello world",
				SourceLanguage: "en",
				TargetLanguage: "es",
				Confidence:     1.0,
			},
		},
		{
			name: "empty source language",
			request: serviceinterfaces.TranslateRequest{
				Text:           "Test text",
				TargetLanguage: "fr",
			},
			expected: serviceinterfaces.TranslateResponse{
				TranslatedText: "Test text",
				SourceLanguage: "",
				TargetLanguage: "fr",
				Confidence:     1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Translate(context.Background(), tt.request)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, *result)
		})
	}
}

func TestNoopTranslationService_ValidateLanguageCode(t *testing.T) {
	service := NewNoopTranslationService()

	validCodes := []string{"en", "es", "fr", "de", "zh-CN", "pt-BR"}
	invalidCodes := []string{"", "a", "toolonglanguagecode", "invalid@code"}

	for _, code := range validCodes {
		t.Run("valid_"+code, func(t *testing.T) {
			err := service.ValidateLanguageCode(code)
			assert.NoError(t, err)
		})
	}

	for _, code := range invalidCodes {
		t.Run("invalid_"+code, func(t *testing.T) {
			err := service.ValidateLanguageCode(code)
			assert.Error(t, err)
		})
	}
}

func TestNoopTranslationService_GetSupportedLanguages(t *testing.T) {
	service := NewNoopTranslationService()

	languages := service.GetSupportedLanguages()
	assert.NotEmpty(t, languages)
	assert.Contains(t, languages, "en")
	assert.Contains(t, languages, "es")
	assert.Contains(t, languages, "fr")
}

func TestGoogleTranslationService_ValidateLanguageCode(t *testing.T) {
	service := NewGoogleTranslationService(&config.Config{}, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	validCodes := []string{"en", "es", "fr", "de", "zh-CN", "pt-BR"}
	invalidCodes := []string{"", "a", "toolonglanguagecode", "invalid@code"}

	for _, code := range validCodes {
		t.Run("valid_"+code, func(t *testing.T) {
			err := service.ValidateLanguageCode(code)
			assert.NoError(t, err)
		})
	}

	for _, code := range invalidCodes {
		t.Run("invalid_"+code, func(t *testing.T) {
			err := service.ValidateLanguageCode(code)
			assert.Error(t, err)
		})
	}
}

func TestGoogleTranslationService_GetSupportedLanguages(t *testing.T) {
	service := NewGoogleTranslationService(&config.Config{}, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	languages := service.GetSupportedLanguages()
	assert.NotEmpty(t, languages)
	assert.Contains(t, languages, "en")
	assert.Contains(t, languages, "es")
	assert.Contains(t, languages, "fr")
	assert.Contains(t, languages, "de")
}

func TestNewTranslationService(t *testing.T) {
	t.Run("disabled translation returns noop service", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				Enabled: false,
			},
		}

		service := NewTranslationService(cfg, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		assert.IsType(t, &NoopTranslationService{}, service)
	})

	t.Run("no provider configured returns noop service", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				Enabled:         true,
				DefaultProvider: "nonexistent",
				Providers:       make(map[string]config.TranslationProviderConfig),
			},
		}

		service := NewTranslationService(cfg, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		assert.IsType(t, &NoopTranslationService{}, service)
	})

	t.Run("google provider returns google service", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				Enabled:         true,
				DefaultProvider: "google",
				Providers: map[string]config.TranslationProviderConfig{
					"google": {
						Code: "google",
						Name: "Google Translate",
					},
				},
			},
		}

		service := NewTranslationService(cfg, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		assert.IsType(t, &GoogleTranslationService{}, service)
	})

	t.Run("unsupported provider returns noop service", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				Enabled:         true,
				DefaultProvider: "unsupported",
				Providers: map[string]config.TranslationProviderConfig{
					"unsupported": {
						Code: "unsupported",
						Name: "Unsupported Provider",
					},
				},
			},
		}

		service := NewTranslationService(cfg, &NoopUsageStatsService{}, &MockTranslationCacheRepository{}, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
		assert.IsType(t, &NoopTranslationService{}, service)
	})
}

func TestConfig_EnvironmentVariableOverride(t *testing.T) {
	// Test that environment variables properly override config values
	t.Run("TRANSLATION_PROVIDERS_GOOGLE_API_KEY environment variable override", func(t *testing.T) {
		// Set the environment variable
		testAPIKey := "test-google-api-key-12345"
		originalEnv := os.Getenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY")
		defer func() {
			// Restore original environment variable
			if originalEnv != "" {
				require.NoError(t, os.Setenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY", originalEnv))
			} else {
				require.NoError(t, os.Unsetenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY"))
			}
		}()

		require.NoError(t, os.Setenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY", testAPIKey))

		// Create a new config which should pick up the environment variable
		cfg, err := config.NewConfig()
		require.NoError(t, err)

		// Verify the API key was properly overridden
		googleProvider, exists := cfg.Translation.Providers["google"]
		require.True(t, exists, "Google provider should exist in config")
		assert.Equal(t, testAPIKey, googleProvider.APIKey, "API key should be overridden by environment variable")
	})

	t.Run("missing environment variable keeps empty API key", func(t *testing.T) {
		// Ensure environment variable is not set
		originalEnv := os.Getenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY")
		defer func() {
			// Restore original environment variable
			if originalEnv != "" {
				require.NoError(t, os.Setenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY", originalEnv))
			} else {
				require.NoError(t, os.Unsetenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY"))
			}
		}()

		require.NoError(t, os.Unsetenv("TRANSLATION_PROVIDERS_GOOGLE_API_KEY"))

		// Create a new config which should not have the API key set
		cfg, err := config.NewConfig()
		require.NoError(t, err)

		// Verify the API key is empty (as defined in config.yaml)
		googleProvider, exists := cfg.Translation.Providers["google"]
		require.True(t, exists, "Google provider should exist in config")
		assert.Empty(t, googleProvider.APIKey, "API key should be empty when environment variable is not set")
	})
}
