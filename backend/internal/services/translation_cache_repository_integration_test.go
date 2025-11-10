//go:build integration

package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TranslationCacheRepositoryTestSuite is a test suite for TranslationCacheRepository integration tests
type TranslationCacheRepositoryTestSuite struct {
	suite.Suite
	db     *sql.DB
	repo   TranslationCacheRepository
	logger *observability.Logger
}

// SetupSuite sets up the test suite
func (suite *TranslationCacheRepositoryTestSuite) SetupSuite() {
	suite.logger = observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	suite.db = SharedTestDBSetup(suite.T())
	suite.repo = NewTranslationCacheRepository(suite.db, suite.logger)
}

// TearDownSuite tears down the test suite
func (suite *TranslationCacheRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		_ = suite.db.Close()
	}
}

// TearDownTest cleans up after each test
func (suite *TranslationCacheRepositoryTestSuite) TearDownTest() {
	// Clean up translation cache entries created during the test
	_, _ = suite.db.Exec("DELETE FROM translation_cache")
}

// TestTranslationCacheRepositoryTestSuite runs the test suite
func TestTranslationCacheRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TranslationCacheRepositoryTestSuite))
}

// TestSaveAndGetTranslation tests saving and retrieving a translation from cache
func (suite *TranslationCacheRepositoryTestSuite) TestSaveAndGetTranslation() {
	ctx := context.Background()
	textHash := HashText("Hello world")
	originalText := "Hello world"
	sourceLang := "en"
	targetLang := "es"
	translatedText := "Hola mundo"

	// Save translation
	err := suite.repo.SaveTranslation(ctx, textHash, originalText, sourceLang, targetLang, translatedText)
	require.NoError(suite.T(), err)

	// Retrieve translation
	cached, err := suite.repo.GetCachedTranslation(ctx, textHash, sourceLang, targetLang)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cached)

	assert.Equal(suite.T(), textHash, cached.TextHash)
	assert.Equal(suite.T(), originalText, cached.OriginalText)
	assert.Equal(suite.T(), sourceLang, cached.SourceLanguage)
	assert.Equal(suite.T(), targetLang, cached.TargetLanguage)
	assert.Equal(suite.T(), translatedText, cached.TranslatedText)
	assert.False(suite.T(), cached.CreatedAt.IsZero())
	assert.False(suite.T(), cached.ExpiresAt.IsZero())
	assert.True(suite.T(), cached.ExpiresAt.After(cached.CreatedAt))
}

// TestGetCachedTranslation_NotFound tests retrieving a non-existent translation
func (suite *TranslationCacheRepositoryTestSuite) TestGetCachedTranslation_NotFound() {
	ctx := context.Background()
	textHash := HashText("Non-existent text")

	cached, err := suite.repo.GetCachedTranslation(ctx, textHash, "en", "es")
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), cached)
}

// TestGetCachedTranslation_Expired tests that expired translations are not returned
func (suite *TranslationCacheRepositoryTestSuite) TestGetCachedTranslation_Expired() {
	ctx := context.Background()
	textHash := HashText("Expired text")
	originalText := "Expired text"
	sourceLang := "en"
	targetLang := "fr"
	translatedText := "Texte expiré"

	// Manually insert an expired translation
	expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	_, err := suite.db.ExecContext(ctx, `
		INSERT INTO translation_cache (text_hash, original_text, source_language, target_language, translated_text, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, textHash, originalText, sourceLang, targetLang, translatedText, expiresAt)
	require.NoError(suite.T(), err)

	// Try to retrieve - should return nil because it's expired
	cached, err := suite.repo.GetCachedTranslation(ctx, textHash, sourceLang, targetLang)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), cached, "Expired translation should not be returned")
}

// TestSaveTranslation_UpdateExisting tests that saving an existing translation updates it
func (suite *TranslationCacheRepositoryTestSuite) TestSaveTranslation_UpdateExisting() {
	ctx := context.Background()
	textHash := HashText("Update test")
	originalText := "Update test"
	sourceLang := "en"
	targetLang := "de"
	translatedText1 := "Aktualisierung test"
	translatedText2 := "Aktualisierungstest"

	// Save first translation
	err := suite.repo.SaveTranslation(ctx, textHash, originalText, sourceLang, targetLang, translatedText1)
	require.NoError(suite.T(), err)

	// Save second translation with same hash and languages (should update)
	err = suite.repo.SaveTranslation(ctx, textHash, originalText, sourceLang, targetLang, translatedText2)
	require.NoError(suite.T(), err)

	// Retrieve and verify it has the updated translation
	cached, err := suite.repo.GetCachedTranslation(ctx, textHash, sourceLang, targetLang)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cached)
	assert.Equal(suite.T(), translatedText2, cached.TranslatedText)
}

// TestSaveTranslation_DifferentLanguagePairs tests caching same text for different language pairs
func (suite *TranslationCacheRepositoryTestSuite) TestSaveTranslation_DifferentLanguagePairs() {
	ctx := context.Background()
	textHash := HashText("Same text")
	originalText := "Same text"

	// Save for English -> Spanish
	err := suite.repo.SaveTranslation(ctx, textHash, originalText, "en", "es", "Mismo texto")
	require.NoError(suite.T(), err)

	// Save for English -> French
	err = suite.repo.SaveTranslation(ctx, textHash, originalText, "en", "fr", "Même texte")
	require.NoError(suite.T(), err)

	// Retrieve both and verify they're different
	cachedES, err := suite.repo.GetCachedTranslation(ctx, textHash, "en", "es")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cachedES)
	assert.Equal(suite.T(), "Mismo texto", cachedES.TranslatedText)

	cachedFR, err := suite.repo.GetCachedTranslation(ctx, textHash, "en", "fr")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cachedFR)
	assert.Equal(suite.T(), "Même texte", cachedFR.TranslatedText)
}

// TestCleanupExpiredTranslations tests removing expired translations
func (suite *TranslationCacheRepositoryTestSuite) TestCleanupExpiredTranslations() {
	ctx := context.Background()

	// Create some expired translations
	expiresAt := time.Now().Add(-2 * time.Hour)
	for i := 0; i < 3; i++ {
		textHash := HashText("Expired text " + string(rune('a'+i)))
		_, err := suite.db.ExecContext(ctx, `
			INSERT INTO translation_cache (text_hash, original_text, source_language, target_language, translated_text, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, textHash, "Original", "en", "es", "Traducido", expiresAt)
		require.NoError(suite.T(), err)
	}

	// Create some valid translations
	validTextHash := HashText("Valid text")
	err := suite.repo.SaveTranslation(ctx, validTextHash, "Valid text", "en", "es", "Texto válido")
	require.NoError(suite.T(), err)

	// Run cleanup
	count, err := suite.repo.CleanupExpiredTranslations(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), count, "Should have deleted 3 expired translations")

	// Verify the valid translation still exists
	cached, err := suite.repo.GetCachedTranslation(ctx, validTextHash, "en", "es")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cached)
	assert.Equal(suite.T(), "Texto válido", cached.TranslatedText)
}

// TestHashText tests the hash function
func (suite *TranslationCacheRepositoryTestSuite) TestHashText() {
	hash1 := HashText("Hello world")
	hash2 := HashText("Hello world")
	hash3 := HashText("Hello World") // Different capitalization

	// Same text should produce same hash
	assert.Equal(suite.T(), hash1, hash2)

	// Different text should produce different hash
	assert.NotEqual(suite.T(), hash1, hash3)

	// Hash should be 64 characters (SHA-256 hex)
	assert.Len(suite.T(), hash1, 64)
}

// TestCacheExpiration tests that translations expire after 30 days
func (suite *TranslationCacheRepositoryTestSuite) TestCacheExpiration() {
	ctx := context.Background()
	textHash := HashText("Test expiration")
	originalText := "Test expiration"
	sourceLang := "en"
	targetLang := "es"
	translatedText := "Prueba de caducidad"

	// Save translation
	err := suite.repo.SaveTranslation(ctx, textHash, originalText, sourceLang, targetLang, translatedText)
	require.NoError(suite.T(), err)

	// Retrieve and check expiration time
	cached, err := suite.repo.GetCachedTranslation(ctx, textHash, sourceLang, targetLang)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), cached)

	// Expiration should be approximately 30 days from now
	expectedExpiration := time.Now().Add(30 * 24 * time.Hour)
	diff := cached.ExpiresAt.Sub(expectedExpiration).Abs()
	assert.Less(suite.T(), diff, 1*time.Minute, "Expiration time should be approximately 30 days from now")
}
