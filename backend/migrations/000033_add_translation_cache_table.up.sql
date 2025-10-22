-- Add translation_cache table for caching translation results
CREATE TABLE IF NOT EXISTS translation_cache (
    id SERIAL PRIMARY KEY,
    text_hash VARCHAR(64) NOT NULL,
    original_text TEXT NOT NULL,
    source_language VARCHAR(10) NOT NULL,
    target_language VARCHAR(10) NOT NULL,
    translated_text TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,

    UNIQUE(text_hash, source_language, target_language)
);

-- Create indexes for efficient lookup and cleanup
CREATE INDEX IF NOT EXISTS idx_translation_cache_lookup
    ON translation_cache(text_hash, source_language, target_language);
CREATE INDEX IF NOT EXISTS idx_translation_cache_expires_at
    ON translation_cache(expires_at);

-- Add comments for documentation
COMMENT ON TABLE translation_cache IS 'Caches translation results to reduce API calls and improve response times';
COMMENT ON COLUMN translation_cache.text_hash IS 'SHA-256 hash of the original text for fast lookups';
COMMENT ON COLUMN translation_cache.original_text IS 'Full original text for debugging and reference';
COMMENT ON COLUMN translation_cache.source_language IS 'Source language code (e.g., en, es, fr)';
COMMENT ON COLUMN translation_cache.target_language IS 'Target language code (e.g., en, es, fr)';
COMMENT ON COLUMN translation_cache.translated_text IS 'Cached translation result';
COMMENT ON COLUMN translation_cache.expires_at IS 'Expiration timestamp (30 days from creation)';

