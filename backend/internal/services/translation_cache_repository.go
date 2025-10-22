package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
)

// TranslationCacheRepository defines the interface for translation cache operations
type TranslationCacheRepository interface {
	// GetCachedTranslation retrieves a cached translation if it exists and is not expired
	GetCachedTranslation(ctx context.Context, textHash, sourceLang, targetLang string) (*models.TranslationCache, error)

	// SaveTranslation stores a translation in the cache with a 30-day expiration
	SaveTranslation(ctx context.Context, textHash, originalText, sourceLang, targetLang, translatedText string) error

	// CleanupExpiredTranslations removes expired translation cache entries
	CleanupExpiredTranslations(ctx context.Context) (int64, error)
}

// TranslationCacheRepositoryImpl implements TranslationCacheRepository
type TranslationCacheRepositoryImpl struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewTranslationCacheRepository creates a new translation cache repository
func NewTranslationCacheRepository(db *sql.DB, logger *observability.Logger) TranslationCacheRepository {
	return &TranslationCacheRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// HashText generates a SHA-256 hash of the input text
func HashText(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}

// GetCachedTranslation retrieves a cached translation if it exists and is not expired
func (r *TranslationCacheRepositoryImpl) GetCachedTranslation(ctx context.Context, textHash, sourceLang, targetLang string) (result *models.TranslationCache, err error) {
	ctx, span := observability.TraceDatabaseFunction(ctx, "get_cached_translation",
		attribute.String("cache.text_hash", textHash),
		attribute.String("cache.source_language", sourceLang),
		attribute.String("cache.target_language", targetLang),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, text_hash, original_text, source_language, target_language, 
		       translated_text, created_at, expires_at
		FROM translation_cache
		WHERE text_hash = $1 
		  AND source_language = $2 
		  AND target_language = $3
		  AND expires_at > NOW()
	`

	cache := &models.TranslationCache{}
	err = r.db.QueryRowContext(ctx, query, textHash, sourceLang, targetLang).Scan(
		&cache.ID,
		&cache.TextHash,
		&cache.OriginalText,
		&cache.SourceLanguage,
		&cache.TargetLanguage,
		&cache.TranslatedText,
		&cache.CreatedAt,
		&cache.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		span.SetAttributes(attribute.Bool("cache.found", false))
		return nil, nil // Not found or expired
	}

	if err != nil {
		err = contextutils.WrapError(err, "failed to query translation cache")
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cache.found", true))
	return cache, nil
}

// SaveTranslation stores a translation in the cache with a 30-day expiration
func (r *TranslationCacheRepositoryImpl) SaveTranslation(ctx context.Context, textHash, originalText, sourceLang, targetLang, translatedText string) (err error) {
	ctx, span := observability.TraceDatabaseFunction(ctx, "save_translation_cache",
		attribute.String("cache.text_hash", textHash),
		attribute.String("cache.source_language", sourceLang),
		attribute.String("cache.target_language", targetLang),
		attribute.Int("cache.original_text_length", len(originalText)),
		attribute.Int("cache.translated_text_length", len(translatedText)),
	)
	defer observability.FinishSpan(span, &err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days from now

	query := `
		INSERT INTO translation_cache (text_hash, original_text, source_language, target_language, translated_text, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (text_hash, source_language, target_language) 
		DO UPDATE SET 
			translated_text = EXCLUDED.translated_text,
			expires_at = EXCLUDED.expires_at,
			created_at = CURRENT_TIMESTAMP
	`

	_, err = r.db.ExecContext(ctx, query, textHash, originalText, sourceLang, targetLang, translatedText, expiresAt)
	if err != nil {
		err = contextutils.WrapError(err, "failed to save translation to cache")
		return err
	}

	span.SetAttributes(
		attribute.String("cache.expires_at", expiresAt.Format(time.RFC3339)),
	)

	return nil
}

// CleanupExpiredTranslations removes expired translation cache entries
func (r *TranslationCacheRepositoryImpl) CleanupExpiredTranslations(ctx context.Context) (count int64, err error) {
	ctx, span := observability.TraceDatabaseFunction(ctx, "cleanup_expired_translations")
	defer observability.FinishSpan(span, &err)

	query := `DELETE FROM translation_cache WHERE expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		err = contextutils.WrapError(err, "failed to cleanup expired translations")
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		err = contextutils.WrapError(err, "failed to get rows affected")
		return 0, err
	}

	span.SetAttributes(attribute.Int64("cache.deleted_count", rowsAffected))
	r.logger.Info(ctx, "Cleaned up expired translation cache entries", map[string]interface{}{
		"deleted_count": rowsAffected,
	})

	return rowsAffected, nil
}
